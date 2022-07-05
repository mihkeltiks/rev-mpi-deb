package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ottmartens/cc-rev-db/logger"
)

type command struct {
	pid      int
	code     commandCode
	argument interface{}
}

func (c command) String() string {
	codeStr := map[commandCode]string{
		bpoint:        "breakpoint",
		singleStep:    "singleStep",
		cont:          "continue",
		restore:       "restore",
		print:         "print",
		help:          "help",
		printInternal: "printInternal",
	}[c.code]

	str := codeStr

	if c.argument != nil {
		str = fmt.Sprintf("%v,%v", str, c.argument)
	}
	if c.pid != 0 {
		str = fmt.Sprintf("%v,pid:%v", str, c.pid)
	}

	return fmt.Sprintf("{%v}", str)
}

type commandCode int

const (
	bpoint commandCode = iota
	singleStep
	cont
	restore
	print
	quit
	help
	printInternal
)

type cmdResult struct {
	err    error
	exited bool
}

func (cmd *command) isProgressCommand() bool {
	return cmd.code == singleStep || cmd.code == cont
}

// a command line prefixed with a pid number
func matchPidRegexp(input string, exp string) bool {

	fullExpr := fmt.Sprintf(`^\d+ %v$`, exp)

	return regexp.MustCompile(fullExpr).Match([]byte(input))
}

func parseCommandFromString(input string) (c *command) {

	if input == "help" {
		return &command{0, help, nil}
	}

	if input == "q" {
		return &command{0, quit, nil}
	}

	matchesPidRegexp := regexp.MustCompile(`^\d+ .+`).Match([]byte(input))

	if !matchesPidRegexp {
		logger.Warn("error parsing break command - no pid specified")
		return nil
	}

	pieces := strings.Split(input, " ")

	pid, _ := strconv.Atoi(pieces[0])

	switch {

	case matchPidRegexp(input, `b \d+`): // breakpoint

		lineNr, _ := strconv.Atoi(pieces[2])

		return &command{pid, bpoint, lineNr}

	case matchPidRegexp(input, "c"): // continue
		return &command{pid, cont, nil}

	case matchPidRegexp(input, "s"): // single step
		return &command{pid, singleStep, nil}

	case matchPidRegexp(input, `p [a-zA-Z_][a-zA-Z0-9_]*`): // print variable
		varName := strings.Split(input, " ")[2]

		return &command{pid, print, varName}

	case matchPidRegexp(input, `r [0-9]*`): // restore

		index := 0
		if len(pieces) > 2 {
			index, _ = strconv.Atoi(pieces[2])
		}

		return &command{pid, restore, index}

	case matchPidRegexp(input, `pd [a-zA-Z_][a-zA-Z0-9_]*`): // debug print
		varName := strings.Split(input, " ")[2]

		return &command{pid, printInternal, varName}

	default:
		return nil
	}
}

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
