package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
)

func setBreakPoint(ctx *processContext, file string, line int) (err error) {

	address, err := ctx.dwarfData.lineToPC(file, line)

	if err != nil {
		logger.Info("cannot set breakpoint at line: %v", err)
		return err
	}

	logger.Info("setting breakpoint at file: %v, line: %d", file, line)
	originalInstruction := insertBreakpoint(ctx, address)

	ctx.bpointData[address] = &bpointData{
		address,
		originalInstruction,
		nil,
		false,
	}

	return
}

func continueExecution(ctx *processContext) (exited bool) {
	var waitStatus syscall.WaitStatus

	for i := 0; i < 100; i++ {
		syscall.PtraceCont(ctx.pid, 0)

		syscall.Wait4(ctx.pid, &waitStatus, 0, nil)

		if waitStatus.Exited() {
			logger.Info("The binary exited with code %v", waitStatus.ExitStatus())
			return true
		}

		if waitStatus.StopSignal() == syscall.SIGTRAP && waitStatus.TrapCause() != syscall.PTRACE_EVENT_CLONE {
			logger.Info("binary hit trap, execution paused")
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

func printVariable(ctx *processContext, varName string) {

	variable := ctx.dwarfData.lookupVariable(varName)

	if variable == nil {
		fmt.Printf("Cannot find variable: %s\n", varName)
		return
	}

	address, _, err := variable.locationInstructions.decode()

	if err != nil {
		panic(fmt.Sprintf("Error decoding variable: %v", err))
	}

	if address == 0 {
		fmt.Println("Cannot locate this variable")
		return
	}

	data := make([]byte, variable.baseType.byteSize)

	syscall.PtracePeekData(ctx.pid, uintptr(address), data)

	logger.Info("Printing variable %v", variable)

	var value interface{}

	switch variable.baseType.byteSize {
	case 4:
		value = int32(binary.LittleEndian.Uint32(data))
	case 8:
		value = int64(binary.LittleEndian.Uint64(data))
	default:
		fmt.Printf("unknown bytesize? %v\n", variable)
		return
	}

	fmt.Printf("Value of variable %s: %v\n", varName, value)
}

func printInternalData(ctx *processContext, varName string) {
	switch varName {
	case "types":
		logger.Info("dwarf types:\n%v", ctx.dwarfData.types)
	case "modules":
		logger.Info("dwarf modules:\n%v", ctx.dwarfData.modules)

	}
}

func quitDebugger() {
	fmt.Println("👋 Exiting..")
	os.Exit(0)
}
