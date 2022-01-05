package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type command struct {
	code   commandCode
	lineNr int
}

// type Weekday string

// const (
// 	Sunday    Weekday = "Sunday"
// 	Monday    Weekday = "Monday"
// 	Tuesday   Weekday = "Tuesday"
// 	Wednesday Weekday = "Wednesday"
// 	Thursday  Weekday = "Thursday"
// 	Friday    Weekday = "Friday"
// 	Saturday  Weekday = "Saturday"
// )

type commandCode int

const (
	bpoint commandCode = iota
	step
	cont
	quit
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
	case quit:
		quitDebugger()
	}

	return &cmdResult{err, exited}
}

func (cmd *command) isProgressCommand() bool {
	return cmd.code == step || cmd.code == cont
}

func (c command) String() string {
	commandStrings := map[commandCode]string{
		bpoint: "bpoint",
		step:   "step",
		cont:   "continue",
		quit:   "quit",
	}

	bpointString := ""
	if c.code == bpoint {
		bpointString = fmt.Sprintf(", line %d", c.lineNr)
	}

	return fmt.Sprintf("Command{%s%s} \n", commandStrings[c.code], bpointString)
}

func parseCommandFromString(input string) (c *command) {

	breakPointRegexp := regexp.MustCompile("^b \\d+$")

	switch {
	case breakPointRegexp.Match([]byte(input)):
		lineNr, _ := strconv.Atoi(strings.Split(input, " ")[1])

		return &command{code: bpoint, lineNr: lineNr}

	case input == "c":
		return &command{code: cont}

	case input == "s":
		return &command{code: step}

	case input == "q":
		return &command{code: quit}

	default:
		return nil
	}
}
