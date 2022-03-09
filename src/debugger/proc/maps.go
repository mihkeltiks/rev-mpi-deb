package proc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ottmartens/cc-rev-db/logger"
)

func GetCheckpointDataAddresses(pid int, sourceFile string) []MemRegion {

	checkpointRegionIdents := map[string]bool{
		"[heap]":   true,
		sourceFile: true,
	}

	return getDataAddressesByIdents(pid, checkpointRegionIdents)
}

func GetStackDataAddresses(pid int) []MemRegion {
	idents := map[string]bool{
		"[stack]": true,
	}

	return getDataAddressesByIdents(pid, idents)
}

func getDataAddressesByIdents(pid int, identifiers map[string]bool) []MemRegion {
	regions := make([]MemRegion, 0)

	mmaps := readMapsFile(pid)

	for _, mmap := range mmaps {
		ident := mmap[len(mmap)-1]

		if identifiers[ident] {
			bounds := strings.Split(mmap[0], "-")

			start, _ := strconv.ParseUint(bounds[0], 16, 64)
			end, _ := strconv.ParseUint(bounds[1], 16, 64)

			regions = append(regions, MemRegion{
				start,
				end,
				ident,
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
		logger.Info("%v", region)
	}
}
