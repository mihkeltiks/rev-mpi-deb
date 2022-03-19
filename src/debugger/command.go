package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type command struct {
	code     commandCode
	argument interface{}
}

func (c command) String() string {
	codeStr := map[commandCode]string{
		bpoint:        "breakpoint",
		singleStep:    "single-singleStep",
		cont:          "continue",
		restore:       "restore",
		print:         "print",
		help:          "help",
		printInternal: "printInternal",
	}[c.code]

	if c.argument == nil {
		return fmt.Sprintf("{%v}", codeStr)
	} else {
		return fmt.Sprintf("{%v,%v}", codeStr, c.argument)
	}
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

func parseCommandFromString(input string) (c *command) {

	breakPointRegexp := regexp.MustCompile(`^b \d+$`)
	printRegexp := regexp.MustCompile(`^p [a-zA-Z_][a-zA-Z0-9_]*$`)
	printInternalRegexp := regexp.MustCompile(`^pd [a-zA-Z_][a-zA-Z0-9_]*$`)

	restoreRegexp := regexp.MustCompile(`^r\s*[0-9]*$`)

	switch {
	case breakPointRegexp.Match([]byte(input)):
		lineNr, _ := strconv.Atoi(strings.Split(input, " ")[1])

		return &command{bpoint, lineNr}

	case input == "c":
		return &command{cont, nil}

	case input == "s":
		return &command{singleStep, nil}

	case printRegexp.Match([]byte(input)):
		varName := strings.Split(input, " ")[1]

		return &command{print, varName}

	case input == "q":
		return &command{quit, nil}

	case restoreRegexp.Match([]byte(input)):
		split := strings.Split(input, " ")

		index := 0
		if len(split) > 1 {
			index, _ = strconv.Atoi(split[1])
		}

		return &command{restore, index}

	case input == "help":
		return &command{help, nil}

	case printInternalRegexp.Match([]byte(input)):
		varName := strings.Split(input, " ")[1]

		return &command{printInternal, varName}

	default:
		return nil
	}
}
