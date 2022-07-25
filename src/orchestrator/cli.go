package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
)

func parseArgs() (numProcesses int, targetPath string) {
	args := os.Args

	if len(args) > 3 || len(args) < 2 {
		panicArgs()
	}

	numProcesses, err := strconv.Atoi(args[1])

	if err != nil || numProcesses < 1 {
		panicArgs()
	}

	targetPath = args[2]
	file, err := os.Stat(targetPath)
	must(err)
	if file.IsDir() {
		panicArgs()
	}

	filepath.EvalSymlinks(targetPath)

	return numProcesses, targetPath
}

func panicArgs() {
	logger.Error("usage: orchestrator <num_processes> <target_file>")
	os.Exit(2)
}

func printInstructions() {

	fmt.Print("\nAvailable commands:\n\n")

	fmt.Println("  <nid> b <lineNr> \tset breakpoint")
	fmt.Println("  <nid> s \t\tsingle-step forward")
	fmt.Println("  <nid> c \t\tcontinue execution")
	fmt.Println("  <nid> p <var>  \tprint a variable")
	fmt.Println("        cp  \t\tlist recorded checkpoints")
	fmt.Println("        r <checkpoint id>  \trollback to checkpoint")

	fmt.Println("        q  \t\tquit")
	fmt.Println("     help  \t\tshow this again")
	fmt.Println()
	fmt.Printf("  nid (node id) in %v\n", registeredNodes.ids())
	fmt.Println()
}

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

	return text
}

func printPrompt() {
	fmt.Printf("insert command > ")
}

// a command line prefixed with a pid number
func matchPidRegexp(input string, exp string) bool {

	fullExpr := fmt.Sprintf(`^\d+ %v$`, exp)

	return regexp.MustCompile(fullExpr).Match([]byte(input))
}

func parseCommandFromString(input string) (c *command.Command) {

	// Global commands (executed on orchestrator)

	if input == "help" {
		return &command.Command{Code: command.Help}
	}

	if input == "q" {
		return &command.Command{Code: command.Quit}
	}

	if input == "cp" { // list recorded checkpoints
		return &command.Command{Code: command.ListCheckpoints}
	}

	pieces := strings.Split(input, " ")

	matchesGlobalRestore := regexp.MustCompile("^r .+").Match([]byte(input))
	if matchesGlobalRestore { // rollback operation (across n>=1 nodes)
		checkpointId := pieces[1]
		return &command.Command{Code: command.GlobalRollback, Argument: checkpointId}
	}

	// Node-specific commands (relayed to designated node for execution)

	matchesPidRegexp := regexp.MustCompile(`^\d+ .+`).Match([]byte(input))

	if !matchesPidRegexp {
		logger.Warn("error parsing break command - no pid specified")
		return nil
	}

	pid, _ := strconv.Atoi(pieces[0])

	switch {

	case matchPidRegexp(input, `[b|B] \d+`): // breakpoint

		lineNr, _ := strconv.Atoi(pieces[2])

		return &command.Command{NodeId: pid, Code: command.Bpoint, Argument: lineNr}

	case matchPidRegexp(input, "[c|C]"): // continue
		return &command.Command{NodeId: pid, Code: command.Cont}

	case matchPidRegexp(input, "[s|S]"): // single step
		return &command.Command{NodeId: pid, Code: command.SingleStep}

	case matchPidRegexp(input, `[p|P] [a-zA-Z_][a-zA-Z0-9_]*`): // print variable
		identifier := strings.Split(input, " ")[2]

		return &command.Command{NodeId: pid, Code: command.Print, Argument: identifier}

	case matchPidRegexp(input, `[r|R] .+`): // restore checkpoint with supplied id
		checkpointId := pieces[2]

		return &command.Command{NodeId: pid, Code: command.Restore, Argument: checkpointId}

	case matchPidRegexp(input, `pd [a-zA-Z_][a-zA-Z0-9_]*`): // debug print
		varName := strings.Split(input, " ")[2]

		return &command.Command{NodeId: pid, Code: command.PrintInternal, Argument: varName}

	default:
		return nil
	}
}

func askForRollbackCommit() bool {
	var s string

	fmt.Printf("Commit rollback? (y/n): ")
	_, err := fmt.Scan(&s)
	if err != nil {
		panic(err)
	}

	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	if s == "y" || s == "yes" {
		return true
	}

	return false
}
