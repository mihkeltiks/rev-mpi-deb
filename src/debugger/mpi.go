package main

import (
	"github.com/ottmartens/cc-rev-db/dwarf"
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

	for _, function := range ctx.dwarfData.Mpi.Functions {

		fName := function.Name()

		funcEntries := ctx.dwarfData.GetEntriesForFunction(fName)

		var breakEntry dwarf.Entry

		if fName == MPI_FUNCS.RECORD {
			breakEntry = funcEntries[len(funcEntries)-1]
		} else {
			breakEntry = funcEntries[0]
		}

		originalInstruction := getOriginalInstruction(ctx, breakEntry.Address)

		MPI_BPOINTS[fName] = &bpointData{
			breakEntry.Address,
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

var currentMPIFunc currentMPIFuncData

type currentMPIFuncData struct {
	addresses []uint64
	function  *dwarf.Function
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

	logger.Debug("\trecording mpi operation %v", opName)

	if opName == MPI_FUNCS.RECORD && !bpoint.isImmediateAfterRestore {
		createCheckpoint(ctx, currentMPIFunc.function.Name())
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
