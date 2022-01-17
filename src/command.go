package main

import (
	"regexp"
	"strconv"
	"strings"
)

type command struct {
	code    commandCode
	lineNr  int
	varName string
}

type commandCode int

const (
	bpoint commandCode = iota
	step
	cont
	print
	quit
	help
)

type cmdResult struct {
	err    error
	exited bool
}

func (cmd *command) handle(ctx *processContext) *cmdResult {
	var err error
	var exited bool

	switch cmd.code {
	case bpoint:
		err = setBreakPoint(ctx, cmd.lineNr)
	case step:
		singleStep(ctx)
	case cont:
		exited = continueExecution(ctx)
	case print:
		printVariable(ctx, cmd.varName)
	case quit:
		quitDebugger()
	case help:
		printInstructions()
	}

	return &cmdResult{err, exited}
}

func (cmd *command) isProgressCommand() bool {
	return cmd.code == step || cmd.code == cont
}

func parseCommandFromString(input string) (c *command) {

	breakPointRegexp := regexp.MustCompile(`^b \d+$`)
	printRegexp := regexp.MustCompile(`^p [a-zA-Z_][a-zA-Z0-9_]*$`)

	switch {
	case breakPointRegexp.Match([]byte(input)):
		lineNr, _ := strconv.Atoi(strings.Split(input, " ")[1])

		return &command{code: bpoint, lineNr: lineNr}

	case input == "c":
		return &command{code: cont}

	case input == "s":
		return &command{code: step}

	case printRegexp.Match([]byte(input)):
		varName := strings.Split(input, " ")[1]

		return &command{code: print, varName: varName}

	case input == "q":
		return &command{code: quit}

	case input == "help":
		return &command{code: help}

	default:
		return nil
	}
}
