package checkpointmanager

import (
	"fmt"
	"strconv"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/rpc"
)

type NodeId int

type checkpointRecord struct {
	Id              string
	nodeId          NodeId
	NodeRank        *int
	OpName          string
	IsSend          bool
	CanBeRestored   bool
	parameters      map[string]string
	MatchingEventId *string
	matchingEvent   *checkpointRecord // for send events, a link to the corresponding message receive event, and vice versa
	Tag             *int              // The mpi message tag, if present
}

type CheckpointLog map[NodeId][]checkpointRecord

// Data structure for maintaining a list of recorded checkpoints by node
var checkpointLog = make(CheckpointLog)

func GetCheckpointLog() CheckpointLog {
	return checkpointLog
}

var nodeRanks = make(map[NodeId]*int)

func RecordCheckpoint(mpiRecord rpc.MPICallRecord) {
	nodeId := NodeId(mpiRecord.NodeId)
	opName := mpiRecord.OpName

	record := checkpointRecord{
		Id:            mpiRecord.Id,
		nodeId:        nodeId,
		OpName:        opName,
		IsSend:        SEND_EVENTS[opName],
		CanBeRestored: RESTORABLE_OPERATIONS[opName],
		parameters:    mpiRecord.Parameters,
	}

	if nodeRanks[nodeId] == nil {
		nodeRanks[nodeId] = tryEvaluateIntegerParam("rank", record)
	}
	record.NodeRank = nodeRanks[nodeId]

	record.Tag = tryEvaluateIntegerParam("tag", record)

	// Link the matching event from other party, if already recorded
	record.matchingEvent, record.MatchingEventId = findMatchingMessage(record)

	if checkpointLog[nodeId] == nil {
		checkpointLog[nodeId] = make([]checkpointRecord, 0)
	}

	checkpointLog[nodeId] = append(checkpointLog[nodeId], record)
}

func findCheckpointById(checkpointId string) *checkpointRecord {
	for _, nodeCheckpoints := range checkpointLog {
		for _, checkpoint := range nodeCheckpoints {
			if checkpoint.Id == checkpointId {
				return &checkpoint
			}
		}
	}
	return nil
}

func ListCheckpoints() {
	for nodeId, nodeCheckpoints := range checkpointLog {
		var str string

		for _, record := range nodeCheckpoints {
			str = fmt.Sprintf("%s{%s - %s}", str, record.OpName, record.Id)
			str = fmt.Sprintf("%s,", str)
		}

		logger.Info("Node %d checkpoints:", nodeId)
		logger.Info(str)
	}
}

// Retrieves a corresponding send event for receive events, and vice versa, if present
func findMatchingMessage(record checkpointRecord) (*checkpointRecord, *string) {
	var matchingMessage *checkpointRecord

	switch record.OpName {
	case MPI_OPS[OP_SEND]:
		matchingNode, _ := strconv.Atoi(record.parameters["dest"])

		matchingMessage = getFirstUnmatchedMessage(matchingNode, MPI_OPS[OP_RECV], record.Tag)

	case MPI_OPS[OP_RECV]:
		matchingNode, _ := strconv.Atoi(record.parameters["source"])

		matchingMessage = getFirstUnmatchedMessage(matchingNode, MPI_OPS[OP_SEND], record.Tag)
	}

	if matchingMessage != nil {
		logger.Verbose("Linking matching messages  %v:%v - %v:%v", record.nodeId, record.OpName, matchingMessage.nodeId, matchingMessage.OpName)
		return matchingMessage, &matchingMessage.Id
	}

	return nil, nil
}

// Finds the first message on a node with specified operation name
func getFirstUnmatchedMessage(nodeId int, opName string, tag *int) *checkpointRecord {
	nodeCheckpoints := checkpointLog[NodeId(nodeId)]
	if nodeCheckpoints == nil {
		return nil
	}

	for _, checkpoint := range nodeCheckpoints {
		if checkpoint.matchingEvent != nil {
			continue
		}
		if checkpoint.OpName == opName && tagsMatch(tag, checkpoint.Tag) {
			return &checkpoint
		}
	}
	return nil
}

func tagsMatch(tag1, tag2 *int) bool {
	// tag retrieval has failed, might be false positive
	if tag1 == nil || tag2 == nil {
		return true
	}

	// wildcard tag used
	if *tag1 == -1 || *tag2 == -1 {
		return true
	}

	// matching tags used
	return *tag1 == *tag2
}

func tryEvaluateIntegerParam(paramName string, record checkpointRecord) *int {
	paramStr := record.parameters[paramName]
	if len(paramStr) == 0 {
		return nil
	}

	if value, err := strconv.Atoi(paramStr); err == nil {
		return &value
	}

	return nil
}
