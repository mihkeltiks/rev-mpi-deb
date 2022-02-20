package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ottmartens/cc-rev-db/logger"
)

type command struct {
	code     commandCode
	argument interface{}
}

type commandCode int

const (
	bpoint commandCode = iota
	step
	cont
	print
	quit
	help
	printInternal
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
		err = setBreakPoint(ctx, ctx.sourceFile, cmd.argument.(int))
	case step:
		singleStep(ctx)
	case cont:
		exited = continueExecution(ctx)
	case print:
		printVariable(ctx, cmd.argument.(string))
	case quit:
		quitDebugger()
	case help:
		printInstructions()
	case printInternal:
		printInternalData(ctx, cmd.argument.(string))
	}

	if cmd.isProgressCommand() {
		for {
			bpoint := restoreCaughtBreakpoint(ctx)

			if bpoint == nil {
				break
			}

			if bpoint.isMPIBpoint {
				recordMPIOperation(ctx, bpoint)

				reinsertMPIBPoints(ctx, bpoint)
			} else {
				logger.Info("stack: %v", getStack(ctx))
			}

			exited = continueExecution(ctx)

			if exited || !bpoint.isMPIBpoint {
				break
			}
		}
	}

	return &cmdResult{err, exited}
}

func (cmd *command) isProgressCommand() bool {
	return cmd.code == step || cmd.code == cont
}

func parseCommandFromString(input string) (c *command) {

	breakPointRegexp := regexp.MustCompile(`^b \d+$`)
	printRegexp := regexp.MustCompile(`^p [a-zA-Z_][a-zA-Z0-9_]*$`)
	printInternalRegexp := regexp.MustCompile(`^pd [a-zA-Z_][a-zA-Z0-9_]*$`)

	switch {
	case breakPointRegexp.Match([]byte(input)):
		lineNr, _ := strconv.Atoi(strings.Split(input, " ")[1])

		return &command{bpoint, lineNr}

	case input == "c":
		return &command{cont, nil}

	case input == "s":
		return &command{step, nil}

	case printRegexp.Match([]byte(input)):
		varName := strings.Split(input, " ")[1]

		return &command{print, varName}

	case input == "q":
		return &command{quit, nil}

	case input == "help":
		return &command{help, nil}

	case printInternalRegexp.Match([]byte(input)):
		varName := strings.Split(input, " ")[1]

		return &command{printInternal, varName}

	default:
		return nil
	}
}
