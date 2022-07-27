package rpc

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"

	"github.com/ottmartens/cc-rev-db/logger"
)

type Registrator func(any) error

func InitializeServer(port int, registerComponents func(Registrator)) {
	// register components
	registerComponents(rpc.Register)

	// register heartbeat
	rpc.Register(new(Health))

	//serve
	rpc.HandleHTTP()

	serverAddress := fmt.Sprintf("localhost:%d", port)

	listener, err := net.Listen("tcp", serverAddress)
	if err != nil {
		logger.Error("logger server listen error: %v", err)
		panic(err)
	} else {
		logger.Verbose("rpc server listening on address: %v", serverAddress)
	}

	http.Serve(listener, nil)
}

type Health int

func (h *Health) Heartbeat(args *int, reply *int) error {
	return nil
}
