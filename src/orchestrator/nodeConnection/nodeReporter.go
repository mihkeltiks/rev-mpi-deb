package nodeconnection

import (
	"time"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/checkpointmanager"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

type NodeReporter struct {
	checkpointRecordChan chan<- rpc.MPICallRecord
	quit                 func()
}

func NewNodeReporter(checkpointRecordChan chan<- rpc.MPICallRecord, quit func()) *NodeReporter {
	return &NodeReporter{checkpointRecordChan, quit}
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

func Empty() {
	registeredNodes = make(nodeMap)
}

func (r NodeReporter) CommandResult(cmd *command.Command, reply *int) error {
	nodeId := cmd.NodeId

	if len(cmd.Result.Error) > 0 {
		logger.Warn(
			"Node %v reported an error while executing command: %v",
			nodeId, cmd.Result.Error,
		)
	} else {
		logger.Verbose("Node %v successfully executed command %v", nodeId, cmd)
	}

	if cmd.Result.Exited {
		logger.Info("Node %v exited", nodeId)

		delete(registeredNodes, nodeId)

		if len(registeredNodes) == 0 {
			go func() {
				logger.Info("All nodes exited. Exiting in 10s")
				time.Sleep(time.Second * 10)
				r.quit()
			}()
		}
	}

	return nil
}

func (r NodeReporter) Progress(cmd *command.Command, reply *int) error {
	checkpointmanager.RemoveCurrentCheckpointMarkersOnNode(checkpointmanager.NodeId(cmd.NodeId))
	return nil
}

func (r NodeReporter) MPICall(callRecord rpc.MPICallRecord, reply *int) error {
	r.checkpointRecordChan <- callRecord
	return nil
}
