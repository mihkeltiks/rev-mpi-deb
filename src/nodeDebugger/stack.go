package main

//lint:file-ignore U1000 ignore unused helpers

import (
	"encoding/binary"
	"fmt"
	"syscall"

	"github.com/ottmartens/cc-rev-db/nodeDebugger/dwarf"
)

type programStack []*stackFunction // the current call stack of the program

type stackFunction struct {
	function     *dwarf.Function // definition of the function
	baseAddress  uint64          // base address of the stack frame
	stackAddress uint64
}

func (stack programStack) String() string {
	str := ""
	for index, stackFunction := range stack {
		str = fmt.Sprintf("%s%v", str, stackFunction.function.Name())

		if index != len(stack)-1 {
			str = fmt.Sprintf("%s <- ", str)
		}
	}

	return str
}

func (sf stackFunction) lookupParameter(varName string) *dwarf.Parameter {
	for _, param := range sf.function.Parameters {
		if param.Name == varName {
			return param
		}
	}

	return nil
}

func (stack programStack) lookupFunction(fn *dwarf.Function) *stackFunction {
	for _, stackFn := range stack {
		if stackFn.function == fn {
			return stackFn
		}
	}

	return nil
}

func getStack(ctx *processContext) programStack {
	regs := getRegs(ctx, false)

	stackPointer := regs.Rsp
	basePointer := regs.Rbp

	var offset uint64

	ptrSize := uint64(ptrSize())

	fn := ctx.dwarfData.PCToFunc(regs.Rip)

	if fn == nil {
		return nil
	}

	fnStack := programStack{
		&stackFunction{
			function:     fn,
			baseAddress:  basePointer,
			stackAddress: stackPointer,
		},
	}

	for {
		offset = 0

		frameSize := basePointer - stackPointer + ptrSize

		if frameSize > 1024 || frameSize <= ptrSize {
			// logger.Debug("invalid base pointer or frame size")
			frameSize = 32
		}

		frameData := make([]byte, frameSize)
		_, err := syscall.PtracePeekData(ctx.pid, uintptr(stackPointer), frameData)
		must(err)

		// First instruction in frame - return address from stack frame
		stackContent := binary.LittleEndian.Uint64(frameData[:ptrSize])

		fn = ctx.dwarfData.PCToFunc(stackContent)

		if fn != nil {
			fnStack = append(fnStack, &stackFunction{function: fn, baseAddress: basePointer, stackAddress: stackPointer})
		}

		for offset = 0; offset < frameSize; offset += ptrSize {

			stackContent = binary.LittleEndian.Uint64(frameData[offset : offset+ptrSize])

			if offset == frameSize-ptrSize {
				basePointer = stackContent
				break
			}
		}

		// end of stack
		if fn.Name() == MAIN_FN {
			break
		}

		stackPointer += frameSize
	}

	return fnStack
}
