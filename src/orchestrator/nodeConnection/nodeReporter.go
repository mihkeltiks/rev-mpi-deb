package nodeconnection

import (
	"sync"
	"time"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/checkpointmanager"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

type NodeReporter struct {
	mu                   sync.Mutex
	checkpointRecordChan chan<- rpc.MPICallRecord
	quit                 func()
}

func NewNodeReporter(checkpointRecordChan chan<- rpc.MPICallRecord, quit func()) *NodeReporter {
	return &NodeReporter{sync.Mutex{}, checkpointRecordChan, quit}
}

func (r *NodeReporter) Register(pid *int, reply *int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := len(registeredNodes.nodes)

	// Make sure new indexes are correct
	if len(registeredNodesSave.nodes) > 0 {
		for index, node := range registeredNodesSave.nodes {
			if node.pid == *pid {
				id = index
				break
			}
		}
	}

	node := node{
		id:         id,
		pid:        *pid,
		Breakpoint: -1,
		counter:    -1,
	}

	registeredNodes.nodes[node.id] = &node
	// logger.Verbose("added process %d (pid: %d) to process list", node.id, node.pid)

	*reply = node.id
	return nil
}

func Empty() {
	// logger.Verbose("EMPTYINGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGg")
	registeredNodes = NewNodeMapContainer()
}

func (r *NodeReporter) CommandResult(cmd *command.Command, reply *int) error {
	// registeredNodes.mu.Lock()
	// defer registeredNodes.mu.Unlock()
	nodeId := cmd.NodeId

	if len(cmd.Result.Error) > 0 {
		logger.Warn(
			"Node %v reported an error while executing command: %v",
			nodeId, cmd.Result.Error,
		)
	} else {
		if len(registeredNodes.nodes) > 0 && cmd.IsForwardProgressCommand() {
			SetNodeDone(nodeId)
		}
		// logger.Verbose("Node %v successfully executed command %v", nodeId, cmd)
	}

	if cmd.Result.Exited {
		logger.Info("Node %v exited", nodeId)

		delete(registeredNodes.nodes, nodeId)

		if len(registeredNodes.nodes) == 0 {
			go func() {
				logger.Info("All nodes exited. Exiting in 10s")
				time.Sleep(time.Second * 10)
				r.quit()
			}()
		}
	}

	return nil
}

func (r *NodeReporter) Breakpoint(info *command.Command, reply *int) error {
	SetNodeBreakpoint(info.NodeId, int(info.Code))
	return nil
}

func (r *NodeReporter) ReportCounter(info *command.Command, reply *int) error {
	// logger.Verbose("NODE %v", info.NodeId)
	// logger.Verbose("COUNTER %v", int(info.Argument.(int32)))
	SetNodeCounter(info.NodeId, int(info.Argument.(int32)))
	return nil
}

func (r *NodeReporter) ReportBreakpoints(breakpoints *[]int, nodeId *int) error {
	SetBreakpoints(*breakpoints)
	return nil
}

func (r *NodeReporter) Progress(cmd *command.Command, reply *int) error {
	checkpointmanager.RemoveCurrentCheckpointMarkersOnNode(checkpointmanager.NodeId(cmd.NodeId))
	return nil
}

func (r *NodeReporter) MPICall(callRecord rpc.MPICallRecord, reply *int) error {
	r.checkpointRecordChan <- callRecord
	return nil
}
