package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

func setBreakPoint(ctx *processContext, line int) (err error) {
	var interruptCode = []byte{0xCC} // code for breakpoint trap

	breakpointAddress, _, err := ctx.symTable.LineToPC(ctx.sourceFile, line)

	if err != nil {
		log.Default().Printf("cannot set breakpoint at line: %v", err)
		return err
	}

	file, line := getLineForPC(ctx.symTable, breakpointAddress)
	log.Default().Printf("setting breakpoint at file: %v, line: %d", file, line)

	// set breakpoint (insert interrup code at the first pc address at the line)
	syscall.PtracePokeData(ctx.pid, uintptr(breakpointAddress), interruptCode)

	return err
}

func continueExecution(ctx *processContext) {
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

func singleStep(ctx *processContext) {
	syscall.PtraceSingleStep(ctx.pid)
}

func logRegistersState(ctx *processContext) {
	var regs syscall.PtraceRegs
	syscall.PtraceGetRegs(ctx.pid, &regs)

	filename, line, fn := ctx.symTable.PCToLine(regs.Rip)

	var fName string
	if fn != nil {
		fName = fn.Name
	}

	log.Default().Printf("instruction pointer: %s (line %d in %s)\n", fName, line, filename)
}

func quitDebugger() {
	fmt.Println("👋 Exiting..")
	os.Exit(0)
}
