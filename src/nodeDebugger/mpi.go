package main

import (
	"fmt"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/nodeDebugger/dwarf"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils/mpi"
)

type FunctionVariableMap map[string]VariableMap

type VariableMap map[string]string

var variablesToCapture FunctionVariableMap = FunctionVariableMap{
	mpi.MPI_OPS[mpi.OP_SEND]: VariableMap{
		"rank": "_MPI_WRAPPER_PROC_RANK",
		"tag":  "tag",
		"dest": "dest",
	},
	mpi.MPI_OPS[mpi.OP_RECV]: VariableMap{
		"rank":   "_MPI_WRAPPER_PROC_RANK",
		"tag":    "tag",
		"source": "source",
	},
	mpi.MPI_OPS[mpi.OP_FINALIZE]: VariableMap{
		"rank": "_MPI_WRAPPER_PROC_RANK",
	},
}

var MPI_BPOINTS map[string]*bpointData

func insertMPIBreakpoints(ctx *processContext) {

	if MPI_BPOINTS == nil {
		initMPIBreakpointsData(ctx)
	}

	for _, bpoint := range MPI_BPOINTS {
		insertMPIBreakpoint(ctx, bpoint, false)
	}
}

func insertMPIBreakpoint(ctx *processContext, bpoint *bpointData, isImmediateAfterRestore bool) {

	address := bpoint.address

	logger.Debug("inserting bpoint for MPI function: %v (at %#x)", bpoint.function.Name(), address)

	insertBreakpoint(ctx, address)

	ctx.bpointData[address] = &bpointData{
		address,
		bpoint.originalInstruction,
		bpoint.function,
		bpoint.isMPIBpoint,
		isImmediateAfterRestore,
		false,
	}
}

func initMPIBreakpointsData(ctx *processContext) {

	MPI_BPOINTS = make(map[string]*bpointData)

	for _, function := range ctx.dwarfData.Mpi.Functions {
		fName := function.Name()

		funcEntries := ctx.dwarfData.GetEntriesForFunction(fName)
		breakAddress := funcEntries[1].Address

		originalInstruction := getOriginalInstruction(ctx, breakAddress)

		MPI_BPOINTS[fName] = &bpointData{
			breakAddress,
			originalInstruction,
			function,
			true,
			false,
			false,
		}
	}
}

func isMPIBpointSet(ctx *processContext, function *dwarf.Function) bool {
	for _, bpoint := range ctx.bpointData {
		if bpoint.isMPIBpoint && bpoint.function == function {
			return true
		}
	}

	return false
}

func reinsertMPIBPoints(ctx *processContext) {
	for _, bp := range MPI_BPOINTS {
		if !isMPIBpointSet(ctx, bp.function) {
			insertMPIBreakpoint(ctx, bp, false)
		}
	}
}

func recordMPIOperation(ctx *processContext, bpoint *bpointData) {
	opName := bpoint.function.Name()

	logger.Info("Recording MPI operation %v", opName)

	checkpointId := createCheckpoint(ctx, opName)

	record := rpc.MPICallRecord{
		Id:         checkpointId,
		OpName:     opName,
		Parameters: make(map[string]string),
		NodeId:     ctx.nodeData.id,
	}

	for varName, identifier := range variablesToCapture[opName] {
		variableValue := getVariableFromMemory(ctx, identifier, true)
		record.Parameters[varName] = fmt.Sprintf("%v", variableValue)
	}

	logger.Debug("MPI Call record: %v", record)
	reportMPICall(ctx, &record)
}
