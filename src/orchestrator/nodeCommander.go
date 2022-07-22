package main

import (
	"fmt"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/orchestrator/checkpointmanager"

	"github.com/ottmartens/cc-rev-db/logger"
)

func handleRemotely(cmd *command.Command) error {
	nodeId := cmd.NodeId

	node := registeredNodes[nodeId]

	if node == nil {
		err := fmt.Errorf("Node %d not found", nodeId)
		logger.Warn("%v", err)
		return err
	}

	err := node.client.Call("RemoteCmdHandler.Handle", cmd, new(int))

	if err != nil {
		logger.Error("Error dispatching command: %v", err)
		return err
	}

	logger.Debug("Command %v dispatched successfully to node %d", cmd, node.id)
	return nil
}

func executeRollback(rollbackMap *checkpointmanager.RollbackMap) error {
	for nodeId, checkpoint := range *rollbackMap {
		err := handleRemotely(&command.Command{
			NodeId:   int(nodeId),
			Code:     command.Restore,
			Argument: checkpoint.Id,
		})

		if err != nil {
			logger.Error("Failed to execute rollback on node %d: %v", nodeId, err)
			return err
		}
	}

	return nil
}
