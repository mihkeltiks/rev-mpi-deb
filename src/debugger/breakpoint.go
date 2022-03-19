package main

import (
	"fmt"
	"path/filepath"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
)

type breakpointData map[uint64]*bpointData

type bpointData struct {
	address                 uint64     // address of the instruction
	originalInstruction     []byte     // actual contents of the instruction at address
	function                *dwarfFunc // the pointer to the function the breakpoint was inserted at
	isMPIBpoint             bool
	isImmediateAfterRestore bool
}

func (b *bpointData) String() string {
	if b.function != nil {
		return fmt.Sprintf("{address: %#x (%v)}", b.address, b.function.name)
	}
	return fmt.Sprintf("{address: %#x}", b.address)
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

	_, err := syscall.PtracePeekData(ctx.pid, uintptr(breakpointAddress), originalInstruction)
	must(err)

	// set breakpoint (insert interrupt code at the address)
	_, err = syscall.PtracePokeData(ctx.pid, uintptr(breakpointAddress), interruptCode)
	must(err)

	return originalInstruction
}

func getOriginalInstruction(ctx *processContext, address uint64) (originalInstruction []byte) {
	var interruptCode = []byte{0xCC} // code for breakpoint trap

	originalInstruction = make([]byte, len(interruptCode))

	syscall.PtracePeekData(ctx.pid, uintptr(address), originalInstruction)

	return originalInstruction
}

// restores the original instruction if the executable is currently caught at a breakpoint
func restoreCaughtBreakpoint(ctx *processContext) (caugtBpoint *bpointData, registers *syscall.PtraceRegs) {
	regs, err := getRegs(ctx, true)
	must(err)

	// line, file, fn, _ := ctx.dwarfData.PCToLine(regs.Rip)
	// logger.Debug("looking to restore bpoint at %#x (line %d in %s, func: %v)", regs.Rip, line, filepath.Base(file), fn.Name())

	bpoint := findBreakpointByAddress(ctx, regs.Rip)

	if bpoint == nil {
		logger.Debug("Cannot find a breakpoint to restore")
		return nil, nil
	}

	if bpoint.isMPIBpoint {
		logger.Debug("Caught auto-inserted MPI breakpoint, func: %v", bpoint.function.name)
	} else {
		line, file, _, _ := ctx.dwarfData.PCToLine(regs.Rip)
		logger.Debug("Caught at a breakpoint: line: %d, file: %v", line, filepath.Base(file))
	}

	// replace the break instruction with the original instruction
	_, err = syscall.PtracePokeData(ctx.pid, uintptr(regs.Rip), bpoint.originalInstruction)
	must(err)

	// set the rewinded instruction pointer
	err = syscall.PtraceSetRegs(ctx.pid, regs)
	must(err)

	// remove record of breakpoint
	delete(ctx.bpointData, bpoint.address)

	return bpoint, regs
}
