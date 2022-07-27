package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/orchestrator/checkpointmanager"
	"github.com/ottmartens/cc-rev-db/orchestrator/gui"
	"github.com/ottmartens/cc-rev-db/rpc"
)

var NODE_DEBUGGER_PATH = fmt.Sprintf("%s/node-debugger", getExecutableDir())

const ORCHESTRATOR_PORT = 3490

func main() {
	logger.SetMaxLogLevel(logger.Levels.Verbose)
	numProcesses, targetPath := parseArgs()

	// start goroutine for collecting checkpoint results
	checkpointRecordChan := make(chan rpc.MPICallRecord)
	go startCheckpointRecordCollector(checkpointRecordChan)

	// start rpc server in separate goroutine
	go func() {
		rpc.InitializeServer(ORCHESTRATOR_PORT, func(register rpc.Registrator) {
			register(new(logger.LoggerServer))
			register(&NodeReporter{checkpointRecordChan})
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
	must(err)

	defer quit()

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
	connectToAllNodes(numProcesses)

	printInstructions()

	for {
		cmd := askForInput()

		switch cmd.Code {
		case command.Quit:
			quit()
		case command.Help:
			printInstructions()
			break
		case command.ListCheckpoints:
			checkpointmanager.ListCheckpoints()
			break
		case command.GlobalRollback:
			handleRollbackSubmission(cmd)
		default:
			handleRemotely(cmd)
			time.Sleep(time.Second)
		}
	}
}

func handleRollbackSubmission(cmd *command.Command) {
	pendingRollback := checkpointmanager.SubmitForRollback(cmd)
	if pendingRollback == nil {
		return
	}

	logger.Info("Following checkpoints scheduled for rollback:")
	logger.Info("%v", pendingRollback)

	commit := askForRollbackCommit()

	if !commit {
		logger.Info("Aborting rollback")
	}

	logger.Info("Executing distributed rollback")
	err := executeRollback(pendingRollback)

	time.Sleep(time.Second)

	if err == nil {
		logger.Info("Distributed rollback executed successfully")
	} else {
		logger.Error("Distributed rollback failed")
	}
}

func quit() {
	stopAllNodes()
	gui.Stop()

	time.Sleep(time.Second)
	logger.Info("👋 exiting")
	time.Sleep(time.Second)
	os.Exit(0)
}
