package main

import (
	"os"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

type info struct {
	line   int
	nodeid int
}

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

func reportBreakpoint(ctx *processContext, cmd *command.Command) {
	logger.Verbose("%v", cmd.Code)
	err := ctx.nodeData.rpcClient.Call("NodeReporter.Breakpoint", cmd, new(int))
	if err != nil {
		logger.Error("Failed to report breakpoint information: %v", err)
		panic(err)
	}
}

func reportProgressCommand(ctx *processContext, cmd *command.Command) {
	err := ctx.nodeData.rpcClient.Call("NodeReporter.Progress", cmd, new(int))
	if err != nil {
		logger.Error("Failed to report progresss command execution: %v", err)
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
