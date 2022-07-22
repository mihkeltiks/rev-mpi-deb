package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/orchestrator/checkpointmanager"
	"github.com/ottmartens/cc-rev-db/rpc"
)

const NODE_DEBUGGER_PATH = "bin/node-debugger"
const ORCHESTRATOR_PORT = 3490

func main() {
	// logger.SetMaxLogLevel(logger.Levels.Verbose)

	numProcesses, targetPath := parseArgs()

	// start rpc server in separate goroutine
	go func() {
		rpc.InitializeServer(ORCHESTRATOR_PORT, func(register rpc.Registrator) {
			register(new(logger.LoggerServer))
			register(new(NodeReporter))
		})
	}()

	logger.Info("executing %v as an mpi job with %d processes", targetPath, numProcesses)

	mpiProcess := exec.Command(
		"mpirun",
		"-np",
		fmt.Sprintf("%d", numProcesses),
		NODE_DEBUGGER_PATH,
		targetPath,
		fmt.Sprintf("localhost:%d", ORCHESTRATOR_PORT),
	)

	mpiProcess.Stdout = os.Stdout
	mpiProcess.Stderr = os.Stderr

	err := mpiProcess.Start()
	must(err)

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

	defer stopAllNodes()

	time.Sleep(time.Second)

	printInstructions()

	for {
		cmd := askForInput()

		switch cmd.Code {
		case command.Quit:
			logger.Info("👋 exiting")
			return
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
