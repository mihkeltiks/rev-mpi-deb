package cli

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

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
