package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/rpc"
)

const DEBUGGER_PATH = "bin/debug"
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

	cmd := exec.Command(
		"mpirun",
		"-np",
		fmt.Sprintf("%d", numProcesses),
		DEBUGGER_PATH,
		targetPath,
		fmt.Sprintf("localhost:%d", ORCHESTRATOR_PORT),
	)

	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	must(err)

	go func() {
		cmd.Wait()

		if err != nil {
			logger.Error("mpi job exited with: %v", err)
			os.Exit(1)
		}
	}()

	// wait for nodes to finish startup sequence
	time.Sleep(time.Second)
	heartbeatAllNodes()
	time.Sleep(time.Second)
	heartbeatAllNodes()

	for {
		cmd := askForInput()

		if cmd.code == quit {
			logger.Info("🌊 exiting")
			os.Exit(0)
		}

		logger.Info("command: %v", cmd)
	}
}
