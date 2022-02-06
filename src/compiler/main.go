package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

const (
	TEMP_FOLDER         = "bin/temp"
	DEST_FOLDER         = "bin/targets"
	WRAPPED_MPI_INCLUDE = `#include "mpi_wrap.h"`
	WRAPPED_MPI_PATH    = "src/compiler/mpi_wrap_include"
)

/*
	Wrapper to compile MPI programs
	Wraps the MPI library for the target to enable intercepting MPI calls
*/
func main() {
	validateArgs()

	// path to input file
	inputFile := os.Args[1]

	ensureTargetExists(inputFile)

	wrappedSource := createWrappedCopy(inputFile)

	//remove the wrapped source file
	defer os.Remove(wrappedSource.Name())

	compile(wrappedSource.Name(), getDestPath(inputFile))
}

func compile(sourcePath string, destPath string) {
	cmd := exec.Command("mpicc", "-g", "-no-pie", "-I", WRAPPED_MPI_PATH, "-o", destPath, sourcePath)

	fmt.Println("compiling target with:")
	fmt.Println(cmd)

	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		fmt.Println("compilation failed: ", err)
		os.Exit(1)
	}

	fmt.Println("compilation finished")
}

func createWrappedCopy(inputFile string) *os.File {

	fileName := fmt.Sprintf("%s-*.c", inputFile[strings.LastIndex(inputFile, "/")+1:])

	dest, err := os.CreateTemp(TEMP_FOLDER, fileName)

	if err != nil {
		panic(err)
	}

	source, _ := os.Open(inputFile)
	defer source.Close()

	scanner := bufio.NewScanner(source)

	dest.WriteString(terminate(WRAPPED_MPI_INCLUDE))

	for scanner.Scan() {
		line := scanner.Text()

		line = prefixMPICalls(line)

		dest.WriteString(terminate(line))
	}

	return dest
}

func prefixMPICalls(line string) string {
	mpiFuncRegexp := regexp.MustCompile(`(MPI_[^\s]*?\()`)

	return mpiFuncRegexp.ReplaceAllString(line, "_$1")
}

func getDestPath(inputFilePath string) string {

	inputFileName := path.Base(inputFilePath)
	withoutExtension := strings.TrimSuffix(inputFileName, path.Ext(inputFileName))

	return path.Join(DEST_FOLDER, withoutExtension)
}

func ensureTargetExists(inputFile string) {
	_, err := os.Stat(inputFile)

	if err != nil {
		fmt.Printf("unable to find file: %v\n", inputFile)
		os.Exit(2)
	}
}

func validateArgs() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: compiler <target file>")
		os.Exit(2)
	}
}

func terminate(line string) string {
	return fmt.Sprintf("%s\n", line)
}
