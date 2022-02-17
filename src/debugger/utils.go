package main

import "unsafe"

const MAIN_FN = "main"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// returns the pointer size of current arch
func ptrSize() int {
	return int(unsafe.Sizeof(uintptr(0)))
}
