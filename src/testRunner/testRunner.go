package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {
	RUN_COUNT := 10

	failCount := 0

	if len(os.Args) > 1 {
		RUN_COUNT, _ = strconv.Atoi(os.Args[1])
	}

	for i := 0; i < RUN_COUNT; i++ {
		fmt.Println("<< new run >>")
		cmd := exec.Command(
			"docker",
			"run",
			"--rm",
			"-i",
			"--cap-add=SYS_PTRACE",
			"--security-opt",
			"seccomp=unconfined",
			"mpi-debugger",
			"hello")

		stdin, err := cmd.StdinPipe()

		if err != nil {
			panic(err)
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Start()

		time.Sleep(time.Millisecond * 400)

		io.WriteString(stdin, "b 26\n")
		time.Sleep(time.Millisecond * 200)

		io.WriteString(stdin, "c\n")
		time.Sleep(time.Millisecond * 100)

		io.WriteString(stdin, "p global\n")
		time.Sleep(time.Millisecond * 100)

		io.WriteString(stdin, "r\n")
		time.Sleep(time.Millisecond * 300)

		io.WriteString(stdin, "p global\n")
		time.Sleep(time.Millisecond * 100)

		io.WriteString(stdin, "c\n")
		time.Sleep(time.Millisecond * 100)

		io.WriteString(stdin, "c\n")

		err = cmd.Wait()

		if err != nil {
			fmt.Printf("%vexit error: %v%v\n", "\033[31m", err, "\033[0m")
			failCount++
		} else {
			fmt.Println("exit code 0")
		}
	}

	fmt.Printf("failed %d times\n", failCount)

}
