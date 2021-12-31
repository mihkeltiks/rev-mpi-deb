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

var interruptCode = []byte{0xCC}


func main() {

	targetFile := getValuesFromArgs()

	symTable := getSymbolTable(targetFile)

	sourceFile, _ := getSourceFileInfo(symTable) 	

	
	runBinary(targetFile, sourceFile, symTable)


}


func runBinary(target string, sourceFile string, symTable *gosym.Table) {
	var cmd *exec.Cmd
	
	cmd = exec.Command(target)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Ptrace: true,
	}

	cmd.Start()
	err := cmd.Wait()

	if err != nil {
		fmt.Printf("Returned from wait: %v\n", err)
	}

	pid := cmd.Process.Pid

	breakPointLine := 9
	breakpointAddress, _, _ := symTable.LineToPC(sourceFile, breakPointLine)

	// this works 
	file, line, fn := symTable.PCToLine(breakpointAddress)
	fmt.Printf("file: %v, line: %d, fn name: %v\n", file, line, fn.Name)


	// set breakpoint (insert interrup code at address where main function starts)
	syscall.PtracePokeData(pid, uintptr(breakpointAddress), interruptCode)
	syscall.PtraceCont(pid, 0)

	var ws syscall.WaitStatus

	for {
		syscall.Wait4(pid, &ws, 0, nil)

		if ws.StopSignal() == syscall.SIGTRAP && ws.TrapCause() != syscall.PTRACE_EVENT_CLONE {
			break
		} else { // received a signal other than trap/a trap from clone event
			syscall.PtraceCont(pid, 0)
		}
	}
	


	

	var regs syscall.PtraceRegs
	syscall.PtraceGetRegs(pid, &regs)

	filename, line, fn := symTable.PCToLine(regs.Rip)
	
	fmt.Printf("%s is at line %d in %s\n", fn.Name, line, filename)
}


func getSourceFileInfo(symTable *gosym.Table) (fileName string, mainFn *gosym.Func)  {
	mainFn = symTable.LookupFunc("main.main")

	fileName, _, _ = symTable.PCToLine(mainFn.Entry)

	return fileName, mainFn;
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




func printLineForPC(symTable *gosym.Table, pc uint64) string {
	var fileName string
	var line int
	
	fileName, line, _ = symTable.PCToLine(pc)

	fmt.Printf("Program counter address %X is in file %s at line %d\n", pc, fileName, line)

	return fileName
}



func printPCForLine(symTable *gosym.Table, fileName string, lineNr int) {
	var pc uint64
	var fn *gosym.Func

	pc, fn, _ = symTable.LineToPC(fileName, lineNr)

	var fnName string;
	if fn == nil {
		fnName = "<no function>"
	} else {
		fnName = fn.Name
	}


	fmt.Printf("In file %s at line %d there is PC address %X and function %s \n", fileName, lineNr, pc, fnName)
}
