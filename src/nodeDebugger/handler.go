package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/nodeDebugger/dwarf"
	"github.com/ottmartens/cc-rev-db/nodeDebugger/proc"
	"github.com/ottmartens/cc-rev-db/utils"
	"github.com/ottmartens/cc-rev-db/utils/command"
)

type RemoteCmdHandler struct {
	ctx          *processContext
	commandQueue chan<- *command.Command
}

func (r RemoteCmdHandler) Handle(cmd *command.Command, reply *int) error {
	logger.Debug("Scheduling command for execution %+v", cmd)
	r.commandQueue <- cmd
	return nil
}

func handleCommand(ctx *processContext, cmd *command.Command) {
	var err error
	var exited bool

	logger.Verbose("handling command %v", cmd)

	if cmd.IsForwardProgressCommand() {
		reportProgressCommand(ctx, cmd)
	}

	switch cmd.Code {
	case command.Bpoint:
		err = setBreakPoint(ctx, ctx.sourceFile, cmd.Argument.(int))
	case command.SingleStep:
		exited = continueExecution(ctx, true)
	case command.Cont:
		exited = continueExecution(ctx, false)
	case command.Restore:
		checkpointId := cmd.Argument.(string)
		err = restoreCheckpoint(ctx, checkpointId)
	case command.Print:
		printVariable(ctx, cmd.Argument.(string))
	case command.Quit:
		quitDebugger()
	case command.Help:
		printInstructions()
	case command.PrintInternal:
		printInternalData(ctx, cmd.Argument.(string))
	}

	if cmd.IsForwardProgressCommand() {

		for {
			if exited {
				break
			}

			bpoint, _ := restoreCaughtBreakpoint(ctx)

			if bpoint == nil {
				break
			}

			if bpoint.isMPIBpoint {
				ctx.stack = getStack(ctx)

				// single-step, then insert all missing mpi bpoints
				continueExecution(ctx, true)
				reinsertMPIBPoints(ctx)

				recordMPIOperation(ctx, bpoint)
			}

			if !bpoint.isMPIBpoint || cmd.Code == command.SingleStep {
				break
			}

			exited = continueExecution(ctx, false)
		}
	}

	if !exited {
		ctx.stack = getStack(ctx)

		if cmd.IsProgressCommand() {
			logger.Info("call stack: %v", ctx.stack)
		}
	}

	cmd.Result = &command.CommandResult{
		Exited: exited,
	}

	if err != nil {
		cmd.Result.Error = err.Error()
	}
}

func setBreakPoint(ctx *processContext, file string, line int) (err error) {
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

	return nil
}

func continueExecution(ctx *processContext, singleStep bool) (exited bool) {
	var waitStatus syscall.WaitStatus

	for i := 0; i < 100; i++ {

		if singleStep {
			err := syscall.PtraceSingleStep(ctx.pid)
			utils.Must(err)
		} else {
			err := syscall.PtraceCont(ctx.pid, 0)
			utils.Must(err)
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
	value := getVariableFromMemory(ctx, varName, false)
	if value == nil {
		return
	}

	fmt.Printf("Value of variable %s: %v\n", varName, value)
}

// Retrieves the value of a variable matching the specified idendifier, if present in the target
func getVariableFromMemory(ctx *processContext, identifier string, suppressLogging bool) (value interface{}) {
	var variable *dwarf.Variable
	var variableStackFunction *stackFunction

	// Process the call stack to find the matching variable
	for _, stackFunction := range ctx.stack {
		// Look for the variable declared in the stack function
		variable = ctx.dwarfData.LookupVariableInFunction(stackFunction.function, identifier)

		if variable != nil {
			if !suppressLogging {
				logger.Debug("Referring to variable %v as declared in function %v", identifier, stackFunction.function.Name())
			}

			variableStackFunction = stackFunction
			break
		}

		// Inspect the parameters of the stack function
		matchingParameter := stackFunction.lookupParameter(identifier)

		if matchingParameter != nil {
			if !suppressLogging {
				logger.Verbose("Referring to variable %v as function parameter for function %v", identifier, stackFunction.function.Name())
			}

			variable = matchingParameter.AsVariable()
			variableStackFunction = stackFunction
			break
		}
	}

	if variable == nil {
		// Look for a global variable
		variable = ctx.dwarfData.LookupVariable(identifier)
		if variable != nil {

			if !suppressLogging {
				logger.Verbose("Referring to variable %v as a global variable", identifier)
			}
		}
	}

	if variable == nil {
		if !suppressLogging {
			logger.Info("Cannot locate variable: %s", identifier)
		}

		return nil
	}

	var frameBase int64

	if variableStackFunction != nil {
		frameBase = int64(variableStackFunction.baseAddress + 16)
	}

	// Debug the variable location instructions to obtain memory address
	address, _, err := variable.DecodeLocation(dwarf.DwarfRegisters{FrameBase: frameBase})

	if err != nil {
		logger.Error("Error decoding variable: %v", err)
		return nil
	}

	if address == 0 {
		logger.Warn("Cannot locate this variable")
		return nil
	}

	// logger.Debug("location of variable: %d", address)

	rawValue := peekDataFromMemory(ctx, address, variable.ByteSize())
	// rawValue := proc.ReadFromMemFile(ctx.pid, address, int(variable.baseType.byteSize))
	// logger.Debug("raw value of variable: %v", rawValue)

	// Convert the binary value to accurate type representation
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
		logger.Error("unknown bytesize %v\n", variable)
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
		regs := getRegs(ctx, false)
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
