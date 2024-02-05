package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/checkpoint-restore/go-criu/v7"
	"github.com/checkpoint-restore/go-criu/v7/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/nodeDebugger/proc"
	"github.com/mihkeltiks/rev-mpi-deb/utils"
	"google.golang.org/protobuf/proto"
)

type CheckpointMode int

const (
	CRIUMode CheckpointMode = iota
	fileMode
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
	return fmt.Sprintf("{%s - %s}", cp.id, cp.opName)
}

func createCheckpoint(ctx *processContext, opName string) string {

	logger.Verbose("creating new checkpoint (%v)", opName)

	var checkpoint cPoint

	if ctx.checkpointMode == CRIUMode {
		logger.Debug("CRIUMode checkpoint")
		checkpoint = createCRIUCheckpoint(ctx, opName)
	} else if ctx.checkpointMode == forkMode {
		checkpoint = createForkCheckpoint(ctx, opName)
	} else {
		checkpoint = createFileCheckpoint(ctx, opName)
	}

	checkpoint.id = utils.RandomId()

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
	var checkpointIndex int

	for index, cp := range ctx.cpointData {
		if cp.id == checkpointId {
			checkpoint = &cp
			checkpointIndex = index
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
	utils.Must(err)

	logger.Debug("reverting breakpoints state")
	ctx.bpointData = checkpoint.bpoints

	// remove subsequent checkpoints
	ctx.cpointData = ctx.cpointData[:checkpointIndex+1]

	logger.Debug("checkpoint restore finished")

	return nil
}

func restoreFileCheckpoint(ctx *processContext, checkpoint cPoint) {
	logger.Debug("restoring memory state: %v (file: %v) ", checkpoint.opName, checkpoint.file)

	readMemoryContentsFromFile(checkpoint)

	err := proc.WriteRegionsContentsToMemFile(ctx.pid, checkpoint.regions)
	utils.Must(err)
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
		utils.Must(err)
	}

	for index, memRegion := range checkpoint.stackRegions {
		// logger.Debug("from stack: %v", memRegion)
		data := checkpoint.stackRawData[index]

		_, err := syscall.PtracePokeData(ctx.pid, uintptr(memRegion.Start), data)
		utils.Must(err)
	}
}

func createCRIUCheckpoint(ctx *processContext, opName string) cPoint {
	logger.Debug("Executing CRIU checkpoint on: %v", ctx.pid)
	c := criu.MakeCriu()

	checkpointDir, err := os.MkdirTemp(fmt.Sprintf("%v/temp", utils.GetExecutableDir()), fmt.Sprintf("%s-cp-*", filepath.Base(ctx.targetFile)))
	if err != nil {
		logger.Error("Error creating folder, %v", err)
	}
	logger.Debug("Saving checkpoint into: %v", checkpointDir)

	err = syscall.PtraceDetach(ctx.pid)
	if err != nil {
		logger.Debug("Error detaching from process: %v", err)
	}
	// Calls CRIU, saves process data to checkpointDir
	Dump(c, strconv.Itoa(ctx.pid), false, checkpointDir, "")

	err = syscall.PtraceAttach(ctx.pid)
	if err != nil {
		logger.Debug("Error attaching to process: %v", err)
	}
	checkpoint := cPoint{
		opName:  opName,
		file:    checkpointDir,
		bpoints: make(breakpointData),
	}

	return checkpoint
}

func createFileCheckpoint(ctx *processContext, opName string) cPoint {
	regs := getRegs(ctx, false)

	checkpointFile, err := os.CreateTemp(fmt.Sprintf("%v/temp", utils.GetExecutableDir()), fmt.Sprintf("%v-cp-*", filepath.Base(ctx.targetFile)))

	regions := proc.GetFileCheckpointDataAddresses(ctx.pid, ctx.targetFile)

	writeCheckpointToFile(ctx, checkpointFile, regions)

	utils.Must(err)

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
	utils.Must(err)

	reader := bufio.NewReader(file)

	for index, memRegion := range checkpoint.regions {

		buffer := make([]byte, memRegion.End-memRegion.Start)

		_, err := io.ReadFull(reader, buffer)
		utils.Must(err)

		checkpoint.regions[index].Contents = buffer
	}
}

func Dump(c *criu.Criu, pidS string, pre bool, imgDir string, prevImg string) {
	pid, err := strconv.ParseInt(pidS, 10, 32)
	if err != nil {
		logger.Error("Can't parse pid: %v", err)
	}
	img, err := os.Open(imgDir)
	if err != nil {
		logger.Error("Can't open image dir: %v", err)
	}

	opts := &rpc.CriuOpts{
		Pid:            proto.Int32(int32(pid)),
		ImagesDirFd:    proto.Int32(int32(img.Fd())),
		LogLevel:       proto.Int32(4),
		ShellJob:       proto.Bool(true),
		LogToStderr:    proto.Bool(true),
		LeaveRunning:   proto.Bool(true),
		LogFile:        proto.String("dump.log"),
		ExtUnixSk:      proto.Bool(true),
		TcpEstablished: proto.Bool(true),
	}

	if prevImg != "" {
		opts.ParentImg = proto.String(prevImg)
		opts.TrackMem = proto.Bool(true)
		time.Sleep(5 * time.Second)
	}

	if pre {
		err = c.PreDump(opts, TestNfy{})
	} else {
		err = c.Dump(opts, TestNfy{})
	}

	if err != nil {
		logger.Error("CRIU error during checkpoint: %v", err)
	}
	img.Close()
}

type TestNfy struct {
	criu.NoNotify
}
