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
	sourceFile    string         // source code file
	dwarfData     *dwarfData     // dwarf debug information about the binary
	process       *exec.Cmd      // the running binary
	pid           int            // the process id of the running binary
	checkpointPid int            // int
	bpointData    breakpointData // holds the instuctions for currently replaced by breakpoints
}

func main() {
	targetFile := getValuesFromArgs()

	ctx := &processContext{}

	ctx.dwarfData = getDwarfData(targetFile)

	ctx.sourceFile = getSourceFileInfo(ctx.dwarfData)
	ctx.bpointData = breakpointData{}.New()

	ctx.process = startBinary(targetFile)
	ctx.pid = ctx.process.Process.Pid

	insertMPIBreakpoints(ctx)

	printInstructions()

	for {
		// cmd := &command{cont, nil}
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
		cmd.Process.Kill()
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

	logger.Info("instruction pointer: %#x (line %d in %s)\n", regs.Rip, line, fileName)

	// data := make([]byte, 4)
	// syscall.PtracePeekData(ctx.pid, uintptr(regs.Rip), data)
	// logger.Info("ip pointing to: %v\n", data)
}

func getRegs(ctx *processContext, rewindIP bool) *syscall.PtraceRegs {
	var regs syscall.PtraceRegs

	err := syscall.PtraceGetRegs(ctx.pid, &regs)

	if err != nil {
		fmt.Printf("getregs error: %v\n", err)
	}

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
func getValuesFromArgs() string {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug <target binary>")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "mpi":
		logger.Info("mpi specified, loading example mpi binary")
		return "examples/hello_mpi_c/hello"
	case "c":
		logger.Info("c specified, loading example c binary")
		return "examples/hello_c/hello"
	case "go":
		logger.Info("go specified, loading example c binary")
		return "examples/hello_go/hello"
	}

	targetFilePath, err := filepath.Abs(os.Args[1])

	if err != nil {
		panic(err)
	}

	if _, err := os.Stat(targetFilePath); errors.Is(err, os.ErrNotExist) {
		panic(err) // file does not exist
	}

	return targetFilePath
}
