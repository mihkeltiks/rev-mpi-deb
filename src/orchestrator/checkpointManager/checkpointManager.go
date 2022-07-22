package checkpointmanager

import (
	"fmt"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/rpc"
)

type nodeId int

type checkpointRecord struct {
	Id            string
	nodeId        nodeId
	opNname       string
	isSend        bool
	canBeRestored bool
	parameters    map[string]string
	matchingEvent *checkpointRecord // for send events, a link to the corresponding message receive events, and vice versa
}

// Data structure for maintaining a list of recorded checkpoints by node
var checkpointLog = make(map[nodeId][]checkpointRecord)

func RecordCheckpoint(mpiRecord rpc.MPICallRecord) {
	nodeId := nodeId(mpiRecord.NodeId)

	opName := mpiRecord.OpName

	record := checkpointRecord{
		Id:            mpiRecord.Id,
		nodeId:        nodeId,
		opNname:       opName,
		isSend:        SEND_EVENTS[opName],
		canBeRestored: RESTORABLE_OPERATIONS[opName],
		parameters:    mpiRecord.Parameters,
	}

	// Link the matching event from other party, if already recorded
	record.matchingEvent = findMatchingMessage(record)

	if checkpointLog[nodeId] == nil {
		checkpointLog[nodeId] = make([]checkpointRecord, 0)
	}

	checkpointLog[nodeId] = append(checkpointLog[nodeId], record)
}

// Retrieves a corresponding send event for receive events, and vice versa, if present
func findMatchingMessage(record checkpointRecord) *checkpointRecord {

	return nil
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
			str = fmt.Sprintf("%s{%s - %s}", str, record.opNname, record.Id)
			str = fmt.Sprintf("%s,", str)
		}

		logger.Info("Node %d checkpoints:", nodeId)
		logger.Info(str)
	}
}
