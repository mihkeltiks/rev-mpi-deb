package nodeconnection

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
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

var registeredNodes nodeMap = make(nodeMap)

func GetRegisteredIds() []int {
	nodeIds := make([]int, 0, len(registeredNodes))
	for nodeId := range registeredNodes {
		nodeIds = append(nodeIds, nodeId)
	}
	sort.Sort(sort.IntSlice(nodeIds))
	return nodeIds
}

func ConnectToAllNodes(desiredNodeCount int) {
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

func DisconnectAllNodes() {
	for _, node := range registeredNodes {
		if node.client != nil {
			node.client.Disconnect()
		}
	}
}

func GetRegisteredNodesLen() int {
	return len(registeredNodes)
}
