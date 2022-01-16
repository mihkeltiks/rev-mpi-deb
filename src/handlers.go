package main

import (
	"fmt"
	"os"
	"syscall"

	Logger "github.com/ottmartens/cc-rev-db/logger"
)

func setBreakPoint(ctx *processContext, line int) (err error) {
	var interruptCode = []byte{0xCC} // code for breakpoint trap

	breakpointAddress, err := ctx.dwarfData.lineToPC(ctx.sourceFile, line)
	// breakpointAddress, _, err := ctx.symTable.LineToPC(ctx.sourceFile, line)

	if err != nil {
		Logger.Info("cannot set breakpoint at line: %v", err)
		return err
	}

	// file, line := getLineForPC(ctx.symTable, breakpointAddress)
	Logger.Info("setting breakpoint at file: %v, line: %d", ctx.sourceFile, line)

	// store the replaced instruction in the process context
	// to swap it in later after breakpoint is hit
	originalInstruction := make([]byte, len(interruptCode))
	syscall.PtracePeekData(ctx.pid, uintptr(breakpointAddress), originalInstruction)

	(*ctx.bpointData)[line] = &bpointData{
		breakpointAddress,
		originalInstruction,
	}

	// set breakpoint (insert interrupt code at the first valid pc address at the line)
	syscall.PtracePokeData(ctx.pid, uintptr(breakpointAddress), interruptCode)

	return err
}

// restores the original instruction if the executable
// is currently caught at a breakpoint
func restoreCaughtBreakpoint(ctx *processContext) {
	_, line, file, _, _ := getCurrentLine(ctx, true)

	bpointData := (*ctx.bpointData)[line]

	if bpointData == nil {
		Logger.Info("Not currently caught at breakpoint: line: %d, file: %v", line, file)
		return
	}

	Logger.Info("Caught at a breakpoint: line: %d, file: %v", line, file)

	var regs syscall.PtraceRegs
	syscall.PtraceGetRegs(ctx.pid, &regs)

	// rewind RIP to the replaced instruction
	regs.Rip -= 1
	syscall.PtraceSetRegs(ctx.pid, &regs)

	// replace breakpoint with original instruction
	syscall.PtracePokeData(ctx.pid, uintptr(bpointData.address), bpointData.data)
}

func continueExecution(ctx *processContext) (exited bool) {
	var waitStatus syscall.WaitStatus

	for i := 0; i < 100; i++ {
		syscall.PtraceCont(ctx.pid, 0)

		syscall.Wait4(ctx.pid, &waitStatus, 0, nil)

		if waitStatus.Exited() {
			Logger.Info("The binary exited with code %v", waitStatus.ExitStatus())
			return true
		}

		if waitStatus.StopSignal() == syscall.SIGTRAP && waitStatus.TrapCause() != syscall.PTRACE_EVENT_CLONE {
			Logger.Info("binary hit trap, execution paused")
			return false
		}
		// else {
		// received a signal other than trap/a trap from clone event, continue and wait more
		// }
	}

	panic(fmt.Sprintf("stuck at wait with signal: %v", waitStatus.StopSignal()))
}

func singleStep(ctx *processContext) {
	syscall.PtraceSingleStep(ctx.pid)
}

func quitDebugger() {
	fmt.Println("👋 Exiting..")
	os.Exit(0)
}
