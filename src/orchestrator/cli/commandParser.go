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

	if input == "lcp" { // list recorded checkpoints
		return &command.Command{Code: command.ListCheckpoints}
	}

	if input == "cp" { // list recorded checkpoints
		return &command.Command{Code: command.Checkpoint}
	}

	if input == "restore" { // list recorded checkpoints
		return &command.Command{Code: command.Restore}
	}

	if input == "kill" { // list recorded checkpoints
		return &command.Command{Code: command.Kill}
	}

	if input == "attach" { // list recorded checkpoints
		return &command.Command{Code: command.Attach}
	}

	if input == "stop" { // list recorded checkpoints
		return &command.Command{Code: command.Stop}
	}

	if input == "detach" { // list recorded checkpoints
		return &command.Command{Code: command.Detach}
	}
	if input == "connect" { // list recorded checkpoints
		return &command.Command{Code: command.Connect}
	}
	if input == "disconnect" { // list recorded checkpoints
		return &command.Command{Code: command.Disconnect}
	}
	if input == "reset" { // list recorded checkpoints
		return &command.Command{Code: command.Reset}
	}

	pieces := strings.Split(input, " ")

	matchesRestore := regexp.MustCompile("restore .+").Match([]byte(input))
	if matchesRestore {
		checkpointId, _ := strconv.Atoi(pieces[1])
		return &command.Command{Code: command.Restore, Argument: checkpointId}
	}

	matchesGlobalRestore := regexp.MustCompile("^r .+").Match([]byte(input))
	if matchesGlobalRestore { // rollback operation (across n>=1 nodes)
		checkpointId := pieces[1]
		return &command.Command{Code: command.GlobalRollback, Argument: checkpointId}
	}

	// All node commands
	pid, _ := strconv.Atoi(pieces[0])

	matchesAllRegexp := regexp.MustCompile(`^all .+`).Match([]byte(input))

	if matchesAllRegexp { // rollback operation (across n>=1 nodes)
		switch {
		case matchAllRegexp(input, `[b|B] \d+`): // breakpoint
			lineNr, _ := strconv.Atoi(pieces[2])
			return &command.Command{NodeId: -1, Code: command.Bpoint, Argument: lineNr}
		case matchAllRegexp(input, "[c|C]"): // continue
			return &command.Command{NodeId: -1, Code: command.Cont}
		case matchAllRegexp(input, `rc`): // continue
			return &command.Command{NodeId: -1, Code: command.ReverseCont}
		case matchAllRegexp(input, `[n|N]`): // breakpoint
			return &command.Command{NodeId: -1, Code: command.Next}
		case matchAllRegexp(input, "[s|S]"): // single step
			return &command.Command{NodeId: -1, Code: command.SingleStep}
		case matchAllRegexp(input, "rs"): // single step
			return &command.Command{NodeId: -1, Code: command.ReverseSingleStep}
		case matchAllRegexp(input, `[p|P] [a-zA-Z_][a-zA-Z0-9_]*`): // print variable
			identifier := strings.Split(input, " ")[2]
			return &command.Command{NodeId: -1, Code: command.Print, Argument: identifier}
		default:
			return nil
		}
	}
	// Node-specific commands (relayed to designated node for execution)

	matchesPidRegexp := regexp.MustCompile(`^\d+ .+`).Match([]byte(input))

	if !matchesPidRegexp {
		logger.Warn("error parsing break command - no pid specified")
		return nil
	}

	switch {

	case matchPidRegexp(input, `[b|B] \d+`): // breakpoint
		lineNr, _ := strconv.Atoi(pieces[2])
		return &command.Command{NodeId: pid, Code: command.Bpoint, Argument: lineNr}

	case matchPidRegexp(input, "[c|C]"): // continue
		return &command.Command{NodeId: pid, Code: command.Cont}

	case matchPidRegexp(input, `rc`): // continue
		return &command.Command{NodeId: pid, Code: command.ReverseCont}

	case matchPidRegexp(input, "[s|S]"): // single step
		return &command.Command{NodeId: pid, Code: command.SingleStep}

	case matchPidRegexp(input, "[n|N]"): // single step
		return &command.Command{NodeId: pid, Code: command.Next}

	case matchPidRegexp(input, "rs"): // single step
		return &command.Command{NodeId: pid, Code: command.ReverseSingleStep}

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
