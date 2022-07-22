package main

import (
	"fmt"
	"net/url"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/rpc"
)

type node struct {
	id             int
	pid            int
	client         *rpc.RPCClient
	pendingCommand *command.Command
}

func (n node) getConnection() *rpc.RPCClient {
	nodeAddress, _ := url.Parse(fmt.Sprintf("localhost:%d", 3500+n.id))

	return rpc.Connect(nodeAddress)
}

// keys - node ids
type nodeMap map[int]*node

func (n nodeMap) ids() []int {
	nodeIds := make([]int, 0, len(n))
	for nodeId := range n {
		nodeIds = append(nodeIds, nodeId)
	}
	return nodeIds
}

var registeredNodes nodeMap = make(nodeMap)

func connectToAllNodes(desiredNodeCount int) {
	for _, node := range registeredNodes {

		if node.client == nil {
			node.client = node.getConnection()
		}

		node.client.Heartbeat()
	}

	if desiredNodeCount == len(registeredNodes) {
		logger.Info("Connected to all nodes")
	} else {
		panic(fmt.Sprintf("%d nodes connected, want %d", len(registeredNodes), desiredNodeCount))
	}
}

func stopAllNodes() {
	for _, node := range registeredNodes {
		if node.client != nil {
			logger.Debug("Stopping node %v", node.id)

			node.client.Call("RemoteCommand.Quit", new(int), new(int))
			node.client = nil
		}
	}
}
