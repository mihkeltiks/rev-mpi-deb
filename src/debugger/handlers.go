package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/go-delve/delve/pkg/dwarf/op"
	"github.com/ottmartens/cc-rev-db/dwarf"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/proc"
)

func (cmd *command) handle(ctx *processContext) *cmdResult {
	var err error
	var exited bool

	logger.Info("handling command %+v", cmd)

	switch cmd.code {
	case bpoint:
		err = setBreakPoint(ctx, ctx.sourceFile, cmd.argument.(int))
	case singleStep:
		exited = continueExecution(ctx, true)
	case cont:
		exited = continueExecution(ctx, false)
	case restore:
		cpIndex := len(ctx.cpointData) - (1 + cmd.argument.(int))

		restoreCheckpoint(ctx, cpIndex)
	case print:
		printVariable(ctx, cmd.argument.(string))
	case quit:
		quitDebugger()
	case help:
		printInstructions()
	case printInternal:
		printInternalData(ctx, cmd.argument.(string))
	}

	if cmd.isProgressCommand() {

		for {

			if exited {
				break
			}

			bpoint, _ := restoreCaughtBreakpoint(ctx)

			if bpoint == nil {
				break
			}

			if bpoint.isMPIBpoint {

				recordMPIOperation(ctx, bpoint)

				reinsertMPIBPoints(ctx, bpoint)
			} else {

				printInternalData(ctx, "loc")
			}

			if !bpoint.isMPIBpoint || cmd.code == singleStep {
				break
			}

			exited = continueExecution(ctx, false)
		}
	}

	return &cmdResult{err, exited}
}

func setBreakPoint(ctx *processContext, file string, line int) (err error) {
	logger.Debug("file %v", file)
	address, err := ctx.dwarfData.LineToPC(file, line)

	if err != nil {
		logger.Debug("cannot set breakpoint at line: %v", err)
		return err
	}

	logger.Debug("setting breakpoint at file: %v, line: %d", file, line)
	originalInstruction := insertBreakpoint(ctx, address)

	ctx.bpointData[address] = &bpointData{
		address:                 address,
		originalInstruction:     originalInstruction,
		function:                nil,
		isMPIBpoint:             false,
		isImmediateAfterRestore: false,
	}

	return
}

func continueExecution(ctx *processContext, singleStep bool) (exited bool) {
	var waitStatus syscall.WaitStatus

	for i := 0; i < 100; i++ {

		if singleStep {
			err := syscall.PtraceSingleStep(ctx.pid)
			must(err)
		} else {
			err := syscall.PtraceCont(ctx.pid, 0)
			must(err)
		}

		syscall.Wait4(ctx.pid, &waitStatus, 0, nil)

		if waitStatus.Exited() {
			logger.Debug("The binary exited with code %v", waitStatus.ExitStatus())
			return true
		}

		if waitStatus.StopSignal() == syscall.SIGTRAP && waitStatus.TrapCause() != syscall.PTRACE_EVENT_CLONE {
			logger.Debug("binary hit trap, execution paused (wait status: %v, trap cause: %v)", waitStatus, waitStatus.TrapCause())
			return false
		}
		// else {
		// received a signal other than trap/a trap from clone event, continue and wait more
		// }
	}

	panic(fmt.Sprintf("stuck at wait with signal: %v", waitStatus.StopSignal()))
}

func printVariable(ctx *processContext, varName string) {

	value := getVariableFromMemory(ctx, varName)

	if value == nil {
		return
	}

	fmt.Printf("Value of variable %s: %v\n", varName, value)
}

func getVariableFromMemory(ctx *processContext, varName string) (value interface{}) {
	variable := ctx.dwarfData.LookupVariable(varName)

	if variable == nil {
		fmt.Printf("Cannot find variable: %s\n", varName)
		return nil
	}

	address, _, err := variable.DecodeLocation(op.DwarfRegisters{})

	if err != nil {
		panic(fmt.Sprintf("Error decoding variable: %v", err))
	}

	if address == 0 {
		fmt.Println("Cannot locate this variable")
		return nil
	}

	rawValue := peekDataFromMemory(ctx, address, variable.ByteSize())

	logger.Info("location of variable: %#x", address)
	logger.Info("raw value of variable: %v", rawValue)
	// memRawValue := proc.ReadFromMemFile(ctx.pid, address, int(variable.baseType.byteSize))

	// fmt.Printf("raw value from ptrace: %v, mem-file: %v\n", rawValue, memRawValue)

	logger.Debug("got raw value of variable %s: %v", varName, rawValue)

	return convertValueToType(rawValue, variable)
}

func peekDataFromMemory(ctx *processContext, address uint64, byteCount int64) []byte {
	data := make([]byte, byteCount)

	syscall.PtracePeekData(ctx.pid, uintptr(address), data)

	return data
}

func convertValueToType(data []byte, variable *dwarf.Variable) interface{} {

	var value interface{}

	switch variable.ByteSize() {
	case 4:
		value = int32(binary.LittleEndian.Uint32(data))
	case 8:
		value = int64(binary.LittleEndian.Uint64(data))
	default:
		fmt.Printf("unknown bytesize? %v\n", variable)
	}

	return value
}

func printInternalData(ctx *processContext, varName string) {
	switch varName {
	case "types":
		logger.Info("dwarf types:\n%v", ctx.dwarfData.Types)
	case "modules":
		logger.Info("dwarf modules:\n%v", ctx.dwarfData.Modules)
	case "vars":
		logger.Info("dwarf variables: %v\n", ctx.dwarfData.Modules[0].Variables)
	case "maps":
		logger.Info("proc/id/maps:")
		proc.LogMapsFile(ctx.pid)
	case "loc":
		regs, _ := getRegs(ctx, false)
		line, fileName, fn, _ := ctx.dwarfData.PCToLine(regs.Rip)
		logger.Info("currently at line %v in %v (func %v) ip:%#x", line, filepath.Base(fileName), fn.Name(), regs.Rip)
	case "cp":
		logger.Info("checkpoints: %v", ctx.cpointData)
	}

}

func quitDebugger() {
	fmt.Println("👋 Exiting..")
	os.Exit(0)
}
