package proc

import (
	"fmt"
	"os"
)

func ReadFromMemFileByRegions(pid int, regions []MemRegion) [][]byte {

	contents := make([][]byte, 0)

	for _, region := range regions {
		contents = append(contents, region.ContentsFromFile(pid))

	}

	return contents
}

func ReadFromMemFile(pid int, address uint64, length int) []byte {

	file, err := os.Open(memFileName(pid))

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

func WriteRegionsContentsToMemFile(pid int, regions []MemRegion) error {
	file, err := os.OpenFile(memFileName(pid), os.O_WRONLY, os.ModeIrregular)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	for _, region := range regions {

		_, err := file.WriteAt(region.Contents, int64(region.Start))

		if err != nil {
			return fmt.Errorf("error writing region %v - %v", region, err)
		}

	}

	return nil
}

func memFileName(pid int) string {
	return fmt.Sprintf("/proc/%d/mem", pid)
}
