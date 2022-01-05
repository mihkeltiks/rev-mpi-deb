package main

// #include "mpi.h"
import "C"

import (
	"fmt"
)

func main() {
	C.MPI_Init(nil, nil)

	var size int
	C.MPI_Comm_size(C.MPI_COMM_WORLD, &size)

	fmt.Printf("rank: %d , size: %d\n", rank, size)
}
