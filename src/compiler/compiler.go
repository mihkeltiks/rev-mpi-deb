package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
)

const (
	TEMP_FOLDER              = "bin/temp"
	DEST_FOLDER              = "bin/targets"
	WRAPPED_MPI_FILE_INCLUDE = `#include "debug_mpi_wrap.h"`
	WRAPPED_MPI_FORK_INCLUDE = `#include "debug_mpi_wrap_fork.h"`
	WRAPPED_MPI_PATH         = "src/compiler/mpi_wrap_include"
)

var WRAPPED_MPI_INCLUDE string = WRAPPED_MPI_FILE_INCLUDE

/*
Compile MPI programs for the debugger
Wraps the MPI library in the target to enable intercepting MPI calls
*/
func main() {
	err := executeWorkflow()

	if err != nil {
		os.Exit(1)
	}
}

func executeWorkflow() error {
	inputFilePath, err := parseArguments()
	if err != nil {
		printUsage()
		return err
	}

	// fork-based checkpointing temporarily disabled
	// determineCheckpointingMode()

	err = ensureValidTargetExists(inputFilePath)
	if err != nil {
		logger.Error("Specified target is not valid: %v", err)
		return err
	}

	wrappedSource, err := createWrappedCopy(inputFilePath)
	if err != nil {
		logger.Error("Failed to create a wrapped source copy: %v", err)
		return err
	}
	source, _ := os.Open(wrappedSource.Name())
	defer source.Close()
	scanner := bufio.NewScanner(source)
	i := 1
	for scanner.Scan() {
		logger.Verbose("%v", i)
		logger.Verbose("%v", scanner.Text())

		i++
	}
	//remove the temporary wrapped source file
	defer os.Remove(wrappedSource.Name())

	err = compile(wrappedSource.Name(), getDestPath(inputFilePath))
	if err != nil {
		logger.Error("Compilation failed: %v ", err)
		return err
	}

	return nil
}

func compile(sourcePath string, destPath string) error {
	cmd := exec.Command("mpicc", "-g3", "-gdwarf-4", "-O0", "-no-pie", "-I", WRAPPED_MPI_PATH, "-o", destPath, sourcePath)

	logger.Info("compiling target")
	// logger.Verbose("%v", cmd)

	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		return err
	}

	logger.Info("wrote compiled target to: %v", destPath)
	logger.Info("compilation finished")

	return nil
}

func createWrappedCopy(inputFilePath string) (*os.File, error) {
	filePath := fmt.Sprintf("%s/%s", TEMP_FOLDER, path.Base(inputFilePath))
	dest, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	source, _ := os.Open(inputFilePath)
	defer source.Close()
	scanner := bufio.NewScanner(source)
	dest.WriteString(terminate(WRAPPED_MPI_INCLUDE))
	dest.WriteString(terminate("#include <signal.h>"))
	dest.WriteString(terminate("int counter = 0;"))
	dest.WriteString(terminate("int target = 2000000;"))
	dest.WriteString(terminate("void call_counter(){\ncounter++;\n if (counter==target){\nraise(SIGTRAP);\n}\n};"))
	dest.WriteString(terminate("#define DECLARE_AND_INIT(type, var, value)\\"))
	dest.WriteString(terminate("call_counter();\\"))
	dest.WriteString(terminate("type var = value;"))
	infunction := 0
	inSingleLineComment := false
	inMultiLineComment := false
	for scanner.Scan() {
		line := scanner.Text()
		line, _, inMultiLineComment = removeCommentsFromLine(line, inSingleLineComment, inMultiLineComment)
		if inMultiLineComment {
			dest.WriteString(terminate(line))
			continue
		}

		line = prefixMPICalls(line)
		for _, char := range line {
			if char == '{' {
				infunction++
			}
			if char == '}' {
				infunction--
			}
		}

		str := strings.TrimSpace(line)
		if len(str) > 2 && str[0:3] == "int" && infunction > 0 {
			index := strings.Index(line, "=")
			if index == -1 {
				fmt.Println("No '=' found in the string.")
				continue
			}

			leftPart := strings.TrimSpace(line[:index])
			rightPart := strings.TrimSpace(line[index+1:])

			// Split the left part into type and variable name
			leftFields := strings.Fields(leftPart)
			if len(leftFields) < 2 {
				fmt.Println("Invalid declaration format.")
				continue
			}

			typeName := leftFields[0]
			varName := leftFields[1]
			value := rightPart

			if strings.HasSuffix(value, ";") {
				value = value[:len(value)-1]
			}

			line = "DECLARE_AND_INIT(" + typeName + "," + varName + "," + value + ");"
		} else if infunction > 0 && str != "" && str[0] != '#' && (len(str) > 2 && str[0:3] != "int") && (len(str) > 3 && str[0:4] != "void" && str[0:4] != "char") {
			line = "call_counter();" + line
		}
		// logger.Verbose("LINE %v", line)

		dest.WriteString(terminate(line))
	}

	return dest, nil
}

func prefixMPICalls(line string) string {
	mpiFuncRegexp := regexp.MustCompile(`(MPI_[^\s]*?\()`)

	return mpiFuncRegexp.ReplaceAllString(line, "_$1")
}

func getDestPath(inputFilePath string) string {
	return path.Join(DEST_FOLDER, fileNameWithoutExtension(inputFilePath))
}

func fileNameWithoutExtension(inputFilePath string) string {
	inputFileName := path.Base(inputFilePath)

	return strings.TrimSuffix(inputFileName, path.Ext(inputFileName))
}

func ensureValidTargetExists(inputFilePath string) error {
	fileInfo, err := os.Stat(inputFilePath)

	if err != nil {
		return fmt.Errorf("unable to find file: %v", inputFilePath)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%v is a directory", inputFilePath)
	}

	validExtensions := map[string]bool{".c": true, ".cpp": true}

	fileExtension := path.Ext(fileInfo.Name())

	if !validExtensions[fileExtension] {
		return fmt.Errorf("unsupported file extension: %v", fileExtension)
	}

	return nil
}

func parseArguments() (string, error) {
	if len(os.Args) < 2 {
		return "", errors.New("")
	}
	return os.Args[1], nil
}

func printUsage() {
	logger.Info("Usage: compiler <target file path>")
	// logger.Info("Usage: compiler <target file> [fork](live-checkpointing)")
}

func determineCheckpointingMode() {
	if len(os.Args) > 2 && os.Args[2] == "fork" {
		WRAPPED_MPI_INCLUDE = WRAPPED_MPI_FORK_INCLUDE
	}
}

func terminate(line string) string {
	return fmt.Sprintf("%s\n", line)
}

func removeCommentsFromLine(line string, inSingleLineComment bool, inMultiLineComment bool) (string, bool, bool) {
	var result strings.Builder
	// logger.Verbose("LINE %v", line)
	// logger.Verbose("MULTICOMM %v", inMultiLineComment)

	lineLen := len(line)

	for i := 0; i < lineLen; i++ {
		if inSingleLineComment {
			break // Ignore the rest of the line
		} else if inMultiLineComment {
			if line[i] == '*' && i+1 < lineLen && line[i+1] == '/' {
				inMultiLineComment = false
				i++ // Skip the closing '/'
			}
		} else {
			if line[i] == '/' && i+1 < lineLen && line[i+1] == '/' {
				inSingleLineComment = true
				break // Ignore the rest of the line
			} else if line[i] == '/' && i+1 < lineLen && line[i+1] == '*' {
				inMultiLineComment = true
				i++ // Skip the '*'
			} else {
				result.WriteByte(line[i])
			}
		}
	}

	return result.String(), inSingleLineComment, inMultiLineComment
}
