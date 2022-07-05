package logger

import (
	"fmt"
	"net/rpc"
	"os"
)

var remoteClient *rpc.Client

// client

func SetRemoteClient(client *rpc.Client) {
	remoteClient = client
}

func logRemotely(level LoggingLevel, message string) {
	var reply int

	args := RemoteLogArgs{
		os.Getpid(),
		level,
		message,
	}
	err := remoteClient.Call("LoggerServer.LogRow", &args, &reply)

	if err != nil {
		panic(err)
	}
}

// server

type LoggerServer int

type RemoteLogArgs struct {
	Pid     int
	Level   LoggingLevel
	Message string
}

func (r *LoggerServer) LogRow(args RemoteLogArgs, reply *int) error {
	message := fmt.Sprintf("%d - %v", args.Pid, args.Message)

	logRow(args.Level, message)

	return nil
}
