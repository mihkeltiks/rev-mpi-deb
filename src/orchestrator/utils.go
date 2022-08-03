package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("process %d - %v", os.Getpid(), err))
	}
}

// Returns the directory of the currently running executable
func getExecutableDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}
