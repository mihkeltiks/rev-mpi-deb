package rpcclient

import (
	"os"
)

func reportAsHealthy() int {
	var nodeId int
	err := remoteClient.Call("Registrator.Register", os.Getpid(), &nodeId)

	if err != nil {
		panic(err)
	}

	return nodeId
}
