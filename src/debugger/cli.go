package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ottmartens/cc-rev-db/logger"
)

func askForInput() *command {
	printPrompt()

	userInput := getUserInputLine()

	command := parseCommandFromString(userInput)

	if command == nil {
		fmt.Println(`Invalid input. Type "help" to see available commands`)
		return askForInput()
	}

	return command
}

func getUserInputLine() string {

	reader := bufio.NewReader(os.Stdin)

	text, _ := reader.ReadString('\n')

	text = strings.Replace(text, "\n", "", 1)

	text = strings.ToLower(text)

	return text
}

func printPrompt() {
	fmt.Printf("insert command > ")
}

func printInstructions() {

	fmt.Print("\nAvailable commands:\n\n")

	fmt.Println("  b <lineNr> \t set breakpoint")
	fmt.Println("  s  \t\t single singleStep forward")
	fmt.Println("  c  \t\t continue execution")
	fmt.Println("  r <cp index> \t restore checkpoint")
	fmt.Println("  p <var>  \t print a variable")
	fmt.Println("  q  \t\t quit")
	fmt.Println("  help  \t show this again")
	fmt.Println()
}

// parse and validate command line arguments
func getValuesFromArgs() (targetFilePath string, checkpointMode CheckpointMode, orchestratorAddress *url.URL) {

	if len(os.Args) < 3 {
		panicArgs()
	}

	var err error

	switch os.Args[1] {
	case "hello":
		logger.Info("loading example mpi hello binary")
		targetFilePath, err = filepath.Abs("bin/targets/hello")
	default:
		targetFilePath, err = filepath.Abs(os.Args[1])
	}

	must(err)

	targetFilePath, err = filepath.EvalSymlinks(targetFilePath)

	must(err)

	if _, err := os.Stat(targetFilePath); errors.Is(err, os.ErrNotExist) {
		panic(err) // file does not exist
	}

	// if len(os.Args) == 3 && os.Args[2] == "fork" {
	// 	checkpointMode = forkMode
	// 	logger.Info("Checkpoint mode: fork")
	// } else {
	// 	checkpointMode = fileMode
	// 	logger.Info("Checkpoint mode: file")
	// }

	if os.Args[2] == "cli" {
		cliMode = true
	} else {
		orchestratorAddress, err = url.ParseRequestURI(os.Args[2])

		if err != nil {
			os.Stderr.WriteString(err.Error())
			panicArgs()
		}
	}

	return targetFilePath, fileMode, orchestratorAddress
}

func panicArgs() {
	fmt.Println("Usage:")
	fmt.Println("cli mode: debug <target binary> cli")
	fmt.Println("network mode: debug <target binary> <orchestrator address>")
	os.Exit(2)
}
