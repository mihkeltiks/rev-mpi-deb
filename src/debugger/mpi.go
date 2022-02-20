package main

import (
	"fmt"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
)

type fn func()

type mpiFuncNames struct {
	SIGNATURE string
	SEND      string
	RECEIVE   string
	RECORD    string
}

var MPI_FUNCS mpiFuncNames = mpiFuncNames{
	SIGNATURE: "_MPI_WRAPPER_INCLUDE",
	RECORD:    "MPI_WRAPPER_RECORD",
	SEND:      "MPI_Send",
	RECEIVE:   "MPI_Receive",
}

func insertMPIBreakpoints(ctx *processContext) {
	for _, function := range ctx.dwarfData.mpi.functions {
		insertMPIBreakpoint(ctx, function)
	}
}

func insertMPIBreakpoint(ctx *processContext, function *dwarfFunc) {
	logger.Info("inserting bpoint for MPI function: %v", function)

	funcEntries := ctx.dwarfData.getEntriesForFunction(function.name)
	breakEntry := funcEntries[len(funcEntries)-1]

	address := breakEntry.address

	originalInstruction := insertBreakpoint(ctx, address)

	ctx.bpointData[address] = &bpointData{
		address,
		originalInstruction,
		function,
		true,
	}
}

func isMPIBpointSet(ctx *processContext, function *dwarfFunc) bool {
	for _, bpoint := range ctx.bpointData {
		if bpoint.isMPIBpoint && bpoint.function == function {
			return true
		}
	}

	return false
}

var currentMPIFunc currentMPIFuncData

type currentMPIFuncData struct {
	addresses []uint64
	function  *dwarfFunc
}

func reinsertMPIBPoints(ctx *processContext, currentBpoint *bpointData) {
	for _, function := range ctx.dwarfData.mpi.functions {
		if function.name != currentBpoint.function.name {
			if !isMPIBpointSet(ctx, function) {
				insertMPIBreakpoint(ctx, function)
			}
		}
	}

}

func runOnCheckpoint(ctx *processContext, function fn) {
	realPid := ctx.pid

	ctx.pid = ctx.checkpointPid

	function()

	ctx.pid = realPid
}

func recordMPIOperation(ctx *processContext, bpoint *bpointData) {

	currentMPIFunc = currentMPIFuncData{
		addresses: make([]uint64, 0),
		function:  bpoint.function,
	}

	switch bpoint.function.name {
	case MPI_FUNCS.SEND:

		// printVariable(ctx, "_MPI_CURRENT_DEST")
		// printVariable(ctx, "_MPI_CURRENT_TAG")
	case MPI_FUNCS.RECEIVE:

		// printVariable(ctx, "_MPI_CURRENT_SOURCE")
		// printVariable(ctx, "_MPI_CURRENT_TAG")
	case MPI_FUNCS.RECORD:
		printVariable(ctx, "_MPI_CURRENT_DEST")
		printVariable(ctx, "_MPI_CURRENT_SOURCE")
		printVariable(ctx, "_MPI_CURRENT_TAG")

		if ctx.checkpointPid == 0 {
			logger.Info("regs in parent")
			printRegs(ctx)
			ctx.checkpointPid = int(getVariableFromMemory(ctx, "_MPI_CHECKPOINT_CHILD").(int32))

			logger.Info("child proc id is %d", ctx.checkpointPid)
		} else {

			err := syscall.PtraceAttach(ctx.checkpointPid)

			if err != nil {
				fmt.Println(err)
			}

			syscall.Wait4(ctx.checkpointPid, nil, 0, nil)

			logger.Info("from child:")
			runOnCheckpoint(ctx, func() {
				printVariable(ctx, "_MPI_CURRENT_DEST")
				printVariable(ctx, "_MPI_CURRENT_SOURCE")
				printVariable(ctx, "_MPI_CURRENT_TAG")
			})

		}

	}

	// var recordArgIndices []int

	// switch function.name {
	// case "MPI_Send":
	// 	recordArgIndices = []int{
	// 		1,
	// 		3,
	// 		4,
	// 	}
	// case "MPI_Recv":
	// 	recordArgIndices = []int{
	// 		1,
	// 		3,
	// 		4,
	// 	}

	// case "MPI_WRAPPER_RECORD":
	// 	recordArgIndices = []int{
	// 		3,
	// 		4,
	// 		5,
	// 	}
	// default:
	// 	recordArgIndices = nil
	// }

	// if recordArgIndices == nil {
	// 	return
	// }

	// regs := getRegs(ctx, false)

	// 		frameBase := int64(regs.Rbp - 16)
	// 		dRegs := DwarfRegisters{FrameBase: frameBase}

	// 		address, _, err := ExecuteStackProgram(dRegs, param.locationInstructions, ptrSize(), nil)
	// 		logger.Info("param %s (type: '%v' location: %s)", param.name, param.baseType.name, param.locationInstructions)
	// 		must(err)

	// 		rawValue := peekDataFromMemory(ctx, uint64(address), 4)
	// 		value := convertValueToType(rawValue, &dwarfBaseType{name: "int", byteSize: 4}).(int32)
	// 		logger.Info("\t value of %s at %#x - %v - framebase: %#x -> address: %#x", param.name, address, value, frameBase, address)

}
