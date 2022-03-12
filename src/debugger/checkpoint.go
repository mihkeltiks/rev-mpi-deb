package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

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
	opName string              // name of the mpi operation where checkpoint was made
	regs   *syscall.PtraceRegs // register values at checkpoint

	// fork mode
	pid          int              // process id of the fork at checkpoint
	stackRegions []proc.MemRegion // stack region addresses of checkpoint
	stackRawData [][]byte         // raw value of stack at checkpoint
	bpoints      breakpointData   // breakpoints at checkpoint time

	// file mode
	file    string           // file in which checkpoint data is stored
	regions []proc.MemRegion // descriptors of memory ranges
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

	if ctx.checkpointMode == forkMode {
		restoreForkCheckpoint(ctx, checkpoint)
	} else {
		restoreFileCheckpoint(ctx, checkpoint)
	}

	logger.Info("restoring registers state")
	err := syscall.PtraceSetRegs(ctx.pid, checkpoint.regs)
	time.Sleep(time.Millisecond * 10)
	must(err)

	logger.Info("reverting breakpoints state")
	ctx.bpointData = checkpoint.bpoints
	insertMPIBreakpoint(ctx, MPI_BPOINTS[MPI_FUNCS.RECORD], true)

	// remove subsequent checkpoints
	ctx.cpointData = ctx.cpointData[:cpIndex+1]

	logger.Info("checkpoint restore finished")

}

func restoreFileCheckpoint(ctx *processContext, checkpoint cPoint) {
	logger.Info("restoring memory state: %v (file: %v) ", checkpoint.opName, checkpoint.file)

	contents := readCheckpointFromFile(checkpoint)

	time.Sleep(time.Second / 10)

	PAGE_SIZE := os.Getpagesize()

	logger.Info("ctx.pid: %v", ctx.pid)

	for index, memRegion := range checkpoint.regions {

		data := contents[index]
		logger.Info("inserting chunk %v to memory - size %v", memRegion.Ident, len(data))

		var i int
		for i = 0; i < len(data); i += PAGE_SIZE {

			pageStart := int(memRegion.Start) + i
			if i == 0 {
				logger.Info("pageStart: %v", pageStart)
			}
			pageData := data[i : i+PAGE_SIZE]

			retryCount := 0

			for {
				time.Sleep(time.Millisecond * 10)
				byteCount, err := syscall.PtracePokeData(ctx.pid, uintptr(pageStart), pageData)

				if err == nil {

					break
				}

				logger.Info("wrote %v < %v (page %d) idx %v:%v", byteCount, PAGE_SIZE, i/PAGE_SIZE, index, i)
				msg, errors := syscall.PtraceGetEventMsg(ctx.pid)

				if msg == 0 {
					syscall.Wait4(ctx.pid, nil, 0, nil)
				}

				logger.Info("event msg %v, %v", msg, errors)

				retryCount++

				if retryCount > 5 {
					logger.Info("count exceeded")
					time.Sleep(time.Second * 10)
					must(err)
				}
			}

		}

		logger.Info("wrote %d pages", i/PAGE_SIZE)

	}
}

func restoreForkCheckpoint(ctx *processContext, checkpoint cPoint) {
	logger.Info("restoring checkpoint: %v (pid %v)", checkpoint.opName, checkpoint.pid)

	logger.Info("fetching memory locations from checkpoint")
	checkpointMemRegions := proc.GetForkCheckpointDataAddresses(ctx.pid, ctx.targetFile)

	logger.Info("restoring memory state from checkpoint")
	for _, memRegion := range checkpointMemRegions {
		// logger.Info("from checkpoint: %v", memRegion)
		data := memRegion.Contents(checkpoint.pid)

		_, err := syscall.PtracePokeData(ctx.pid, uintptr(memRegion.Start), data)
		must(err)
	}

	for index, memRegion := range checkpoint.stackRegions {
		// logger.Info("from stack: %v", memRegion)
		data := checkpoint.stackRawData[index]

		_, err := syscall.PtracePokeData(ctx.pid, uintptr(memRegion.Start), data)
		must(err)
	}
}

func createCheckpoint(ctx *processContext, opName string) {

	var checkpoint cPoint

	if ctx.checkpointMode == fileMode {
		checkpoint = createFileCheckpoint(ctx, opName)
	} else {
		checkpoint = createForkCheckpoint(ctx, opName)
	}

	for address, bp := range ctx.bpointData {
		checkpoint.bpoints[address] = &bpointData{
			bp.address,
			bp.originalInstruction,
			bp.function,
			bp.isMPIBpoint,
			false,
		}
	}

	ctx.cpointData = append(ctx.cpointData, checkpoint)
}

func createFileCheckpoint(ctx *processContext, opName string) cPoint {
	regs := getRegs(ctx, false)

	checkpointFile, err := os.CreateTemp("bin/temp", fmt.Sprintf("%v-cp-*", filepath.Base(ctx.targetFile)))

	regions := proc.GetFileCheckpointDataAddresses(ctx.pid, ctx.targetFile)

	logger.Info("regions:\n%v", regions)

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
	checkpoint := cPoint{
		pid:          int(getVariableFromMemory(ctx, "_MPI_CHECKPOINT_CHILD").(int32)),
		opName:       opName,
		regs:         getRegs(ctx, false),
		stackRegions: proc.GetStackDataAddresses(ctx.pid),
		bpoints:      make(breakpointData),
	}

	checkpoint.stackRawData = proc.ReadFromMemFileByRegions(ctx.pid, checkpoint.stackRegions)

	return checkpoint
}

func writeCheckpointToFile(ctx *processContext, file *os.File, regions []proc.MemRegion) {

	contents := proc.ReadFromMemFileByRegions(ctx.pid, regions)

	for index, chunk := range contents {
		logger.Info("writing chunk %v to cp file - size %v", regions[index].Ident, len(chunk))
		file.Write(chunk)
	}

	file.Close()
}

func readCheckpointFromFile(checkpoint cPoint) [][]byte {

	file, err := os.Open(checkpoint.file)
	must(err)

	reader := bufio.NewReader(file)

	contents := make([][]byte, 0)

	for _, memRegion := range checkpoint.regions {
		chunk := make([]byte, memRegion.End-memRegion.Start)

		_, err := io.ReadFull(reader, chunk)
		must(err)

		contents = append(contents, chunk)
	}

	return contents
}
