package main

import (
	"debug/gosym"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type processContext struct {
	sourceFile string       // source code file
	symTable   *gosym.Table // symbol table for the source code file
	cmd        *exec.Cmd    // the running binary
	pid        int          // the process id of the running binary
}

func main() {

	targetFile := getValuesFromArgs()

	ctx := processContext{}

	ctx.symTable = getSymbolTable(targetFile)
	ctx.sourceFile = getSourceFileInfo(ctx.symTable)

	ctx.cmd = startBinary(targetFile, ctx.sourceFile, ctx.symTable)
	ctx.pid = ctx.cmd.Process.Pid

	_ = setBreakPoint(ctx, 19)
	continueExecution(ctx)

	logRegistersState(ctx)
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
