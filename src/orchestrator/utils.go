package main

import (
	"fmt"
	"os"
)

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("process %d - %v", os.Getpid(), err))
	}
}
