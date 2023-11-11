package proc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
)

func GetFileCheckpointDataAddresses(pid int, sourceFile string) []MemRegion {

	idents := []string{
		// "[heap]",
		"[stack]",
		sourceFile,
	}

	return GetDataAddressesByIdents(pid, idents)
}

func GetForkCheckpointDataAddresses(pid int, sourceFile string) []MemRegion {
	idents := []string{
		"[heap]",
		sourceFile,
	}

	return GetDataAddressesByIdents(pid, idents)
}

func GetStackDataAddresses(pid int) []MemRegion {
	idents := []string{
		"[stack]",
	}

	return GetDataAddressesByIdents(pid, idents)
}

func GetDataAddressesByIdents(pid int, identifiers []string) []MemRegion {

	identsMap := make(map[string]bool)

	for _, ident := range identifiers {
		identsMap[ident] = true
	}

	logger.Debug("reading memory regions with following identifiers: %v", identifiers)

	regions := make([]MemRegion, 0)

	mmaps := readMapsFile(pid)

	for _, mmap := range mmaps {

		ident := mmap[len(mmap)-1]

		if identsMap[ident] {
			bounds := strings.Split(mmap[0], "-")

			start, _ := strconv.ParseUint(bounds[0], 16, 64)
			end, _ := strconv.ParseUint(bounds[1], 16, 64)

			regions = append(regions, MemRegion{
				start,
				end,
				ident,
				nil,
			})

		}
	}

	return regions
}

func readMapsFile(pid int) [][]string {
	regions := make([][]string, 0)

	mapFile := fmt.Sprintf("/proc/%d/maps", pid)

	source, err := os.Open(mapFile)
	if err != nil {
		panic(err)
	}

	defer source.Close()

	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		line := strings.Fields(scanner.Text())

		regions = append(regions, line)
	}

	return regions
}

func LogMapsFile(pid int) {
	regions := readMapsFile(pid)

	for _, region := range regions {
		logger.Debug("%v", region)
	}
}
