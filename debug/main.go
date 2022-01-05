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
	dwarfData  *dwarfData     //
	symTable   *gosym.Table   // symbol table for the source code file
	process    *exec.Cmd      // the running binary
	pid        int            // the process id of the running binary
	bpointData *bpointDataMap // holds the instuctions for currently replaced by breakpoints
}

type bpointDataMap map[int]*bpointData // keys - line numbers

type bpointData struct {
	address uint64 // address of the instruction
	data    []byte // actual contents of the instruction
}

func main() {

	targetFile := getValuesFromArgs()

	ctx := &processContext{}

	ctx.dwarfData = getDwarfData(targetFile)

	// ctx.symTable = getSymbolTable(targetFile)

	ctx.sourceFile = getSourceFileInfo(ctx.dwarfData)

	ctx.process = startBinary(targetFile)

	ctx.pid = ctx.process.Process.Pid

	_bpointDataMap := make(bpointDataMap)
	ctx.bpointData = &_bpointDataMap

	var regs syscall.PtraceRegs

	syscall.PtraceGetRegs(ctx.pid, &regs)
	fmt.Printf("rip register at %x\n", regs.Rip)

	setBreakPoint(ctx, 24)
	continueExecution(ctx)

	syscall.PtraceGetRegs(ctx.pid, &regs)
	fmt.Printf("rip register at %x\n", regs.Rip)

	restoreCaughtBreakpoint(ctx)
	// continueExecution(ctx)

	//syscall.RawSyscall(syscall.SYS_PERSONALITY) try this to not have to -no-pie c files

	// for {
	// 	cmd := askForInput()

	// 	cmd.handle(ctx)

	// 	if cmd.isProgressCommand() {
	// 		ctx.restoreCaughtBreakpoint()

	// askForInput()
	// logRegistersState(ctx)
	// 	}
	// }

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
		log.Default().Println("binary started, waiting for continuation")
	}

	return cmd
}

func getSourceFileInfo(d *dwarfData) string {
	mainFunctionName := map[language]string{
		golang: "main.main",
		c:      "main",
	}[d.lang]

	module, function := d.lookupFunc(mainFunctionName)

	sourceFile := module.files[function.file]

	log.Default().Printf("entry source file: %v\n", sourceFile)

	return sourceFile
}

func logRegistersState(ctx *processContext) {
	line, fileName, fnName, _ := getCurrentLine(ctx)

	log.Default().Printf("instruction pointer: %s (line %d in %s)\n", fnName, line, fileName)
}

func getCurrentLine(ctx *processContext) (line int, fileName string, fnName string, err error) {
	var regs syscall.PtraceRegs
	syscall.PtraceGetRegs(ctx.pid, &regs)

	line, fileName, fnName, err = ctx.dwarfData.PCToLine(regs.Rip)

	return line, fileName, fnName, err
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
