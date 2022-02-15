package main

import (
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
)

type breakpointData map[uint64]*bpointData

type bpointData struct {
	address             uint64     // address of the instruction
	originalInstruction []byte     // actual contents of the instruction at address
	function            *dwarfFunc // the pointer to the function the breakpoint was inserted at
	isMPIBpoint         bool
}

func (b breakpointData) New() breakpointData {
	return make(map[uint64]*bpointData)
}

func findBreakpointByAddress(ctx *processContext, address uint64) *bpointData {
	for bPointAddress, bPoint := range ctx.bpointData {
		if bPointAddress == address {
			return bPoint
		}
	}

	return nil
}

func insertBreakpoint(ctx *processContext, breakpointAddress uint64) (originalInstruction []byte) {
	var interruptCode = []byte{0xCC} // code for breakpoint trap

	// store the replaced instruction in the process context
	// to swap it in later after breakpoint is hit
	originalInstruction = make([]byte, len(interruptCode))
	syscall.PtracePeekData(ctx.pid, uintptr(breakpointAddress), originalInstruction)

	// set breakpoint (insert interrupt code at the address)
	syscall.PtracePokeData(ctx.pid, uintptr(breakpointAddress), interruptCode)

	return originalInstruction
}

// restores the original instruction if the executable is currently caught at a breakpoint
func restoreCaughtBreakpoint(ctx *processContext) (caugtBpoint *bpointData) {
	regs := getRegs(ctx, true)

	bpoint := findBreakpointByAddress(ctx, regs.Rip)

	if bpoint == nil {
		logger.Info("Cannot find a breakpoint to restore")
		return nil
	}

	if bpoint.isMPIBpoint {
		logger.Info("Caught auto-inserted MPI breakpoint, func: %v", bpoint.function.name)
	} else {
		line, file, _, _ := ctx.dwarfData.PCToLine(regs.Rip)
		logger.Info("Caught at a breakpoint: line: %d, file: %v", line, file)
	}

	// replace the break instruction with the original instruction
	syscall.PtracePokeData(ctx.pid, uintptr(regs.Rip), bpoint.originalInstruction)

	// set the rewinded instruction pointer
	syscall.PtraceSetRegs(ctx.pid, regs)

	return bpoint
}
