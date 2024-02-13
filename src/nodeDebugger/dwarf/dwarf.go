package dwarf

import (
	"fmt"
	"unsafe"
)

type DwarfData struct {
	Modules []*Module
	Types   typeMap
	Mpi     MPIData
}

func (m *Module) LookupFunc(functionName string) *Function {
	for _, function := range m.functions {
		if function.name == functionName {
			return function
		}
	}
	return nil
}

// Retrieve a function with a matching name
func (d *DwarfData) LookupFunc(functionName string) (module *Module, function *Function) {
	for _, module := range d.Modules {
		if function := module.LookupFunc(functionName); function != nil {
			return module, function
		}
	}
	return nil, nil
}

// Retrieve a variable with a matching identifier
func (d *DwarfData) LookupVariable(idendifier string) *Variable {
	for _, module := range d.Modules {
		for _, variable := range module.Variables {
			if variable.name == idendifier {
				return variable
			}
		}
	}

	return nil
}

// Retrieve a variable with a matching identifier only if defined in the supplied function
func (d *DwarfData) LookupVariableInFunction(function *Function, identifier string) *Variable {
	for _, module := range d.Modules {
		for _, variable := range module.Variables {
			if variable.name == identifier {
				if variable.Function != nil && variable.Function == function {
					return variable
				}
			}
		}
	}

	return nil
}

// Retrieve intruction entries for the function matching the identifier
func (d *DwarfData) GetEntriesForFunction(functionName string) []Entry {
	entries := make([]Entry, 0)
	module, function := d.LookupFunc(functionName)

	for _, entry := range module.entries {
		if entry.Address >= function.lowPC && entry.Address < function.highPC {
			entries = append(entries, entry)
		}
	}

	return entries
}

func (d *DwarfData) LineToPC(file string, line int) (address uint64, err error) {

	for _, module := range d.Modules {
		for _, moduleFile := range module.files {
			if moduleFile == file {
				for _, entry := range module.entries {
					if entry.line == line && module.files[entry.file] == file {
						if entry.isStmt {
							return entry.Address, nil
						}
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("unable to find suitable instruction for line %d in file %s", line, file)
}

func (d *DwarfData) PCToLine(pc uint64) (line int, file string, function *Function, err error) {
	for _, module := range d.Modules {
		if pc >= module.startAddress && pc <= module.endAddress {
			for _, entry := range module.entries {
				if entry.Address == pc {
					function := d.PCToFunc(pc)

					return entry.line, module.files[entry.file], function, nil
				}
			}
		}
	}
	return 0, "", nil, fmt.Errorf("unable to find instruction matching address %v", pc)
}

func (d *DwarfData) PCToFunc(pc uint64) *Function {
	// logger.Debug("pc to func %#x", pc)
	for _, module := range d.Modules {
		for _, function := range module.functions {
			if pc >= function.lowPC && pc < function.highPC {
				return function
			}
			// logger.Debug("func %v does not match", function)
		}
	}

	return nil
}

func (d DwarfData) FindEntrySourceFile(mainFn string) (sourceFile string) {

	module, function := d.LookupFunc(mainFn)

	sourceFile = module.files[function.file]

	return sourceFile
}

func (d *DwarfData) ResolveMPIDebugInfo() {
	mpiSignatureFunc := "_MPI_WRAPPER_INCLUDE"

	mpiWrapFunctions := make([]*Function, 0)

	module, sigFunc := d.LookupFunc(mpiSignatureFunc)

	for _, function := range module.functions {
		if function.file == sigFunc.file && function != sigFunc {
			function.name = function.name[1:]
			mpiWrapFunctions = append(mpiWrapFunctions, function)
		}
	}

	d.Mpi =
		MPIData{
			mpiWrapFunctions,
			module.files[sigFunc.file],
		}

}

// returns the pointer size of current arch
func ptrSize() int {
	return int(unsafe.Sizeof(uintptr(0)))
}
