package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/orchestrator/checkpointmanager"
	"github.com/ottmartens/cc-rev-db/orchestrator/cli"
	"github.com/ottmartens/cc-rev-db/orchestrator/gui"
	"github.com/ottmartens/cc-rev-db/orchestrator/gui/websocket"
	nodeconnection "github.com/ottmartens/cc-rev-db/orchestrator/nodeConnection"
	"github.com/ottmartens/cc-rev-db/rpc"
	"github.com/ottmartens/cc-rev-db/utils"
	"github.com/ottmartens/cc-rev-db/utils/command"
)

var NODE_DEBUGGER_PATH = fmt.Sprintf("%s/node-debugger", utils.GetExecutableDir())

const ORCHESTRATOR_PORT = 3490

func main() {
	// logger.SetMaxLogLevel(logger.Levels.Verbose)
	numProcesses, targetPath := cli.ParseArgs()

	// start goroutine for collecting checkpoint results
	checkpointRecordChan := make(chan rpc.MPICallRecord)
	go startCheckpointRecordCollector(checkpointRecordChan)

	// start rpc server in separate goroutine
	go func() {
		rpc.InitializeServer(ORCHESTRATOR_PORT, func(register rpc.Registrator) {
			register(new(logger.LoggerServer))
			register(nodeconnection.NewNodeReporter(checkpointRecordChan, quit))
		})
	}()

	logger.Info("executing %v as an mpi job with %d processes", targetPath, numProcesses)

	// Start the MPI job
	mpiProcess := exec.Command(
		"mpirun",
		"-np",
		fmt.Sprintf("%d", numProcesses),
		NODE_DEBUGGER_PATH,
		targetPath,
		fmt.Sprintf("localhost:%d", ORCHESTRATOR_PORT),
	)

	mpiProcess.Stdout = os.Stdout
	// mpiProcess.Stderr = os.Stderr

	err := mpiProcess.Start()
	utils.Must(err)

	defer quit()

	// start the graphical user interface
	// when running with docker, gui must be started on the host
	if !utils.IsRunningInContainer() {
		gui.Start()

		websocket.InitServer()
		websocket.WaitForClientConnection()
	}

	// asyncronously wait for the MPI job to finish
	go func() {
		mpiProcess.Wait()

		if err != nil {
			logger.Error("mpi job exited with: %v", err)
			os.Exit(1)
		}
	}()

	// wait for nodes to finish startup sequence
	time.Sleep(time.Second)
	nodeconnection.ConnectToAllNodes(numProcesses)

	time.Sleep(time.Second)
	nodeconnection.HandleRemotely(&command.Command{NodeId: 1, Code: command.Bpoint, Argument: 52})
	nodeconnection.HandleRemotely(&command.Command{NodeId: 0, Code: command.Bpoint, Argument: 52})
	nodeconnection.HandleRemotely(&command.Command{NodeId: 1, Code: command.Cont})
	time.Sleep(time.Second * 1)
	nodeconnection.HandleRemotely(&command.Command{NodeId: 0, Code: command.Cont})
	time.Sleep(time.Second)

	cli.PrintInstructions()

	for {
		cmd := cli.AskForInput()

		switch cmd.Code {
		case command.Quit:
			quit()
		case command.Help:
			cli.PrintInstructions()
			break
		case command.ListCheckpoints:
			checkpointmanager.ListCheckpoints()
			break
		case command.GlobalRollback:
			handleRollbackSubmission(cmd)
			break
		default:
			nodeconnection.HandleRemotely(cmd)
			time.Sleep(time.Second)
			break
		}
	}
}

func handleRollbackSubmission(cmd *command.Command) {
	pendingRollback := checkpointmanager.SubmitForRollback(cmd.Argument.(string))
	if pendingRollback == nil {
		return
	}

	logger.Info("Following checkpoints scheduled for rollback:")
	logger.Info("%v", pendingRollback)

	commit := cli.AskForRollbackCommit()

	if !commit {
		logger.Verbose("Cancelling pending rollback")
		checkpointmanager.ResetPendingRollback()
		return
	}

	nodeconnection.ExecutePendingRollback()
}

func startCheckpointRecordCollector(
	channel <-chan rpc.MPICallRecord,
) {
	for {
		callRecord := <-channel

		logger.Debug("Node %v reported MPI call: %v", callRecord.NodeId, callRecord.OpName)

		checkpointmanager.RecordCheckpoint(callRecord)
		websocket.SendCheckpointUpdateMessage(checkpointmanager.GetCheckpointLog())
	}
}

func quit() {
	nodeconnection.StopAllNodes()
	gui.Stop()

	time.Sleep(time.Second)
	logger.Info("👋 exiting")
	time.Sleep(time.Second)
	os.Exit(0)
}
