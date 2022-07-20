package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
)

func askForInput() *command.Command {
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
		printUsage()
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
			printUsage()
		}
	}

	return targetFilePath, fileMode, orchestratorAddress
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("cli mode: node-debugger <target binary> cli")
	fmt.Println("network mode: node-debugger <target binary> <orchestrator address>")
	os.Exit(2)
}

func parseCommandFromString(input string) (c *command.Command) {

	breakPointRegexp := regexp.MustCompile(`^b \d+$`)
	printRegexp := regexp.MustCompile(`^p [a-zA-Z_][a-zA-Z0-9_]*$`)
	printInternalRegexp := regexp.MustCompile(`^pd [a-zA-Z_][a-zA-Z0-9_]*$`)

	restoreRegexp := regexp.MustCompile(`^r\s*[0-9]*$`)

	switch {
	case breakPointRegexp.Match([]byte(input)):
		lineNr, _ := strconv.Atoi(strings.Split(input, " ")[1])

		return &command.Command{Code: command.Bpoint, Argument: lineNr}

	case input == "c":
		return &command.Command{Code: command.Cont, Argument: nil}

	case input == "s":
		return &command.Command{Code: command.SingleStep, Argument: nil}

	case printRegexp.Match([]byte(input)):
		varName := strings.Split(input, " ")[1]

		return &command.Command{Code: command.Print, Argument: varName}

	case input == "q":
		return &command.Command{Code: command.Quit, Argument: nil}

	case restoreRegexp.Match([]byte(input)):
		split := strings.Split(input, " ")

		index := 0
		if len(split) > 1 {
			index, _ = strconv.Atoi(split[1])
		}

		return &command.Command{Code: command.Restore, Argument: index}

	case input == "help":
		return &command.Command{Code: command.Help, Argument: nil}

	case printInternalRegexp.Match([]byte(input)):
		varName := strings.Split(input, " ")[1]

		return &command.Command{Code: command.PrintInternal, Argument: varName}

	default:
		return nil
	}
}
