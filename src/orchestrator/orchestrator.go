package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/ottmartens/cc-rev-db/logger"
)

const DEBUGGER_PATH = "bin/debug"

const ORCHESTRATOR_PORT = 3500

func main() {
	logger.SetMaxLogLevel(logger.Levels.Debug)

	numProcesses, targetPath := parseArgs()

	initRPCServer(ORCHESTRATOR_PORT)

	logger.Info("executing %v as an mpi job with %d processes", targetPath, numProcesses)

	cmd := exec.Command(
		"mpirun",
		"-np",
		fmt.Sprintf("%d", numProcesses),
		DEBUGGER_PATH,
		targetPath,
		fmt.Sprintf("localhost:%d", ORCHESTRATOR_PORT),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	must(err)

	err = cmd.Wait()

	if err != nil {
		logger.Error("mpi job exited with: %v", err)
	}

	for {
		cmd := askForInput()

		if cmd.code == quit {
			logger.Info("🌊 exiting")
			os.Exit(0)
		}

		logger.Info("command: %v", cmd)
	}
}

func parseArgs() (numProcesses int, targetPath string) {
	if len(os.Args) > 3 {
		panicArgs()
	}

	numProcesses, err := strconv.Atoi(os.Args[1])

	if err != nil || numProcesses < 1 {
		panicArgs()
	}

	targetPath = os.Args[2]

	file, err := os.Stat(os.Args[2])
	must(err)
	if file.IsDir() {
		panicArgs()
	}

	filepath.EvalSymlinks(targetPath)

	return numProcesses, targetPath
}

func panicArgs() {
	fmt.Println("usage: orchestrator <num_processes> <target_file>")
	os.Exit(2)
}
