package main

import (
	"fmt"
	"net/url"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/rpc"
)

type node struct {
	id     int
	pid    int
	client *rpc.RPCClient
}

var registeredNodes []node = make([]node, 0)

type NodeReporter struct{}

func (r NodeReporter) Register(pid *int, reply *int) error {

	node := node{
		id:  len(registeredNodes),
		pid: *pid,
	}

	registeredNodes = append(registeredNodes, node)
	logger.Verbose("added process %d (pid: %d) to process list", node.id, node.pid)

	*reply = node.id
	return nil
}

func (n node) getConnection() *rpc.RPCClient {
	nodeAddress, _ := url.Parse(fmt.Sprintf("localhost:%d", 3500+n.id))

	return rpc.Client.Connect(nodeAddress)
}

func heartbeatAllNodes() {
	for _, node := range registeredNodes {

		if node.client == nil {
			node.client = node.getConnection()
		}

		node.client.Heartbeat()
	}
}
