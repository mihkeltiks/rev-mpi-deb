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
	Err    error
	Exited bool
}

const (
	Bpoint CommandCode = iota
	SingleStep
	Cont
	Restore
	Print
	Quit
	Help
	PrintInternal
)

func (c Command) String() string {
	codeStr := map[CommandCode]string{
		Bpoint:        "breakpoint",
		SingleStep:    "single-step",
		Cont:          "continue",
		Restore:       "restore",
		Print:         "print",
		Help:          "help",
		PrintInternal: "print-internal",
	}[c.Code]

	if c.Argument == nil {
		return fmt.Sprintf("{%v}", codeStr)
	} else {
		return fmt.Sprintf("{%v,%v}", codeStr, c.Argument)
	}
}

func (cmd *Command) IsProgressCommand() bool {
	return cmd.Code == SingleStep || cmd.Code == Cont
}
