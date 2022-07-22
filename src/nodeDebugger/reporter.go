package main

import (
	"os"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/rpc"
)

func reportAsHealthy(ctx *processContext) (nodeId int) {
	err := ctx.nodeData.rpcClient.Call("NodeReporter.Register", os.Getpid(), &nodeId)
	if err != nil {
		logger.Error("Failed to report self as healthy: %v", err)
		panic(err)
	}

	return nodeId
}

func reportCommandResult(ctx *processContext, cmd *command.Command) {
	err := ctx.nodeData.rpcClient.Call("NodeReporter.CommandResult", cmd, new(int))
	if err != nil {
		logger.Error("Failed to report command result: %v", err)
		panic(err)
	}
}

func reportMPICall(ctx *processContext, record *rpc.MPICallRecord) {
	err := ctx.nodeData.rpcClient.Call("NodeReporter.MPICall", record, new(int))
	if err != nil {
		logger.Error("Failed to report MPI call: %v", err)
		panic(err)
	}
}
