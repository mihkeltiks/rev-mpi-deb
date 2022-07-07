package logger

var sendRemoteLog func(args *RemoteLogArgs) error
var nodeId int

// client

func SetSendRemoteLog(sendLog func(args *RemoteLogArgs) error, _nodeId int) {
	sendRemoteLog = sendLog
	nodeId = _nodeId
}

func logRemotely(level LoggingLevel, message string) {
	err := sendRemoteLog(&RemoteLogArgs{
		nodeId,
		level,
		message,
	})

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
