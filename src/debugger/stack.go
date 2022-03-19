package main

import (
	"encoding/binary"
	"fmt"
	"syscall"

	"github.com/ottmartens/cc-rev-db/logger"
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

func getStack(ctx *processContext, bpoint *bpointData) functionStack {

	regs := getRegs(ctx, false)

	stackPointer := regs.Rsp
	basePointer := regs.Rbp

	var offset uint64

	ptrSize := uint64(ptrSize())
	// logger.Debug("bpoint: %v", bpoint)
	// logger.Debug("ip: %#x", regs.Rip)
	// logger.Debug("sp: %#x", regs.Rsp)
	// logger.Debug("bp: %#x", regs.Rbp)
	fn := ctx.dwarfData.PCToFunc(regs.Rip)
	fnStack := []*dwarfFunc{fn}

	for {
		// logger.Debug("func: %s", fn.name)

		offset = 0

		frameSize := basePointer - stackPointer + ptrSize

		// logger.Debug("stack pointer: %#x", stackPointer)
		// logger.Debug("base pointer: %#x", basePointer)
		// logger.Debug("frame size: %d", frameSize)

		// logger.Debug("frame size: %v", frameSize)

		if frameSize > 1024 || frameSize <= ptrSize {
			logger.Debug("invalid base pointer or frame size")
			frameSize = 32
		}

		frameData := make([]byte, frameSize)
		_, err := syscall.PtracePeekData(ctx.pid, uintptr(stackPointer), frameData)
		must(err)

		// First instruction in frame - return address from stack frame
		content := binary.LittleEndian.Uint64(frameData[:ptrSize])
		fn = ctx.dwarfData.PCToFunc(content)

		if fn != nil {
			fnStack = append(fnStack, fn)
		} else {
			logger.Debug("stack return address fallback")
			// break
			content := binary.LittleEndian.Uint64(frameData[ptrSize : 2*ptrSize])
			fn = ctx.dwarfData.PCToFunc(content)

			if fn != nil {
				fnStack = append(fnStack, fn)
			} else {
				logger.Debug("no matching function found for stack frame return address")
				break
			}

		}

		for offset = 0; offset < frameSize; offset += ptrSize {

			content = binary.LittleEndian.Uint64(frameData[offset : offset+ptrSize])
			// _fn := ctx.dwarfData.PCToFunc(content)

			// logger.Debug("content at offset %d : %#x matching func: %v", offset, content, _fn)
			// reached the end of the stack frame
			if offset == frameSize-ptrSize {
				logger.Debug("end of frame")
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
