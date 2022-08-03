package main

import (
	"fmt"
	"reflect"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/utils"
)

func logRegistersState(ctx *processContext) {
	regs := getRegs(ctx, false)

	line, fileName, _, _ := ctx.dwarfData.PCToLine(regs.Rip)

	logger.Debug("instruction pointer: %#x (line %d in %s)\n", regs.Rip, line, fileName)
}

func getRegs(ctx *processContext, rewindIP bool) *syscall.PtraceRegs {
	var regs syscall.PtraceRegs

	err := syscall.PtraceGetRegs(ctx.pid, &regs)

	if err != nil {
		logger.Error("error getting registers: %v", err)
		utils.Must(err)
	}

	// if currently stopped by a breakpoint, rewind the instruction pointer by 1
	// to find the correct instruction pointer location (rewind the interrupt instruction)
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
