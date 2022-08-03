package dwarf

//lint:file-ignore U1000 ignore unused helpers

// initial implementation from delve debugger https://github.com/go-delve/delve

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Opcode byte

type stackfn func(Opcode, *context) error

type ReadMemoryFunc func([]byte, uint64) (int, error)

type context struct {
	buf     *bytes.Buffer
	prog    []byte
	stack   []int64
	pieces  []Piece
	ptrSize int

	DwarfRegisters
	readMemory ReadMemoryFunc
}

// Piece is a piece of memory stored either at an address or in a register.
type Piece struct {
	Size  int
	Kind  PieceKind
	Val   uint64
	Bytes []byte
}

type PieceKind uint8

type DwarfRegisters struct {
	StaticBase uint64

	CFA       int64
	FrameBase int64
	ObjBase   int64
	regs      []*DwarfRegister

	ByteOrder binary.ByteOrder
	PCRegNum  uint64
	SPRegNum  uint64
	BPRegNum  uint64
	LRRegNum  uint64

	loadMoreCallback func()
}

type DwarfRegister struct {
	Uint64Val uint64
	Bytes     []byte
}

const arbitraryExecutionLimitFactor = 10

func ExecuteStackProgram(regs DwarfRegisters, instructions []byte, ptrSize int, readMemory ReadMemoryFunc) (int64, []Piece, error) {
	ctxt := &context{
		buf:            bytes.NewBuffer(instructions),
		prog:           instructions,
		stack:          make([]int64, 0, 3),
		DwarfRegisters: regs,
		ptrSize:        ptrSize,
	}

	for tick := 0; tick < len(instructions)*arbitraryExecutionLimitFactor; tick++ {
		opcodeByte, err := ctxt.buf.ReadByte()
		if err != nil {
			break
		}
		opcode := Opcode(opcodeByte)

		// logger.Debug("executing %s", opcodeName[opcode])

		if opcode == DW_OP_nop {
			continue
		}
		fn, ok := oplut[opcode]
		if !ok {
			return 0, nil, fmt.Errorf("invalid instruction %#v", opcode)
		}

		err = fn(opcode, ctxt)
		if err != nil {
			return 0, nil, err
		}
	}

	if ctxt.pieces != nil {
		return 0, nil, fmt.Errorf("support for pieced instructions not implemented")
	}

	if len(ctxt.stack) == 0 {
		return 0, nil, errors.New("empty OP stack")
	}

	return ctxt.stack[len(ctxt.stack)-1], nil, nil
}

func addr(opcode Opcode, ctxt *context) error {
	buf := ctxt.buf.Next(ctxt.ptrSize)
	stack, err := ReadUintRaw(bytes.NewReader(buf), binary.LittleEndian, ctxt.ptrSize)
	if err != nil {
		return err
	}
	ctxt.stack = append(ctxt.stack, int64(stack+ctxt.StaticBase))
	return nil
}

func callframecfa(opcode Opcode, ctxt *context) error {
	if ctxt.CFA == 0 {
		return errors.New("could not retrieve CFA for current PC")
	}
	ctxt.stack = append(ctxt.stack, int64(ctxt.CFA))
	return nil
}

func framebase(opcode Opcode, ctxt *context) error {
	num, _ := DecodeSLEB128(ctxt.buf)
	ctxt.stack = append(ctxt.stack, ctxt.FrameBase+num)
	return nil
}

const (
	DW_OP_addr           Opcode = 0x03
	DW_OP_fbreg          Opcode = 0x91
	DW_OP_nop            Opcode = 0x96
	DW_OP_call_frame_cfa Opcode = 0x9c
)

var opcodeName = map[Opcode]string{
	DW_OP_addr:           "DW_OP_addr",
	DW_OP_fbreg:          "DW_OP_fbreg",
	DW_OP_call_frame_cfa: "DW_OP_call_frame_cfa",
}
var opcodeArgs = map[Opcode]string{
	DW_OP_addr:           "8",
	DW_OP_fbreg:          "s",
	DW_OP_call_frame_cfa: "",
}

var oplut = map[Opcode]stackfn{
	DW_OP_addr: addr,

	DW_OP_fbreg: framebase,

	DW_OP_call_frame_cfa: callframecfa,
}

type ByteReaderWithLen interface {
	io.ByteReader
	io.Reader
	Len() int
}

// DecodeSLEB128 decodes a signed Little Endian Base 128
// represented number.

func DecodeSLEB128(buf ByteReaderWithLen) (int64, uint32) {
	var (
		b      byte
		err    error
		result int64
		shift  uint64
		length uint32
	)

	if buf.Len() == 0 {
		return 0, 0
	}

	for {
		b, err = buf.ReadByte()
		if err != nil {
			panic("Could not parse SLEB128 value")
		}
		length++

		result |= int64((int64(b) & 0x7f) << shift)
		shift += 7
		if b&0x80 == 0 {
			break
		}
	}

	if (shift < 8*uint64(length)) && (b&0x40 > 0) {
		result |= -(1 << shift)
	}

	return result, length
}

func ReadUintRaw(reader io.Reader, order binary.ByteOrder, ptrSize int) (uint64, error) {
	switch ptrSize {
	case 2:
		var n uint16
		if err := binary.Read(reader, order, &n); err != nil {
			return 0, err
		}
		return uint64(n), nil
	case 4:
		var n uint32
		if err := binary.Read(reader, order, &n); err != nil {
			return 0, err
		}
		return uint64(n), nil
	case 8:
		var n uint64
		if err := binary.Read(reader, order, &n); err != nil {
			return 0, err
		}
		return n, nil
	}
	return 0, fmt.Errorf("pointer size %d not supported", ptrSize)
}
