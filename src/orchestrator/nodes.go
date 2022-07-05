package main

import (
	"github.com/ottmartens/cc-rev-db/logger"
)

var aliveProcessIds []int = make([]int, 0)

type Registrator struct{}

func (r Registrator) Register(pid *int, reply *int) error {
	logger.Verbose("adding process %v to process list", *pid)
	aliveProcessIds = append(aliveProcessIds, *pid)

	*reply = len(aliveProcessIds) - 1

	return nil
}
