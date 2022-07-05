package rpcclient

import (
	"fmt"
	"net/rpc"
	"net/url"

	"github.com/ottmartens/cc-rev-db/logger"
)

var remoteClient *rpc.Client

func Connect(serverUrl *url.URL) {

	remoteAddress := fmt.Sprintf("%s", serverUrl.String())

	client, err := rpc.DialHTTP("tcp", remoteAddress)
	if err != nil {
		panic(err)
	}

	remoteClient = client

	logger.SetRemoteClient(client)

	reportAsHealthy()
	logger.Info("Process registered")
}
