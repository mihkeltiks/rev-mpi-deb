package logger

import (
	"net/rpc"
)

var remoteClient *rpc.Client
var nodeId int

// client

func SetRemoteClient(client *rpc.Client) {
	remoteClient = client
}

func SetNodeId(id int) {
	nodeId = id
}

func logRemotely(level LoggingLevel, message string) {
	var reply int

	args := RemoteLogArgs{
		nodeId,
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
	logRow(args.Level, args.Message, &args.Pid)

	return nil
}
