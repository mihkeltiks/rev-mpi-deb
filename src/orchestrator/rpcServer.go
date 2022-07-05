package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/ottmartens/cc-rev-db/logger"
)

func initRPCServer(port int) {
	// register components
	rpc.Register(new(logger.LoggerServer))
	rpc.Register(new(Registrator))

	//serve
	rpc.HandleHTTP()

	serverAddress := fmt.Sprintf("localhost:%d", port)

	listener, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Fatal("logger server listen error:", err)
	} else {
		logger.Info("rpc server listening on address: %v", serverAddress)
	}

	go http.Serve(listener, nil)
}
