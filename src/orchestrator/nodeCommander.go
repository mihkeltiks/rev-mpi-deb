package main

import (
	"github.com/ottmartens/cc-rev-db/command"

	"github.com/ottmartens/cc-rev-db/logger"
)

func handleRemotely(cmd *command.Command) {
	nodeId := cmd.NodeId

	node := registeredNodes[nodeId]

	if node == nil {
		logger.Warn("Node %d not found", nodeId)
		return
	}

	err := node.client.Call("RemoteCmdHandler.Handle", cmd, new(int))

	if err != nil {
		logger.Error("Error on dispatching command: %v", err)
		return
	}

	logger.Debug("Command dispatched successfully to node %d", node.id)
}
