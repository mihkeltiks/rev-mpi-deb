package main

import (
	"github.com/ottmartens/cc-rev-db/logger"
)

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

	for _, function := range ctx.dwarfData.mpi.functions {

		funcEntries := ctx.dwarfData.getEntriesForFunction(function.name)

		var breakEntry dwarfEntry

		if function.name == MPI_FUNCS.RECORD {
			breakEntry = funcEntries[len(funcEntries)-1]
		} else {
			breakEntry = funcEntries[0]
		}

		originalInstruction := getOriginalInstruction(ctx, breakEntry.address)

		MPI_BPOINTS[function.name] = &bpointData{
			breakEntry.address,
			originalInstruction,
			function,
			true,
			false,
		}
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
	for _, bp := range MPI_BPOINTS {
		if bp.function.name != currentBpoint.function.name {
			if !isMPIBpointSet(ctx, bp.function) {
				insertMPIBreakpoint(ctx, bp, false)
			}
		}
	}

}

func recordMPIOperation(ctx *processContext, bpoint *bpointData) {

	opName := bpoint.function.name

	logger.Debug("\trecording mpi operation %v", opName)

	if opName == MPI_FUNCS.RECORD && !bpoint.isImmediateAfterRestore {
		createCheckpoint(ctx, currentMPIFunc.function.name)
	}
	// else {
	// printVariable(ctx, "_MPI_CURRENT_DEST")
	// printVariable(ctx, "_MPI_CURRENT_SOURCE")
	// printVariable(ctx, "_MPI_CURRENT_TAG")
	// }

	currentMPIFunc = currentMPIFuncData{
		addresses: make([]uint64, 0),
		function:  bpoint.function,
	}

}
