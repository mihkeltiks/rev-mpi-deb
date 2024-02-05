package command

import "fmt"

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
	ListCheckpoints
	GlobalRollback
	RestoreCRIU
	CheckpointCRIU
	// Node-specific commands - executed on designated node
	Bpoint
	SingleStep
	Cont
	Restore
	Print
	PrintInternal
)

func (c Command) String() string {
	codeStr := map[CommandCode]string{
		Bpoint:          "breakpoint",
		CheckpointCRIU:  "checkpointCRIU",
		SingleStep:      "single-step",
		Cont:            "continue",
		Restore:         "restore",
		RestoreCRIU:     "restoreCRIU",
		Connect:         "connect",
		Disconnect:      "disconnect",
		Reset:           "reset",
		Stop:            "stop",
		Attach:          "attach",
		Detach:          "detach",
		Kill:            "kill",
		Print:           "print",
		Help:            "help",
		PrintInternal:   "print-internal",
		ListCheckpoints: "list-checkpoints",
	}[c.Code]

	if c.Argument == nil {
		return fmt.Sprintf("{%v}", codeStr)
	} else {
		return fmt.Sprintf("{%v,%v}", codeStr, c.Argument)
	}
}

func (cmd *Command) IsForwardProgressCommand() bool {
	return cmd.Code == SingleStep || cmd.Code == Cont
}

func (cmd *Command) IsProgressCommand() bool {
	return cmd.IsForwardProgressCommand() || cmd.Code == Restore
}
