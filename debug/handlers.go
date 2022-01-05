package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

func setBreakPoint(ctx *processContext, line int) (err error) {
	var interruptCode = []byte{0xCC} // code for breakpoint trap

	breakpointAddress, err := ctx.dwarfData.lineToPC(ctx.sourceFile, line)
	// breakpointAddress, _, err := ctx.symTable.LineToPC(ctx.sourceFile, line)

	if err != nil {
		log.Default().Printf("cannot set breakpoint at line: %v", err)
		return err
	}

	// file, line := getLineForPC(ctx.symTable, breakpointAddress)
	log.Default().Printf("setting breakpoint at file: %v, line: %d", ctx.sourceFile, line)

	// store the replaced instruction in the process context
	// to swap it in later after breakpoint is hit
	originalInstruction := make([]byte, len(interruptCode))
	syscall.PtracePeekData(ctx.pid, uintptr(breakpointAddress), originalInstruction)

	log.Default().Printf("saving breakpoint data: %x, %v", breakpointAddress, originalInstruction)

	(*ctx.bpointData)[line] = &bpointData{
		breakpointAddress,
		originalInstruction,
	}

	// set breakpoint (insert interrup code at the first pc address at the line)
	syscall.PtracePokeData(ctx.pid, uintptr(breakpointAddress), interruptCode)

	return err
}

// restores the original instruction if the executable
// is currently caught at a breakpoint
func restoreCaughtBreakpoint(ctx *processContext) {
	line, _, _, _ := getCurrentLine(ctx)

	bpointData := (*ctx.bpointData)[line]

	if bpointData == nil {
		fmt.Printf("caughtAtBreakpoint false: %x, %v\n", bpointData, bpointData)
		return
	}

	fmt.Printf("caughtAtBreakpoint true: %x, %v\n", bpointData.address, bpointData.data)

	bpointData.address += 1

	fmt.Printf("fixed address: %x\n", bpointData.address)

	syscall.PtracePokeData(ctx.pid, uintptr(bpointData.address), bpointData.data)
}

func continueExecution(ctx *processContext) (exited bool) {
	var waitStatus syscall.WaitStatus

	i := 0
	for {

		fmt.Printf("waiting, ws is %v\n", waitStatus.StopSignal().String())

		syscall.PtraceCont(ctx.pid, 0)

		syscall.Wait4(ctx.pid, &waitStatus, 0, nil)

		if waitStatus.Exited() {
			log.Default().Printf("The binary exited with code %v", waitStatus.ExitStatus())
			return true
		}

		if waitStatus.StopSignal() == syscall.SIGTRAP && waitStatus.TrapCause() != syscall.PTRACE_EVENT_CLONE {
			log.Default().Println("hit breakpoint, binary execution paused")
			break
		} else {
			// received a signal other than trap/a trap from clone event, continue and wait more
		}
		i++

		if i > 10 {
			fmt.Println("wont wait anymore")
			break
		}
	}

	return false
}

func singleStep(ctx *processContext) {
	syscall.PtraceSingleStep(ctx.pid)
}

func quitDebugger() {
	fmt.Println("👋 Exiting..")
	os.Exit(0)
}
