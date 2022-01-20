package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	Logger "github.com/ottmartens/cc-rev-db/logger"
)

type processContext struct {
	sourceFile string         // source code file
	dwarfData  *dwarfData     //
	process    *exec.Cmd      // the running binary
	pid        int            // the process id of the running binary
	bpointData *bpointDataMap // holds the instuctions for currently replaced by breakpoints
	lang
}

type lang string

const (
	golang lang = "go"
	c      lang = "c"
)

type bpointDataMap map[int]*bpointData // keys - line numbers

type bpointData struct {
	address uint64 // address of the instruction
	data    []byte // actual contents of the instruction
}

func main() {
	targetFile := getValuesFromArgs()

	ctx := &processContext{}

	ctx.dwarfData = getDwarfData(targetFile)

	ctx.sourceFile, ctx.lang = getSourceFileInfo(ctx.dwarfData)

	ctx.process = startBinary(targetFile)

	ctx.pid = ctx.process.Process.Pid

	_bpointDataMap := make(bpointDataMap)
	ctx.bpointData = &_bpointDataMap

	printInstructions()

	for {
		cmd := askForInput()

		res := cmd.handle(ctx)

		if res.exited { // binary exited
			break
		}

		if cmd.isProgressCommand() {
			restoreCaughtBreakpoint(ctx)
			logRegistersState(ctx)
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
		Logger.Info("binary started, waiting for continuation")
	}

	return cmd
}

func getSourceFileInfo(d *dwarfData) (sourceFile string, language lang) {
	languageEntryFuncs := map[lang]string{
		golang: "main.main",
		c:      "main",
	}

	module, function := d.lookupFunc(languageEntryFuncs[golang])
	if module != nil {
		language = golang
	} else {
		module, function = d.lookupFunc(languageEntryFuncs[c])
		language = c
	}

	sourceFile = module.files[function.file]

	return sourceFile, language
}

func logRegistersState(ctx *processContext) {
	registers, line, fileName, _, _ := getCurrentLine(ctx, false)

	Logger.Info("instruction pointer: %x (line %d in %s)\n", registers.Rip, line, fileName)

	data := make([]byte, 4)
	syscall.PtracePeekData(ctx.pid, uintptr(registers.Rip), data)
	Logger.Info("ip pointing to: %v\n", data)
}

func getCurrentLine(ctx *processContext, rewindIP bool) (registers *syscall.PtraceRegs, line int, fileName string, fnName string, err error) {
	var regs syscall.PtraceRegs

	err = syscall.PtraceGetRegs(ctx.pid, &regs)

	if err != nil {
		fmt.Printf("getregs error: %v\n", err)
	}

	// if currently stopped by a breakpoint, rewind the instruction pointer by 1
	// to find the correct instruction
	if rewindIP {
		regs.Rip -= 1
	}

	line, fileName, fnName, err = ctx.dwarfData.PCToLine(regs.Rip)

	return &regs, line, fileName, fnName, err
}

// parse and validate command line arguments
func getValuesFromArgs() string {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug <target binary>")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "c":
		Logger.Info("c specified, loading example c binary")
		return "examples/hello_c/hello"
	case "go":
		Logger.Info("go specified, loading example c binary")
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
