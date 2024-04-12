package nodeconnection

import (
	"fmt"
	"net/url"
	"sort"
	"sync"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

type node struct {
	id             int
	pid            int
	client         *rpc.RPCClient
	pendingCommand *command.Command
	Breakpoint     int
	pending        bool
	counter        int
	breakpoints    []int
}

func (n node) getConnection() *rpc.RPCClient {
	nodeAddress, _ := url.Parse(fmt.Sprintf("localhost:%d", 3500+n.id))

	return rpc.Connect(nodeAddress)
}

// keys - node ids
type nodeMap map[int]*node

type NodeMapContainer struct {
	mu    sync.Mutex
	nodes nodeMap
}

func NewNodeMapContainer() *NodeMapContainer {
	return &NodeMapContainer{
		mu:    sync.Mutex{},
		nodes: make(nodeMap),
	}
}

var registeredNodes = NewNodeMapContainer()
var registeredNodesSave = NewNodeMapContainer()

func SaveRegisteredNodes() {
	for index, node := range registeredNodes.nodes {
		registeredNodesSave.nodes[index] = node
	}
}

func SetBreakpoints(bps []int) {
	registeredNodes.nodes[bps[0]].breakpoints = bps[1:]
}

func GetBreakpoints(NodeId int) []int {
	return registeredNodes.nodes[NodeId].breakpoints
}

func GetRegisteredIds() []int {
	nodeIds := make([]int, 0, len(registeredNodes.nodes))
	for nodeId := range registeredNodes.nodes {
		nodeIds = append(nodeIds, nodeId)
	}
	sort.Sort(sort.IntSlice(nodeIds))
	return nodeIds
}

func GetNodeBreakpoint(id int) int {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	fmt.Println(registeredNodes.nodes[id].Breakpoint)
	return registeredNodes.nodes[id].Breakpoint
}

func SetNodeBreakpoint(id int, line int) {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	// logger.Verbose("%v", registeredNodes.nodes)
	registeredNodes.nodes[id].Breakpoint = line
}

func SetNodeCounter(id int, counter int) {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	// logger.Verbose("%v", registeredNodes.nodes)
	registeredNodes.nodes[id].counter = counter
}

func GetNodeCounter(id int) int {
	return registeredNodes.nodes[id].counter
}

func GetAllNodeCounters() []int {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	counters := make([]int, len(registeredNodes.nodes))

	for index, node := range registeredNodes.nodes {
		counters[index] = node.counter
	}

	return counters
}

func ResetAllNodeCounters() {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()

	for index := range registeredNodes.nodes {
		registeredNodes.nodes[index].counter = -1
	}
}

func GetNodePending(id int) bool {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	if id == -1 {
		for _, node := range registeredNodes.nodes {
			if node.pending {
				return true
			}
		}
		return false
	}
	return registeredNodes.nodes[id].pending
}

func GetReadyNode() int {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()

	for _, node := range registeredNodes.nodes {
		if !node.pending {
			return node.id
		}
	}

	return -1
}

func GetNodesPending(ids []int) bool {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()

	for _, num := range ids {
		logger.Verbose("Checking node %v,", num)

		if registeredNodes.nodes[num].pending {
			logger.Verbose("node %v pending,", num)
			return true
		}
	}
	return false
}

func SetNodePending(id int) {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()

	if id == -1 {
		for _, node := range registeredNodes.nodes {
			node.pending = true
		}

	} else {
		registeredNodes.nodes[id].pending = true
	}
}

func SetNodesPending(ids []int) {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	for _, num := range ids {
		registeredNodes.nodes[num].pending = true
	}
}

func SetNodesDone(ids []int) {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	for _, num := range ids {
		registeredNodes.nodes[num].pending = false
	}
}

func SetNodeDone(id int) {
	registeredNodes.mu.Lock()
	defer registeredNodes.mu.Unlock()
	if id == -1 {
		for _, node := range registeredNodes.nodes {
			node.pending = false
		}

	} else {
		registeredNodes.nodes[id].pending = false
	}
}

func ConnectToAllNodes(desiredNodeCount int) {
	for _, node := range registeredNodes.nodes {

		if node.client == nil {
			node.client = node.getConnection()
		}

		node.client.Heartbeat()
	}

	if desiredNodeCount == len(registeredNodes.nodes) {
		logger.Info("Connected to all nodes")
	} else {
		panic(fmt.Sprintf("%d nodes connected, want %d", len(registeredNodes.nodes), desiredNodeCount))
	}
}

func DisconnectAllNodes() {
	for _, node := range registeredNodes.nodes {
		if node.client != nil {
			node.client.Disconnect()
		}
	}
}

func GetRegisteredNodesLen() int {
	return len(registeredNodes.nodes)
}
