package rpc

import (
	"errors"
	"fmt"
	"net/rpc"
	"net/url"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
)

type RPCClient struct {
	connection *rpc.Client
	address    *url.URL
}

func Connect(serverAddress *url.URL) *RPCClient {
	// logger.Debug("connecting to rpc server at %v", serverAddress)

	connection, err := rpc.DialHTTP("tcp", serverAddress.String())
	if err != nil {
		logger.Error("Failed to connect to rpc server at %v", serverAddress)
		panic(err)
	}
	// logger.Debug("connected")

	client := RPCClient{
		connection: connection,
		address:    serverAddress,
	}

	return &client
}

func (r *RPCClient) Disconnect() {
	// r.mu.Lock()
	// defer r.mu.Unlock()

	if r.connection != nil {
		err := r.connection.Close()
		if err == nil {
			r.connection = nil
		}
	} else {
		logger.Debug("Already disconnected from rpc server at %v", r.address)
	}
}

func (r *RPCClient) Call(methodName string, args any, reply any) error {
	if r.connection == nil {
		return errors.New("Not connected to rpc server")
	}

	return r.connection.Call(methodName, args, reply)
}

func (r *RPCClient) Heartbeat() {
	err := r.Call("Health.Heartbeat", new(int), new(int))

	if err != nil {
		panic(fmt.Sprintf("Heartbeat error: %v", err))
	}

	logger.Debug("Heartbeat ok (server %v)", r.address)
}
