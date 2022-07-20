package main

import (
	"fmt"
	"reflect"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
)

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
		logger.Warn("error getting registers, retrying in 1 sec")

		return nil, err

	}

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
