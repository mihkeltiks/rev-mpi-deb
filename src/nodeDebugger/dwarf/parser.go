package dwarf

import (
	"debug/dwarf"
	"debug/elf"
	"io"
	"reflect"
    "fmt"
)

func ParseDwarfData(targetFile string) *DwarfData {

	data := &DwarfData{
		Modules: make([]*Module, 0),
		Types:   make(typeMap),
	}

	var currentModule *Module
	var currentFunction *Function

	elfFile, err := elf.Open(targetFile)
	if err != nil {
		panic(err)
	}
	dwarfRawData, err := elfFile.DWARF()
	if err != nil {
		panic(err)
	}
	reader := dwarfRawData.Reader()

	for {
		entry, err := reader.Next()

		if err == io.EOF || entry == nil {
			break
		}
		if err != nil {
			panic(err)
		}

		switch entry.Tag {

		// base type declaration
		case dwarf.TagBaseType:
			data.Types[entry.Offset] = &BaseType{
				name:     entry.Val(dwarf.AttrName).(string),
				byteSize: entry.Val(dwarf.AttrByteSize).(int64),
				encoding: entry.Val(dwarf.AttrEncoding).(int64),
			}

		// entering a new module
		case dwarf.TagCompileUnit:
			currentModule = parseModule(entry, dwarfRawData)

			data.Modules = append(data.Modules, currentModule)

			currentFunction = nil

		// function declaration
		case dwarf.TagSubprogram:
			currentFunction = parseFunction(entry, dwarfRawData)

			currentModule.functions = append(currentModule.functions, currentFunction)

		case dwarf.TagFormalParameter:
			parameter := parseFunctionParameter(entry, data)

			currentFunction.Parameters = append(currentFunction.Parameters, parameter)

		// variable declaration
		case dwarf.TagVariable:
			baseType := data.Types[entry.Val(dwarf.AttrType).(dwarf.Offset)]

			if baseType == nil {
				baseType = &BaseType{
					name: "unknown type",
				}
			}

			variable := &Variable{
				name:     entry.Val(dwarf.AttrName).(string),
				baseType: baseType,
				Function: currentFunction,
			}

			locationInstructions := entry.Val(dwarf.AttrLocation)

			if reflect.TypeOf(locationInstructions) != nil {
				variable.locationInstructions = entry.Val(dwarf.AttrLocation).([]byte)
			}

			currentModule.Variables = append(currentModule.Variables, variable)

		default:
			// logger.Debug("unhandled tag type: %v", entry.Tag)
		}

	}

	return data
}

func parseFunctionParameter(entry *dwarf.Entry, data *DwarfData) *Parameter {

	baseType := data.Types[(entry.Val(dwarf.AttrType).(dwarf.Offset))]

	if baseType == nil {
		baseType = &BaseType{
			name: "unknown type",
		}
	}
	name := entry.Val(dwarf.AttrName)
	if name == nil {  
		return nil
	}
	parameter := &Parameter{
		Name:				  entry.Val(dwarf.AttrName). (string),
		baseType:             baseType,
		locationInstructions: entry.Val(dwarf.AttrLocation).([]byte),
	}
	
	return parameter
}

func parseFunction(entry *dwarf.Entry, dwarfRawData *dwarf.Data) *Function {
	function := Function{}

	for _, field := range entry.Field {
		switch field.Attr {
		case dwarf.AttrName:
			function.name = field.Val.(string)
		case dwarf.AttrDeclFile:
			function.file = int(field.Val.(int64))
		case dwarf.AttrDeclLine:
			function.line = field.Val.(int64)
			// adjust for inserted line
			function.line--
		case dwarf.AttrDeclColumn:
			function.col = field.Val.(int64)
		case dwarf.AttrFrameBase:

			fmt.Printf("frame base : %v, %v, %T, %x\n", field.Attr, field.Val, field.Val, field.Val)

			// buf := new(bytes.Buffer)
			// op.PrettyPrint(buf, field.Val.([]byte))
			// fmt.Println(buf.String())

			// memory, pieces, err := op.ExecuteStackProgram(op.DwarfRegisters{}, field.Val.([]byte), 8, nil)

			// fmt.Printf("%v, %v, %v\n", memory, pieces, err)
		}

	}
	ranges, err := dwarfRawData.Ranges(entry)

	if err != nil {
		panic(err)
	}
    if(ranges!=nil){
		function.lowPC = ranges[0][0]
		function.highPC = ranges[0][1]
	}
	function.Parameters = make([]*Parameter, 0)

	return &function
}

func parseModule(entry *dwarf.Entry, dwarfRawData *dwarf.Data) *Module {
	module := Module{
		files:     make(map[int]string),
		functions: make([]*Function, 0),
		Variables: make([]*Variable, 0),
	}

	for _, field := range entry.Field {
		switch field.Attr {
		case dwarf.AttrName:
			module.name = field.Val.(string)
		case dwarf.AttrLanguage:
			// language can be inferred from the cu attributes. 22-golang, 12-clang
			// if field.Val.(int64) == 22 {
			// 	data.lang = golang
			// }
		case dwarf.AttrProducer:
			// can infer arch
		}
	}

	ranges, err := dwarfRawData.Ranges(entry)

	if err != nil {
		panic(err)
	}
	// might be more than 1 range entry in theory
	module.startAddress = ranges[0][0]
	module.endAddress = ranges[0][1]

	lineReader, err := dwarfRawData.LineReader(entry)
	if err != nil {
		panic(err)
	}

	moduleFileIndexMap := make(map[string]int)

	files := lineReader.Files()
	for fileIndex, file := range files {
		if file != nil {
			module.files[fileIndex] = file.Name
			moduleFileIndexMap[file.Name] = fileIndex
		}
	}

	dEntries := make([]Entry, 0)
	for {
		var le dwarf.LineEntry

		err := lineReader.Next(&le)

		if err == io.EOF {
			break
		}

		entry := Entry{
			Address:       le.Address,
			file:          moduleFileIndexMap[le.File.Name],
			line:          le.Line,
			col:           le.Column,
			prologueEnd:   le.PrologueEnd,
			epilogueBegin: le.EpilogueBegin,
			isStmt:        le.IsStmt,
		}

		// adjust for inserted line
		entry.line--

		dEntries = append(dEntries, entry)

	}

	module.entries = dEntries

	return &module
}
