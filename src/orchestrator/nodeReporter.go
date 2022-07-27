package main

import (
	"time"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/orchestrator/checkpointmanager"
	"github.com/ottmartens/cc-rev-db/orchestrator/gui/websocket"
	"github.com/ottmartens/cc-rev-db/rpc"
)

type NodeReporter struct {
	checkpointRecordChan chan<- rpc.MPICallRecord
}

func startCheckpointRecordCollector(
	channel <-chan rpc.MPICallRecord,
) {
	for {
		callRecord := <-channel

		logger.Verbose("Node %v reported MPI call: %v", callRecord.NodeId, callRecord.OpName)

		checkpointmanager.RecordCheckpoint(callRecord)
		websocket.SendCheckpointUpdateMessage(checkpointmanager.GetCheckpointLog())
	}
}

func (r NodeReporter) Register(pid *int, reply *int) error {

	node := node{
		id:  len(registeredNodes),
		pid: *pid,
	}

	registeredNodes[node.id] = &node

	logger.Verbose("added process %d (pid: %d) to process list", node.id, node.pid)

	*reply = node.id
	return nil
}

func (r NodeReporter) CommandResult(cmd *command.Command, reply *int) error {
	nodeId := cmd.NodeId

	if len(cmd.Result.Error) > 0 {
		logger.Warn(
			"Node %v reported an error while executing command: %v",
			nodeId, cmd.Result.Error,
		)
	} else {
		logger.Info("Node %v successfully executed command %v", nodeId, cmd)
	}

	if cmd.Result.Exited {
		logger.Info("Node %v exited", nodeId)

		delete(registeredNodes, nodeId)

		if len(registeredNodes) == 0 {
			go func() {
				logger.Info("All nodes exited. Exiting in 5s")
				time.Sleep(time.Second * 5)
				quit()
			}()
		}
	}

	return nil
}

func (r NodeReporter) MPICall(callRecord rpc.MPICallRecord, reply *int) error {
	r.checkpointRecordChan <- callRecord
	return nil
}
