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
	if nodeId == -1 {
		return GlobalHandleRemotely(cmd)
	}

	node := registeredNodes.nodes[nodeId]

	if node == nil {
		err := fmt.Errorf("Node %d not found", nodeId)
		logger.Warn("%v", err)
		return err
	}
	if len(registeredNodes.nodes) > 0 && cmd.IsForwardProgressCommand() {
		logger.Verbose("Locking node %v,", node.id)
		SetNodePending(node.id)
	}

	err := node.client.Call("RemoteCmdHandler.Handle", cmd, new(int))

	if err != nil {
		logger.Error("Error dispatching command: %v", err)
		return err
	}

	logger.Debug("Command %v dispatched successfully to node %d", cmd, node.id)
	return nil
}

func GlobalHandleRemotely(cmd *command.Command) (err error) {
	logger.Info("Distributing global command")
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			logger.Verbose("Sending to %v,", node.id)

			newCmd := command.Command{NodeId: node.id, Code: cmd.Code, Argument: cmd.Argument, Result: cmd.Result}

			if len(registeredNodes.nodes) > 0 && cmd.IsForwardProgressCommand() {
				logger.Verbose("Locking node %v,", node.id)
				SetNodePending(node.id)
			}

			err := node.client.Call("RemoteCmdHandler.Handle", newCmd, new(int))
			logger.Verbose("Sent to%v,", node.id)
			if err != nil {
				logger.Error("Error dispatching command: %v", err)
				return err
			}
			logger.Verbose("Command %v dispatched successfully to node %d", cmd, node.id)

		}
	}

	return nil
}

func Reset() (err error) {
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			logger.Debug("Reseting %v", node.id)
			err = HandleRemotely(&command.Command{NodeId: node.id, Code: command.Reset})
			if err != nil {
				logger.Error("Failed to reset %d: %v", node.id, err)
				break
			}
		}
	}

	return nil

}

func Attach() (err error) {
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			logger.Debug("Attaching debugger to process on node %v", node.id)
			err = HandleRemotely(&command.Command{NodeId: node.id, Code: command.Attach})
			if err != nil {
				logger.Error("Failed to attach debugger %d: %v", node.id, err)
				break
			}
		}
	}

	return nil

}

func Stop() (err error) {
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			err = HandleRemotely(&command.Command{NodeId: node.id, Code: command.Stop})
			if err != nil {
				logger.Error("Failed to stop process %d: %v", node.id, err)
				break
			}
		}
	}
	return nil
}

func Detach() (err error) {
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			err = HandleRemotely(&command.Command{NodeId: node.id, Code: command.Detach})
			if err != nil {
				logger.Error("Failed to detach debugger %d: %v", node.id, err)
				break
			}
		}
	}

	return nil
}

func Kill() (err error) {
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			err = HandleRemotely(&command.Command{NodeId: node.id, Code: command.Kill})
			if err != nil {
				logger.Error("Failed to kill %d: %v", node.id, err)
				break
			}
		}
	}
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
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			logger.Debug("Stopping node %v", node.id)
			HandleRemotely(&command.Command{NodeId: node.id, Code: command.Quit})

			node.client = nil
		}
	}
}
