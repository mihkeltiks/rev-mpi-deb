package main

import "github.com/ottmartens/cc-rev-db/logger"

var aliveProcessIds []int = make([]int, 0)

func AddNewProcess(pid int) {
	logger.Verbose("adding process %v to process list", pid)

	aliveProcessIds = append(aliveProcessIds, pid)
}

type Registrator struct{}

func (r Registrator) Register(pid *int, reply *int) error {
	AddNewProcess(*pid)

	return nil
}
