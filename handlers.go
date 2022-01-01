package main

import (
	"fmt"
	"log"
	"syscall"
)

func setBreakPoint(ctx processContext, line int) {
	var interruptCode = []byte{0xCC} // code for breakpoint trap

	fmt.Printf("sourcefie: %s\n", ctx.sourceFile)
	breakpointAddress, _, err := ctx.symTable.LineToPC(ctx.sourceFile, line)

	if err != nil {
		panic(err)
	}

	file, line, _ := ctx.symTable.PCToLine(breakpointAddress)
	log.Default().Printf("setting breakpoint at file: %v, line: %d", file, line)

	// set breakpoint (insert interrup code at the first pc address at the line)
	syscall.PtracePokeData(ctx.pid, uintptr(breakpointAddress), interruptCode)
}

func continueExecution(ctx processContext) {
	var waitStatus syscall.WaitStatus

	for {
		syscall.PtraceCont(ctx.pid, 0)

		syscall.Wait4(ctx.pid, &waitStatus, 0, nil)

		if waitStatus.StopSignal() == syscall.SIGTRAP && waitStatus.TrapCause() != syscall.PTRACE_EVENT_CLONE {
			break
		} else {
			// received a signal other than trap/a trap from clone event, continue and wait more
		}
	}
}
