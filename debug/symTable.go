package main

import (
	"debug/elf"
	"debug/gosym"
)

// Assembles the symbol table from the target binary using the go builtin elf utility.
func getSymbolTable(program string) *gosym.Table {
	elfFile, err := elf.Open(program)

	if err != nil {
		panic(err)
	}

	defer elfFile.Close()

	symTableData, err := elfFile.Section(".gosymtab").Data()
	if err != nil {
		panic(err)
	}

	address := elfFile.Section(".text").Addr

	lineTableData, err := elfFile.Section(".gopclntab").Data()
	if err != nil {
		panic(err)
	}

	lineTable := gosym.NewLineTable(lineTableData, address)
	if err != nil {
		panic(err)
	}

	symTable, err := gosym.NewTable(symTableData, lineTable)
	if err != nil {
		panic(err)
	}

	return symTable
}

func getLineForPC(symTable *gosym.Table, pc uint64) (fileName string, line int) {
	fileName, line, _ = symTable.PCToLine(pc)

	return fileName, line
}

func getPCForLine(symTable *gosym.Table, fileName string, lineNr int) (pc uint64, fn *gosym.Func) {
	pc, fn, err := symTable.LineToPC(fileName, lineNr)

	if err != nil {
		panic(err)
	}
	return pc, fn
}
