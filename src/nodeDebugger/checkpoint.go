package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/nodeDebugger/proc"
)

type CheckpointMode int

const (
	fileMode CheckpointMode = iota
	forkMode
)

type checkpointData []cPoint

type cPoint struct {
	opName string              // name of the mpi operation where checkpoint was made
	regs   *syscall.PtraceRegs // register values at checkpoint
	id     string              // unique id of the checkpoint

	// file mode
	file    string           // file in which checkpoint data is stored
	regions []proc.MemRegion // descriptors of memory ranges

	// fork mode
	pid          int              // process id of the fork at checkpoint
	stackRegions []proc.MemRegion // stack region addresses of checkpoint
	stackRawData [][]byte         // raw value of stack at checkpoint
	bpoints      breakpointData   // breakpoints at checkpoint time

}

func (c checkpointData) New() checkpointData {
	return make([]cPoint, 0)
}

func (cp cPoint) String() string {
	return fmt.Sprintf("cp{opName: %s}", cp.opName)
}

func createCheckpoint(ctx *processContext, opName string) string {

	logger.Verbose("creating new checkpoint (%v)", opName)

	var checkpoint cPoint

	if ctx.checkpointMode == fileMode {
		checkpoint = createFileCheckpoint(ctx, opName)
	} else {
		checkpoint = createForkCheckpoint(ctx, opName)
	}

	checkpoint.id = randomId()

	for address, bp := range ctx.bpointData {
		checkpoint.bpoints[address] = &bpointData{
			address:                 bp.address,
			originalInstruction:     bp.originalInstruction,
			function:                bp.function,
			isMPIBpoint:             bp.isMPIBpoint,
			isImmediateAfterRestore: false,
		}
	}

	ctx.cpointData = append(ctx.cpointData, checkpoint)

	return checkpoint.id
}

func restoreCheckpoint(ctx *processContext, checkpointId string) error {
	var checkpoint *cPoint

	for _, cp := range ctx.cpointData {
		if cp.id == checkpointId {
			checkpoint = &cp
			break
		}
	}

	if checkpoint == nil {
		err := fmt.Errorf("Checkpoint with id %v not found", checkpointId)
		logger.Error("%v", err)
		return err
	}

	logger.Info("restoring checkpoint %v", checkpoint)

	if ctx.checkpointMode == forkMode {
		restoreForkCheckpoint(ctx, *checkpoint)
	} else {
		restoreFileCheckpoint(ctx, *checkpoint)
	}

	logger.Debug("restoring registers state")

	err := syscall.PtraceSetRegs(ctx.pid, checkpoint.regs)
	must(err)

	logger.Debug("reverting breakpoints state")
	ctx.bpointData = checkpoint.bpoints

	// remove subsequent checkpoints
	// ctx.cpointData = ctx.cpointData[:cpIndex+1]

	logger.Debug("checkpoint restore finished")

	return nil
}

func restoreFileCheckpoint(ctx *processContext, checkpoint cPoint) {
	logger.Debug("restoring memory state: %v (file: %v) ", checkpoint.opName, checkpoint.file)

	readMemoryContentsFromFile(checkpoint)

	err := proc.WriteRegionsContentsToMemFile(ctx.pid, checkpoint.regions)
	must(err)
}

func restoreForkCheckpoint(ctx *processContext, checkpoint cPoint) {
	logger.Debug("restoring checkpoint: %v (pid %v)", checkpoint.opName, checkpoint.pid)

	logger.Debug("fetching memory locations from checkpoint")
	checkpointMemRegions := proc.GetForkCheckpointDataAddresses(ctx.pid, ctx.targetFile)

	logger.Debug("restoring memory state from checkpoint")
	for _, memRegion := range checkpointMemRegions {
		// logger.Debug("from checkpoint: %v", memRegion)
		data := memRegion.ContentsFromFile(checkpoint.pid)

		_, err := syscall.PtracePokeData(ctx.pid, uintptr(memRegion.Start), data)
		must(err)
	}

	for index, memRegion := range checkpoint.stackRegions {
		// logger.Debug("from stack: %v", memRegion)
		data := checkpoint.stackRawData[index]

		_, err := syscall.PtracePokeData(ctx.pid, uintptr(memRegion.Start), data)
		must(err)
	}
}

func createFileCheckpoint(ctx *processContext, opName string) cPoint {
	regs := getRegs(ctx, false)

	checkpointFile, err := os.CreateTemp(fmt.Sprintf("%v/temp", getExecutableDir()), fmt.Sprintf("%v-cp-*", filepath.Base(ctx.targetFile)))

	regions := proc.GetFileCheckpointDataAddresses(ctx.pid, ctx.targetFile)

	writeCheckpointToFile(ctx, checkpointFile, regions)

	must(err)

	checkpoint := cPoint{
		opName:  opName,
		regs:    regs,
		regions: regions,
		file:    checkpointFile.Name(),
		bpoints: make(breakpointData),
	}

	return checkpoint
}

func createForkCheckpoint(ctx *processContext, opName string) cPoint {
	regs := getRegs(ctx, false)

	stackMemRegions := proc.GetStackDataAddresses(ctx.pid)

	checkpoint := cPoint{
		pid:          int(getVariableFromMemory(ctx, "_MPI_CHECKPOINT_CHILD", true).(int32)),
		opName:       opName,
		regs:         regs,
		stackRegions: stackMemRegions,
		stackRawData: proc.ReadFromMemFileByRegions(ctx.pid, stackMemRegions),
		bpoints:      make(breakpointData),
	}

	return checkpoint
}

func writeCheckpointToFile(ctx *processContext, file *os.File, regions []proc.MemRegion) {

	contents := proc.ReadFromMemFileByRegions(ctx.pid, regions)

	for _, chunk := range contents {
		// logger.Debug("writing chunk %v to cp file - size %v", regions[index].Ident, len(chunk))
		file.Write(chunk)
	}

	file.Close()
}

func readMemoryContentsFromFile(checkpoint cPoint) {
	file, err := os.Open(checkpoint.file)
	must(err)

	reader := bufio.NewReader(file)

	for index, memRegion := range checkpoint.regions {

		buffer := make([]byte, memRegion.End-memRegion.Start)

		_, err := io.ReadFull(reader, buffer)
		must(err)

		checkpoint.regions[index].Contents = buffer
	}
}
