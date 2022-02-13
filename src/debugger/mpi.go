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
