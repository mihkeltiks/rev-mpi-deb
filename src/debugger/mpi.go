package main

import (
	"github.com/ottmartens/cc-rev-db/logger"
)

const MPI_WRAP_SIGNATURE_FUNC = "_MPI_WRAPPER_INCLUDE"

func insertMPIBreakpoints(ctx *processContext) {

	for _, function := range ctx.dwarfData.mpi.functions {
		address := function.lowPC

		logger.Info("Inserting mpi breakpoint: func: %v, address: %x", function.name, address)
		originalInstruction := insertBreakpoint(ctx, address)

		ctx.bpointData[address] = &bpointData{
			address,
			originalInstruction,
			function,
			true,
		}
	}
}

func recordMPIOperation(ctx *processContext, bpoint *bpointData) {
	// regs := getRegs(ctx, true)

	module, function := ctx.dwarfData.lookupFunc(bpoint.function.name)

	logger.Info("recording: %v from module %v", function.name, module.name)

	for _, param := range function.parameters {
		address, _, err := param.locationInstructions.decode()

		must(err)

		logger.Info("param %v of type %d at %#x", param.name, param.baseType.name, address)
	}
}
