package rpcclient

import (
	"fmt"
	"net/rpc"
	"net/url"
	"os"

	"github.com/ottmartens/cc-rev-db/logger"
)

var remoteClient *rpc.Client

var nodeId int = -1

func Connect(serverUrl *url.URL) {

	remoteAddress := fmt.Sprintf("%s", serverUrl.String())

	client, err := rpc.DialHTTP("tcp", remoteAddress)
	if err != nil {
		panic(err)
	}

	remoteClient = client

	logger.SetRemoteClient(client)
	nodeId = reportAsHealthy()

	logger.SetNodeId(nodeId)
	logger.Info("Process (id: %d) registered", os.Getpid())
}
