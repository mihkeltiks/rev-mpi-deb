package command

import (
	"fmt"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
)

type Command struct {
	NodeId   int
	Code     CommandCode
	Argument interface{}
	Result   *CommandResult
}

type CommandCode int

type CommandResult struct {
	Error  string
	Exited bool
}

const (
	Quit CommandCode = iota
	Help
	Detach
	Attach
	Stop
	Kill
	// Global commands - executed on orchestrator
	Connect
	Disconnect
	Reset
	Insert
	Retrieve
	RetrieveBreakpoints
	ChangeBreakpoints
	RemoveBreakpoints
	ListCheckpoints
	GlobalRollback
	GRestore
	Checkpoint
	// Node-specific commands - executed on designated node
	Bpoint
	Next
	SingleStep
	ReverseSingleStep
	Cont
	ReverseCont
	Restore
	Print
	PrintInternal
)

func (c Command) String() string {
	codeStr := map[CommandCode]string{
		Bpoint:            "breakpoint",
		Checkpoint:        "checkpoint",
		SingleStep:        "single-step",
		Next:              "next",
		ReverseSingleStep: "reverse-single-step",
		Cont:              "continue",
		ReverseCont:       "reverse-continue",
		GRestore:           "restore",
		Restore:           "restore",
		Connect:           "connect",
		Disconnect:        "disconnect",
		Reset:             "reset",
		Stop:              "stop",
		Attach:            "attach",
		Detach:            "detach",
		Kill:              "kill",
		Print:             "print",
		Help:              "help",
		PrintInternal:     "print-internal",
		ListCheckpoints:   "list-checkpoints",
	}[c.Code]

	if c.Argument == nil {
		return fmt.Sprintf("{%v}", codeStr)
	} else {
		return fmt.Sprintf("{%v,%v}", codeStr, c.Argument)
	}
}

func (cmd *Command) IsForwardProgressCommand() bool {
	return cmd.Code == SingleStep || cmd.Code == Cont || cmd.Code == Next
}

func (cmd *Command) IsProgressCommand() bool {
	return cmd.IsForwardProgressCommand() || cmd.Code == Restore
}

func (cmd *Command) Print() {
	logger.Verbose("%v", cmd.NodeId, cmd.Code, cmd.Argument, cmd.Result)
}
