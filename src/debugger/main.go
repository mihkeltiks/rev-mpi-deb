package main

//lint:file-ignore U1000 ignore unused helpers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
)

type processContext struct {
	targetFile     string         // the executing binary file
	sourceFile     string         // source code file
	dwarfData      *dwarfData     // dwarf debug information about the binary
	process        *exec.Cmd      // the running binary
	pid            int            // the process id of the running binary
	bpointData     breakpointData // holds the instuctions for currently replaced by breakpoints
	cpointData     checkpointData // holds data about currently recorded checkppoints
	checkpointMode CheckpointMode // whether checkpoints are recorded in files or in forked processes
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

	ctx.dwarfData = getDwarfData(ctx.targetFile)

	ctx.sourceFile = getSourceFileInfo(ctx.dwarfData)

	ctx.process = startBinary(ctx.targetFile)
	ctx.pid = ctx.process.Process.Pid

	insertMPIBreakpoints(ctx)

	// printInstructions()

	// time.Sleep(time.Millisecond * 500)

	// (&command{bpoint, 26}).handle(ctx)

	// (&command{cont, nil}).handle(ctx)
	// time.Sleep(time.Millisecond * 100)
	// (&command{restore, 0}).handle(ctx)

	// time.Sleep(time.Millisecond * 500)
	// (&command{cont, nil}).handle(ctx)

	// time.Sleep(time.Millisecond * 500)
	// (&command{cont, nil}).handle(ctx)

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

	// handle termination of child on exit
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		os.Exit(1)
	}()

	cmd.Start()
	err := cmd.Wait()

	if err != nil {
		// arrived at auto-inserted initial breakpoint trap
		logger.Info("binary started, waiting for command")
	}

	return cmd
}

func getSourceFileInfo(d *dwarfData) (sourceFile string) {

	module, function := d.lookupFunc(MAIN_FN)

	sourceFile = module.files[function.file]

	return sourceFile
}

func logRegistersState(ctx *processContext) {
	regs := getRegs(ctx, false)

	line, fileName, _, _ := ctx.dwarfData.PCToLine(regs.Rip)

	logger.Debug("instruction pointer: %#x (line %d in %s)\n", regs.Rip, line, fileName)

	// data := make([]byte, 4)
	// syscall.PtracePeekData(ctx.pid, uintptr(regs.Rip), data)
	// logger.Debug("ip pointing to: %v\n", data)
}

func getRegs(ctx *processContext, rewindIP bool) *syscall.PtraceRegs {
	var regs syscall.PtraceRegs

	err := syscall.PtraceGetRegs(ctx.pid, &regs)

	if err != nil {
		logger.Warn("error getting registers")
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

	return &regs
}

func printRegs(ctx *processContext) {
	regs := getRegs(ctx, false)

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
