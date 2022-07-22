package main

import (
	"fmt"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/nodeDebugger/dwarf"
	"github.com/ottmartens/cc-rev-db/rpc"
)

type mpiFuncNames struct {
	SIGNATURE string
	SEND      string
	RECEIVE   string
	FINALIZE  string
}

var MPI_FUNCS mpiFuncNames = mpiFuncNames{
	SIGNATURE: "_MPI_WRAPPER_INCLUDE",
	SEND:      "MPI_Send",
	RECEIVE:   "MPI_Recv",
	FINALIZE:  "MPI_Finalize",
}

type FunctionVariableMap map[string]VariableMap

type VariableMap map[string]string

var variablesToCapture FunctionVariableMap = FunctionVariableMap{
	MPI_FUNCS.SEND: VariableMap{
		"rank": "_MPI_WRAPPER_PROC_RANK",
		"tag":  "tag",
		"dest": "dest",
	},
	MPI_FUNCS.RECEIVE: VariableMap{
		"rank":   "_MPI_WRAPPER_PROC_RANK",
		"tag":    "tag",
		"source": "source",
	},
	MPI_FUNCS.FINALIZE: VariableMap{
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

func reinsertMPIBPoints(ctx *processContext, currentBpoint *bpointData) {
	for _, bp := range MPI_BPOINTS {
		if bp.function.Name() != currentBpoint.function.Name() {
			if !isMPIBpointSet(ctx, bp.function) {
				insertMPIBreakpoint(ctx, bp, false)
			}
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
