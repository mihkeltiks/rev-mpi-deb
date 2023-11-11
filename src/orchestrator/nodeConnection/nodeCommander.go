package nodeconnection

import (
	"errors"
	"fmt"
	"time"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/checkpointmanager"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

func HandleRemotely(cmd *command.Command) error {
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

func ExecutePendingRollback() (err error) {
	rollbackMap := checkpointmanager.GetPendingRollback()

	if rollbackMap == nil {
		err = errors.New("Pending rollback not found")
		logger.Warn("%v", err)
		return err
	}

	logger.Info("Executing distributed rollback on %v nodes", len(*rollbackMap))

	for nodeId, checkpoint := range *rollbackMap {
		err = HandleRemotely(&command.Command{
			NodeId:   int(nodeId),
			Code:     command.Restore,
			Argument: checkpoint.Id,
		})

		if err != nil {
			logger.Error("Failed to execute rollback on node %d: %v", nodeId, err)
			break
		}

		checkpointmanager.RemoveSubsequentCheckpoints(checkpoint)
	}

	time.Sleep(time.Second)

	if err == nil {
		logger.Info("Distributed rollback executed successfully")
	} else {
		logger.Error("Distributed rollback failed")
	}

	checkpointmanager.ResetPendingRollback()

	return err
}

func StopAllNodes() {
	for _, node := range registeredNodes {
		if node.client != nil {
			logger.Debug("Stopping node %v", node.id)
			HandleRemotely(&command.Command{NodeId: node.id, Code: command.Quit})

			node.client = nil
		}
	}
}
