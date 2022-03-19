package main

//lint:file-ignore U1000 ignore unused helpers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"

	"github.com/ottmartens/cc-rev-db/dwarf"
	"github.com/ottmartens/cc-rev-db/logger"
)

const MAIN_FN = "main"

type processContext struct {
	targetFile     string           // the executing binary file
	sourceFile     string           // source code file
	dwarfData      *dwarf.DwarfData // dwarf debug information about the binary
	process        *exec.Cmd        // the running binary
	pid            int              // the process id of the running binary
	bpointData     breakpointData   // holds the instuctions for currently replaced by breakpoints
	cpointData     checkpointData   // holds data about currently recorded checkppoints
	checkpointMode CheckpointMode   // whether checkpoints are recorded in files or in forked processes
}

func main() {
	// logger.SetMaxLogLevel(logger.Levels.Info)

	defer cleanup()
	precleanup()

	targetFile, checkpointMode := getValuesFromArgs()

	ctx := &processContext{
		targetFile:     targetFile,
		checkpointMode: checkpointMode,
		bpointData:     breakpointData{}.New(),
		cpointData:     checkpointData{}.New(),
	}

	ctx.dwarfData = dwarf.ParseDwarfData(ctx.targetFile)
	ctx.dwarfData.ResolveMPIDebugInfo(MPI_FUNCS.SIGNATURE)

	ctx.sourceFile = ctx.dwarfData.FindEntrySourceFile(MAIN_FN)

	ctx.process = startBinary(ctx.targetFile)
	ctx.pid = ctx.process.Process.Pid

	insertMPIBreakpoints(ctx)

	// printInstructions()

	for {
		cmd := askForInput()

		res := cmd.handle(ctx)

		if res.exited { // binary exited
			break
		}
	}

}

func startBinary(target string) *exec.Cmd {

	cmd := exec.Command(target)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Ptrace: true,
	}

	cmd.Start()
	err := cmd.Wait()

	if err != nil {
		// arrived at auto-inserted initial breakpoint trap
		logger.Debug("child: %v", err)
		logger.Info("binary started, waiting for command")
	}

	return cmd
}

func logRegistersState(ctx *processContext) {
	regs, _ := getRegs(ctx, false)

	line, fileName, _, _ := ctx.dwarfData.PCToLine(regs.Rip)

	logger.Debug("instruction pointer: %#x (line %d in %s)\n", regs.Rip, line, fileName)

	// data := make([]byte, 4)
	// syscall.PtracePeekData(ctx.pid, uintptr(regs.Rip), data)
	// logger.Debug("ip pointing to: %v\n", data)
}

func getRegs(ctx *processContext, rewindIP bool) (*syscall.PtraceRegs, error) {
	var regs syscall.PtraceRegs

	err := syscall.PtraceGetRegs(ctx.pid, &regs)

	if err != nil {
		logger.Warn("error getting registers: %v", err)

		return nil, err
	}

	// if err != nil {
	// 	fmt.Printf("getregs error: %v\n\n\n", err)

	// 	logger.Debug("sending signal")

	// 	ctx.process.Process.Signal(syscall.Signal(syscall.SIGCONT))

	// 	logger.Debug("waiting")

	// 	syscall.Wait4(ctx.pid, nil, 0, nil)
	// 	logger.Debug("getting regs again")

	// 	err := syscall.PtraceGetRegs(ctx.pid, &regs)

	// 	if err != nil {
	// 		panic(err)
	// 	}

	// }

	// if currently stopped by a breakpoint, rewind the instruction pointer by 1
	// to find the correct instruction (rewind the interrupt instruction)
	if rewindIP {
		regs.Rip -= 1
	}

	return &regs, nil
}

func printRegs(ctx *processContext) {
	regs, err := getRegs(ctx, false)
	must(err)

	s := reflect.ValueOf(regs).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fmt.Printf(" %s = %#x\n", typeOfT.Field(i).Name, f.Interface())
	}
}

// parse and validate command line arguments
func getValuesFromArgs() (targetFilePath string, checkpointMode CheckpointMode) {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug <target binary> <checkpoint mode (file|fork)")
		os.Exit(2)
	}

	var err error

	switch os.Args[1] {

	case "hello":
		logger.Info("loading example mpi hello binary")
		targetFilePath, err = filepath.Abs("bin/targets/hello")
	default:
		targetFilePath, err = filepath.Abs(os.Args[1])
	}

	must(err)

	targetFilePath, err = filepath.EvalSymlinks(targetFilePath)

	must(err)

	if _, err := os.Stat(targetFilePath); errors.Is(err, os.ErrNotExist) {
		panic(err) // file does not exist
	}

	if len(os.Args) == 3 && os.Args[2] == "fork" {
		checkpointMode = forkMode
		logger.Info("Checkpoint mode: fork")
	} else {
		checkpointMode = fileMode
		logger.Info("Checkpoint mode: file")
	}

	return targetFilePath, checkpointMode
}
