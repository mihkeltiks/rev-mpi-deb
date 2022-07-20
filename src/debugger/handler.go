package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/debugger/dwarf"
	"github.com/ottmartens/cc-rev-db/debugger/proc"
	"github.com/ottmartens/cc-rev-db/logger"
)

type RemoteCmdHandler struct {
	ctx *processContext
}

func (r RemoteCmdHandler) Handle(cmd *command.Command, reply *int) error {
	logger.Debug("Scheduling command for execution %+v", cmd)
	r.ctx.commandQueue = append(r.ctx.commandQueue, cmd)
	return nil
}

func handleCommand(ctx *processContext, cmd *command.Command) {
	var err error
	var exited bool

	logger.Verbose("handling command %v", cmd)

	switch cmd.Code {
	case command.Bpoint:
		err = setBreakPoint(ctx, ctx.sourceFile, cmd.Argument.(int))
	case command.SingleStep:
		exited = continueExecution(ctx, true)
	case command.Cont:
		exited = continueExecution(ctx, false)
	case command.Restore:
		cpIndex := len(ctx.cpointData) - (1 + cmd.Argument.(int))

		restoreCheckpoint(ctx, cpIndex)
	case command.Print:
		printVariable(ctx, cmd.Argument.(string))
	case command.Quit:
		quitDebugger()
	case command.Help:
		printInstructions()
	case command.PrintInternal:
		printInternalData(ctx, cmd.Argument.(string))
	}

	if cmd.IsProgressCommand() {

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
			}

			if !bpoint.isMPIBpoint || cmd.Code == command.SingleStep {
				break
			}

			exited = continueExecution(ctx, false)
		}
	}

	if !exited {
		ctx.stack = getStack(ctx)

		if cmd.IsProgressCommand() || cmd.Code == command.Restore {
			logger.Info("call stack: %v", ctx.stack)
		}
	}

	cmd.Result = &command.CommandResult{Err: err, Exited: exited}
}

func setBreakPoint(ctx *processContext, file string, line int) (err error) {
	// logger.Debug("file %v", file)
	address, err := ctx.dwarfData.LineToPC(file, line)

	if err != nil {
		logger.Debug("cannot set breakpoint at line: %v", err)
		return err
	}

	logger.Debug("setting breakpoint at line: %d, file: %v", line, file)
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
			logger.Verbose("The binary exited with code %v", waitStatus.ExitStatus())
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

	parameter, stackFunction := ctx.stack.lookupParameter(varName)

	variable := parameter.AsVariable()

	if variable == nil {
		variable = ctx.dwarfData.LookupVariable(varName)

		if variable != nil && variable.Function != nil {
			stackFunction = ctx.stack.lookupFunction(variable.Function)
		}
	}

	if variable == nil {
		fmt.Printf("Cannot find variable: %s\n", varName)
		return nil
	}

	var frameBase int64

	if stackFunction != nil {
		frameBase = int64(stackFunction.baseAddress + 16)
	}

	address, _, err := variable.DecodeLocation(dwarf.DwarfRegisters{FrameBase: frameBase})

	// for _, sf := range ctx.stack {
	// 	logger.Info("fn %s, bp %d, sp %d", sf.function.Name(), sf.baseAddress, sf.stackAddress)
	// }

	// logger.Info("frameBase: %d", frameBase)

	if err != nil {
		panic(fmt.Sprintf("Error decoding variable: %v", err))
	}

	if address == 0 {
		fmt.Println("Cannot locate this variable")
		return nil
	}

	// logger.Debug("location of variable: %d", address)

	rawValue := peekDataFromMemory(ctx, address, variable.ByteSize())

	// logger.Debug("raw value of variable: %v", rawValue)

	// memRawValue := proc.ReadFromMemFile(ctx.pid, address, int(variable.baseType.byteSize))

	// fmt.Printf("raw value from ptrace: %v, mem-file: %v\n", rawValue, memRawValue)

	// logger.Debug("got raw value of variable %s: %v", varName, rawValue)

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
	logger.Info("Exiting")
	os.Exit(0)
}
