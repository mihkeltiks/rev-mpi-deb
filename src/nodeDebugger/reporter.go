package main

import (
	"os"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/rpc"
)

func reportCommandResult(cmd *command.Command) {
	err := rpc.Client.Call("NodeReporter.ReportCommandResult", cmd, new(int))
	if err != nil {
		logger.Error("Failed to report command result: %v", err)
		panic(err)
	}
}

func reportAsHealthy() (nodeId int) {
	err := rpc.Client.Call("NodeReporter.Register", os.Getpid(), &nodeId)
	if err != nil {
		panic(err)
	}

	return nodeId
}
