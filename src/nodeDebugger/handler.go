package main

import (
	"encoding/binary"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/nodeDebugger/dwarf"
	"github.com/mihkeltiks/rev-mpi-deb/nodeDebugger/proc"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
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
		exited = SingleStep(ctx)
	case command.Next:
		exited = continueExecution(ctx, false, true, false)
	case command.Cont:
		exited = continueExecution(ctx, false, false, false)
	case command.Restore:
		checkpointId := cmd.Argument.(string)
		err = restoreCheckpoint(ctx, checkpointId)
	case command.Print:
		printVariable(ctx, cmd.Argument.(string))
	case command.Insert:
		changeValueOfTarget(cmd.Argument.(int), ctx)
	case command.Retrieve:
		logger.Verbose("RETRIEVING, %v", cmd.Argument)
		retrieveVariable(cmd.Argument.(string), ctx)
	case command.RetrieveBreakpoints:
		logger.Verbose("RETRIEVING BREAKPOINTS")
		retrieveBreakpoints(ctx)
	case command.ChangeBreakpoints:
		AddNewBreakpoints(ctx, cmd.Argument.([]int))
	case command.RemoveBreakpoints:
		RemoveBreakpoints(ctx)
	case command.Quit:
		quitDebugger()
	case command.Help:
		printInstructions()
	case command.PrintInternal:
		printInternalData(ctx, cmd.Argument.(string))
	case command.Stop:
		if err := syscall.Kill(ctx.pid, syscall.SIGSTOP); err != nil {
			fmt.Println("Error sending SIGSTOP to the child process:", err)
		}
	case command.Kill:
		if err := syscall.Kill(-ctx.pid, syscall.SIGKILL); err != nil {
			fmt.Println("Error detaching from the child process:", err)
		}
		syscall.Wait4(ctx.pid, nil, 0, nil)
	case command.Detach:
		if err := syscall.PtraceDetach(ctx.pid); err != nil {
			fmt.Println("Error detaching from the child process:", err)
		}
	case command.Attach:
		if err := syscall.PtraceAttach(ctx.pid); err != nil {
			fmt.Println("Error attaching to the child process:", err)
		}
	case command.Reset:
		disconnect(ctx)
		time.Sleep(time.Duration(1200) * time.Millisecond)
		connect(ctx)
	}

	if cmd.IsForwardProgressCommand() {

		for {
			logger.Verbose("STUCK HERE")
			if exited {
				break
			}

			target, _, _ := getVariableFromMemory(ctx, "target", true)
			counter, _, _ := getVariableFromMemory(ctx, "counter", true)

			if target == counter {
				// For reverse continue recognize counter hit target
				if cmd.Code == command.Cont && cmd.Argument != nil {
					newCmd := command.Command{NodeId: ctx.nodeData.id, Code: command.CommandCode(-5)}
					reportBreakpoint(ctx, &newCmd)
				}
				break
			}

			bpoint, _, line := restoreCaughtBreakpoint(ctx)

			if bpoint == nil {
				break
			}

			if bpoint.isMPIBpoint {
				ctx.stack = getStack(ctx)

				// single-step, then insert all missing mpi bpoints
				continueExecution(ctx, true, false, false)
				reinsertMPIBPoints(ctx)

				recordMPIOperation(ctx, bpoint)
			}

			if bpoint.ignoreFirstHit {
				ctx.stack = getStack(ctx)
				// single-step, then reinsert bp
				continueExecution(ctx, true, false, false)
				err = setBreakPoint(ctx, ctx.sourceFile, line)
			}

			if (!bpoint.isMPIBpoint && !bpoint.ignoreFirstHit) || cmd.Code == command.SingleStep {
				break
			}

			exited = continueExecution(ctx, false, false, false)
		}
	}
	if !exited && command.Detach != cmd.Code && command.Kill != cmd.Code && command.Stop != cmd.Code && cmd.Code != command.Reset {
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

	if cmd.Code == command.Cont {
		handleCommand(ctx, &command.Command{NodeId: cmd.NodeId, Code: command.SingleStep})
	}
}

func disconnect(ctx *processContext) {
	if ctx.nodeData != nil && ctx.nodeData.rpcClient != nil {
		// Close the RPC connection
		ctx.nodeData.rpcClient.Disconnect()
	}
}

func connect(ctx *processContext) {
	orchestratorAdress, _ := url.ParseRequestURI("localhost:3490")
	ctx.nodeData = &nodeData{
		rpcClient: rpc.Connect(orchestratorAdress),
	}

	ctx.nodeData.id = reportAsHealthy(ctx)
	logger.SetRemoteClient(ctx.nodeData.rpcClient, ctx.nodeData.id)

	// logger.Info("Process (pid: %d) registered", os.Getpid())
}

func setBreakPoint(ctx *processContext, file string, line int) (err error) {
	ignore := false
	if line < 0 {
		logger.Verbose("%v", line)
		ignore = true
		line = -line
	}
	address, err := ctx.dwarfData.LineToPC(file, line)

	if err != nil {
		logger.Warn("cannot set breakpoint at line: %v", err)
		return err
	}

	logger.Info("setting breakpoint at line: %d", line)
	originalInstruction := insertBreakpoint(ctx, address)

	ctx.bpointData[address] = &bpointData{
		address:                 address,
		originalInstruction:     originalInstruction,
		function:                nil,
		isMPIBpoint:             false,
		isImmediateAfterRestore: false,
		ignoreFirstHit:          ignore,
		line:                    line,
	}

	return nil
}

func SingleStep(ctx *processContext) bool {
	intialValue := changeTargetForStep(ctx)
	logger.Verbose("INITIAL %v", intialValue)
	exited := continueExecution(ctx, false, false, true)
	changeValueOfTarget(intialValue, ctx)
	printInternalData(ctx, "loc")
	stepOutOfCounter(ctx)
	return exited
}

func continueExecution(ctx *processContext, singleStep bool, next bool, counter bool) (exited bool) {
	var waitStatus syscall.WaitStatus

	for i := 0; i < 20; i++ {

		if singleStep || next {
			err := syscall.PtraceSingleStep(ctx.pid)
			utils.Must(err)
		} else {
			logger.Verbose("STUCK HERE 2 %v", counter)

			err := syscall.PtraceCont(ctx.pid, 0)
			utils.Must(err)
		}

		syscall.Wait4(ctx.pid, &waitStatus, 0, nil)

		if waitStatus.Exited() {
			logger.Verbose("The binary exited with code %v", waitStatus.ExitStatus())
			return true
		}

		if waitStatus.StopSignal() == syscall.SIGTRAP && waitStatus.TrapCause() != syscall.PTRACE_EVENT_CLONE {
			logger.Verbose("In here, binary hit trap, execution paused (wait status: %v, trap cause: %v)", waitStatus, waitStatus.TrapCause())
			if counter {
				if compareTargetAndCounter(ctx) {
					return false
				}
			} else {
				return false
			}
		}

		// else {
		// received a signal other than trap/a trap from clone event, continue and wait more
		// }
	}

	printVariable(ctx, "counter")
	return false
}

func AddNewBreakpoints(ctx *processContext, breakpoints []int) {
	for _, line := range breakpoints {
		err := setBreakPoint(ctx, ctx.sourceFile, line)
		utils.Must(err)
	}
}

func RemoveBreakpoints(ctx *processContext) {
	for address, bpoint := range ctx.bpointData {
		if !bpoint.isMPIBpoint {
			_, err := syscall.PtracePokeData(ctx.pid, uintptr(address), bpoint.originalInstruction)
			utils.Must(err)

			// remove record of breakpoint
			delete(ctx.bpointData, bpoint.address)
		}
	}
}

func stepOutOfCounter(ctx *processContext) (exited bool) {
	var waitStatus syscall.WaitStatus

	for i := 0; i < 1000; i++ {
		err := syscall.PtraceSingleStep(ctx.pid)
		utils.Must(err)

		syscall.Wait4(ctx.pid, &waitStatus, 0, nil)

		regs := getRegs(ctx, false)
		line, _, _, _ := ctx.dwarfData.PCToLine(regs.Rip)
		// Outside of counter bounds
		if line > 0 {
			logger.Verbose("LINE %v", line)
			return false
		}

		if waitStatus.Exited() {
			logger.Verbose("The binary exited with code %v", waitStatus.ExitStatus())
			return true
		}
	}
	panic(fmt.Sprintf("stuck at wait with signal: %v", waitStatus.StopSignal()))

}

func printVariable(ctx *processContext, varName string) {
	value, _, _ := getVariableFromMemory(ctx, varName, false)
	if value == nil {
		return
	}

	fmt.Printf("Value of variable %s: %v\n", varName, value)
}

// Retrieves the value of a variable matching the specified idendifier, if present in the target
func getVariableFromMemory(ctx *processContext, identifier string, suppressLogging bool) (value interface{}, address uint64, size int64) {
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
			logger.Verbose("Cannot locate variable: %s", identifier)
		}

		return nil, 0, 0
	}

	var frameBase int64

	if variableStackFunction != nil {
		frameBase = int64(variableStackFunction.baseAddress + 16)
	}

	// Debug the variable location instructions to obtain memory address
	address, _, err := variable.DecodeLocation(dwarf.DwarfRegisters{FrameBase: frameBase})

	if err != nil {
		logger.Error("Error decoding variable: %v", err)
		return nil, 0, 0
	}

	if address == 0 {
		logger.Warn("Cannot locate this variable")
		return nil, 0, 0
	}

	// logger.Debug("location of variable: %d", address)

	rawValue := peekDataFromMemory(ctx, address, variable.ByteSize())
	// rawValue := proc.ReadFromMemFile(ctx.pid, address, int(variable.baseType.byteSize))
	// logger.Debug("raw value of variable: %v", rawValue)
	// Convert the binary value to accurate type representation
	return convertValueToType(rawValue, variable), address, variable.ByteSize()
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

func changeValueOfTarget(newValue int, ctx *processContext) {
	_, address, size := getVariableFromMemory(ctx, "target", true)
	bs := make([]byte, size)
	binary.LittleEndian.PutUint32(bs, uint32(newValue))
	_, err := syscall.PtracePokeData(ctx.pid, uintptr(address), bs)
	utils.Must(err)
}

func compareTargetAndCounter(ctx *processContext) bool {
	counter, _, _ := getVariableFromMemory(ctx, "counter", true)
	target, _, _ := getVariableFromMemory(ctx, "target", true)
	logger.Verbose("COUNTER %v", int(counter.(int32)))
	logger.Verbose("TARGET %v", int(target.(int32)))
	return counter.(int32) == target.(int32)
}

func changeTargetForStep(ctx *processContext) int {
	counter, _, _ := getVariableFromMemory(ctx, "counter", true)
	target, address, size := getVariableFromMemory(ctx, "target", true)

	newValue := counter.(int32)
	newValue++

	bs := make([]byte, size)
	binary.LittleEndian.PutUint32(bs, uint32(newValue))

	_, err := syscall.PtracePokeData(ctx.pid, uintptr(address), bs)
	utils.Must(err)
	logger.Verbose("COUNTER IS %v", int(counter.(int32)))
	logger.Verbose("TARGET SET TO %v", int(newValue))

	return int(target.(int32))
}

func retrieveVariable(name string, ctx *processContext) {
	value, _, size := getVariableFromMemory(ctx, name, true)
	counter := value.(int32)
	logger.Verbose("REPORTING VALUE %v", counter)
	logger.Verbose("SIZE  %v", int(size))
	reportCounter(ctx, &command.Command{NodeId: ctx.nodeData.id, Code: command.Retrieve, Argument: value})
}

func retrieveBreakpoints(ctx *processContext) {
	var breakpoints []int
	breakpoints = append(breakpoints, ctx.nodeData.id)

	for _, value := range ctx.bpointData {
		if !value.isMPIBpoint {
			breakpoints = append(breakpoints, value.line)
		}
	}
	reportBreakpoints(ctx, &breakpoints)
}

func printInternalData(ctx *processContext, varName string) {
	switch varName {
	case "last":
		logger.Info("%d", len(ctx.stack))
		funct := getLastExecutedFunction(ctx.stack)
		funct2 := ctx.stack[0].function
		logger.Verbose("%v", funct2.Name() == "call_counter")
		printRegs(ctx)
		logger.Info(funct.String())
		logger.Info(funct2.String())
	case "types":
		logger.Info("dwarf types:\n%v", ctx.dwarfData.Types)
	case "modules":
		logger.Info("dwarf modules:\n%v", ctx.dwarfData.Modules)
	case "vars":
		logger.Info("dwarf variables: %v\n", ctx.dwarfData.Modules[0].Variables)
	case "maps":
		logger.Info("proc/id/maps:")
		proc.LogMapsFile(ctx.pid)
	case "breakpoints":
		printVariable(ctx, "target")
		for _, value := range ctx.bpointData {
			if !value.isMPIBpoint {
				logger.Verbose("BP %v", value.line)
			}
		}
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
