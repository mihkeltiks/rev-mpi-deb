package main

import "unsafe"

func wordByteSize() int {
	var i int
	return int(unsafe.Sizeof(i))
}
