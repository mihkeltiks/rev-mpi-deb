package main

import (
	"fmt"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/proc"
)

type CheckpointMode int

const (
	fileMode CheckpointMode = iota
	forkMode
)

type checkpointData []cPoint

type cPoint struct {
	pid          int                 // process id of the fork at checkpoint
	opName       string              // name of the mpi operation where checkpoint was made
	regs         *syscall.PtraceRegs // register values at checkpoint
	stackRegions []proc.MemRegion    // stack region addresses of checkpoint
	stackRawData [][]byte            // raw value of stack at checkpoint
	bpoints      breakpointData      // breakpoints that were kept
}

func (c checkpointData) New() checkpointData {
	return make([]cPoint, 0)
}

func (cp cPoint) String() string {
	return fmt.Sprintf("cp{pid:%d, opName: %s}", cp.pid, cp.opName)
}

func restoreCheckpoint(ctx *processContext, cpIndex int) {

	if cpIndex < 0 || cpIndex >= len(ctx.cpointData) {
		fmt.Printf("No checkpoint at index %d\n", (len(ctx.cpointData) - (1 + cpIndex)))
		return
	}

	checkpoint := ctx.cpointData[cpIndex]

	logger.Info("restoring checkpoint: %v (pid %v)", checkpoint.opName, checkpoint.pid)

	logger.Info("fetching memory locations from checkpoint")

	checkpointMemRegions := proc.GetCheckpointDataAddresses(ctx.pid, ctx.targetFile)

	logger.Info("restoring memory state from checkpoint")

	for _, memRegion := range checkpointMemRegions {
		logger.Info("from checkpoint: %v", memRegion)

		data := memRegion.Contents(checkpoint.pid)

		_, err := syscall.PtracePokeData(ctx.pid, uintptr(memRegion.Start), data)

		must(err)
	}

	for index, memRegion := range checkpoint.stackRegions {
		logger.Info("from stack: %v", memRegion)

		data := checkpoint.stackRawData[index]

		_, err := syscall.PtracePokeData(ctx.pid, uintptr(memRegion.Start), data)

		must(err)
	}

	logger.Info("restoring registers state")

	// executeOnProcess(ctx, ctx.checkpointPid, func() {
	// 	checkpointRegs = getRegs(ctx, false)
	// })

	err := syscall.PtraceSetRegs(ctx.pid, checkpoint.regs)

	must(err)

	logger.Info("reverting breakpoints state")

	ctx.bpointData = checkpoint.bpoints

	insertMPIBreakpoint(ctx, MPI_BPOINTS[MPI_FUNCS.RECORD], true)

	logger.Info("bpoints at cp restore: %v", ctx.bpointData)

	// remove subsequent checkpoints

	ctx.cpointData = ctx.cpointData[:cpIndex+1]

	logger.Info("checkpoint restore finished")
}
