package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/ottmartens/cc-rev-db/logger"
)

const SLEEP_MS = 300

var runInDocker = false

// Utility for automated testing of the node debugger
func main() {
	RUN_COUNT := 10

	failCount := 0

	if len(os.Args) > 1 {
		RUN_COUNT, _ = strconv.Atoi(os.Args[1])
	}

	if len(os.Args) > 2 {
		if os.Args[2] == "docker" {
			runInDocker = true
		}
	}

	if runInDocker {
		logger.Info("running in docker")
	} else {
		logger.Info("running natively")
	}

	for i := 0; i < RUN_COUNT; i++ {
		logger.Info("<< new run >>")

		var cmd *exec.Cmd

		if runInDocker {
			cmd = exec.Command(
				"docker",
				"run",
				"--rm",
				"-i",
				"--cap-add=SYS_PTRACE",
				"--security-opt",
				"seccomp=unconfined",
				"mpi-debugger",
				"hello")
		} else {
			cmd = exec.Command(
				"bin/node-debugger",
				"hello")
		}

		stdin, err := cmd.StdinPipe()

		if err != nil {
			panic(err)
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Start()

		wait()

		io.WriteString(stdin, "b 26\n")
		wait()

		io.WriteString(stdin, "c\n")
		wait()

		io.WriteString(stdin, "p global\n")
		wait()

		io.WriteString(stdin, "r\n")
		wait()

		io.WriteString(stdin, "p global\n")
		wait()

		io.WriteString(stdin, "c\n")
		wait()

		io.WriteString(stdin, "c\n")

		err = cmd.Wait()

		if err != nil {
			fmt.Printf("%vexit error: %v%v\n", "\033[31m", err, "\033[0m")
			failCount++
		} else {
			logger.Info("exit code 0")
		}
	}

	fmt.Printf("failed %d times\n", failCount)

}

func wait() {
	time.Sleep(time.Millisecond * SLEEP_MS)
}
