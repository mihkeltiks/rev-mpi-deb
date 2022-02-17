package main

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

type functionStack []*dwarfFunc

func (stack functionStack) String() string {
	str := ""
	for index, fn := range stack {
		str = fmt.Sprintf("%s%s", str, fn.name)

		if index != len(stack)-1 {
			str = fmt.Sprintf("%s <- ", str)
		}
	}

	return str
}

func getStack(ctx *processContext) functionStack {

	regs := getRegs(ctx, false)

	stackPointer := regs.Rsp
	basePointer := regs.Rbp

	var offset uint64

	ptrSize := uint64(ptrSize())

	fn := ctx.dwarfData.PCToFunc(regs.Rip)
	fnStack := []*dwarfFunc{fn}

	for {

		offset = 0

		frameSize := basePointer - stackPointer + ptrSize

		// logger.Info("func: %s", fn.name)
		// logger.Info("stack pointer: %x", stackPointer)
		// logger.Info("base pointer: %x", basePointer)
		// logger.Info("frame size: %d", frameSize)

		if frameSize > 1024 || basePointer < 1 {
			panic("invalid base pointer or frame size")
		}

		frameData := make([]byte, frameSize)
		_, err := syscall.PtracePeekData(ctx.pid, uintptr(stackPointer), frameData)
		must(err)

		// First instruction in frame - return address from stack frame
		content := binary.LittleEndian.Uint64(frameData[offset : offset+ptrSize])
		fn = ctx.dwarfData.PCToFunc(content)

		if fn != nil {
			fnStack = append(fnStack, fn)
		} else {
			panic("no matching function found for stack frame return address")
		}

		for offset = ptrSize; stackPointer+offset <= basePointer; offset += ptrSize {
			content := binary.LittleEndian.Uint64(frameData[offset : offset+ptrSize])

			// reached the end of the stack frame
			if stackPointer+offset == basePointer {
				basePointer = content
				break
			}
		}

		// end of stack
		if fn.name == MAIN_FN {
			break
		}

		stackPointer += frameSize
	}

	return fnStack
}
