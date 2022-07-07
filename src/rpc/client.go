package rpc

import (
	"errors"
	"fmt"
	"net/rpc"
	"net/url"
	"os"

	"github.com/ottmartens/cc-rev-db/logger"
)

type RPCClient struct {
	connection *rpc.Client
	address    *url.URL
}

var Client *RPCClient = &RPCClient{}

func (r *RPCClient) Connect(serverAddress *url.URL) *RPCClient {
	logger.Debug("connecting to rpc server at %v", serverAddress)

	client, err := rpc.DialHTTP("tcp", serverAddress.String())
	if err != nil {
		panic(err)
	}
	logger.Debug("connected")

	r.connection = client
	r.address = serverAddress

	return r
}

func (r *RPCClient) Call(methodName string, args any, reply any) error {
	if r.connection == nil {
		return errors.New("Not connected to rpc server")
	}

	return r.connection.Call(methodName, args, reply)
}

func (r *RPCClient) ReportAsHealthy() (nodeId int) {

	err := r.Call("NodeReporter.Register", os.Getpid(), &nodeId)
	if err != nil {
		panic(err)
	}

	return nodeId
}

func (r *RPCClient) SendLog(args *logger.RemoteLogArgs) error {
	return r.Call("LoggerServer.Log", args, new(int))
}

func (r *RPCClient) Heartbeat() {
	err := r.Call("Health.Heartbeat", new(int), new(int))

	if err != nil {
		panic(fmt.Sprintf("Heartbeat error, %v", err))
	}

	logger.Debug("Heartbeat ok (server %v)", r.address)
}
