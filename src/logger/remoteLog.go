package logger

// client

var remoteLoggerClient RemoteLoggerClient
var nodeId int

type RemoteLoggerClient interface {
	Call(methodName string, args any, reply any) error
}

func SetRemoteClient(client RemoteLoggerClient, _nodeId int) {
	remoteLoggerClient = client
	nodeId = _nodeId
}

func logRemotely(level LoggingLevel, message string) {
	err := remoteLoggerClient.Call("LoggerServer.Log", &RemoteLogArgs{
		nodeId,
		level,
		message,
	}, new(int))

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

func (r *LoggerServer) Log(args RemoteLogArgs, reply *int) error {
	logRow(args.Level, args.Message, &args.Pid)

	return nil
}
