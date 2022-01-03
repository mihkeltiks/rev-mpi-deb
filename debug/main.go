package main

import (
	"debug/gosym"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type processContext struct {
	sourceFile string         // source code file
	symTable   *gosym.Table   // symbol table for the source code file
	process    *exec.Cmd      // the running binary
	pid        int            // the process id of the running binary
	bpointData *bpointDataMap // holds the instuctions currently replaced by breakpoints
}

type bpointDataMap map[int]*bpointData // keys - line numbers

type bpointData struct {
	address uint64 // address of the instruction
	data    []byte // actual contents of the instruction
}

// restores the original instruction if the executable
// is currently caught at a breakpoint
func (ctx *processContext) restoreCaughtBreakpoint() {
	line, _, _ := getCurrentLine(ctx)

	bpointData := (*ctx.bpointData)[line]

	if bpointData == nil {
		fmt.Printf("caughtAtBreakpoint false: %v, %v\n", bpointData, bpointData)
		return
	}

	fmt.Printf("caughtAtBreakpoint true: %v, %v\n", bpointData.address, bpointData.data)

	syscall.PtracePokeData(ctx.pid, uintptr(bpointData.address), bpointData.data)
}

func main() {

	targetFile := getValuesFromArgs()

	ctx := &processContext{}

	ctx.symTable = getSymbolTable(targetFile)
	ctx.sourceFile = getSourceFileInfo(ctx.symTable)

	ctx.process = startBinary(targetFile, ctx.sourceFile, ctx.symTable)

	ctx.pid = ctx.process.Process.Pid

	_bpointDataMap := make(bpointDataMap)
	ctx.bpointData = &_bpointDataMap

	for {
		cmd := askForInput()

		cmd.handle(ctx)

		if cmd.isProgressCommand() {
			ctx.restoreCaughtBreakpoint()

			logRegistersState(ctx)
		}
	}
}

func startBinary(target string, sourceFile string, symTable *gosym.Table) *exec.Cmd {

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
	}

	return cmd
}

func getSourceFileInfo(symTable *gosym.Table) (fileName string) {
	mainFn := symTable.LookupFunc("main.main")

	fileName, _, _ = symTable.PCToLine(mainFn.Entry)

	return fileName
}

func logRegistersState(ctx *processContext) {
	line, fileName, fnName := getCurrentLine(ctx)

	log.Default().Printf("instruction pointer: %s (line %d in %s)\n", fnName, line, fileName)
}

func getCurrentLine(ctx *processContext) (line int, fileName string, fnName string) {
	var regs syscall.PtraceRegs
	syscall.PtraceGetRegs(ctx.pid, &regs)

	fileName, line, fn := ctx.symTable.PCToLine(regs.Rip)

	if fn == nil {
		fnName = "<no function>"
	} else {
		fnName = fn.Name
	}

	return line, fileName, fnName
}

func getPCAddressForLine(symTable *gosym.Table, fileName string, lineNr int) uint64 {
	var pc uint64
	var err error

	pc, _, err = symTable.LineToPC(fileName, lineNr)

	if err != nil {
		panic(fmt.Sprintf("Could not get address for line %d: %v", lineNr, err))
	}

	return pc
}

// parse and validate command line arguments
func getValuesFromArgs() string {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug <target binary>")
		os.Exit(2)
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
