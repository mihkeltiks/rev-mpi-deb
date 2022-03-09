package proc

import (
	"fmt"
	"os"
)

func ReadFromMemFileByRegions(pid int, regions []MemRegion) [][]byte {

	contents := make([][]byte, 0)

	for _, region := range regions {
		contents = append(contents, region.Contents(pid))
	}

	return contents
}

func ReadFromMemFile(pid int, address uint64, length int) []byte {
	memFile := fmt.Sprintf("/proc/%d/mem", pid)

	file, err := os.Open(memFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	data := make([]byte, length)

	_, err = file.ReadAt(data, int64(address))

	if err != nil {
		panic(err)
	}

	// fmt.Printf("read from proc/mem: %v\n", data)

	return data
}
