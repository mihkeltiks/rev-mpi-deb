package rpcclient

import (
	"os"
)

func reportAsHealthy() {
	err := remoteClient.Call("Registrator.Register", os.Getpid(), new(int))

	if err != nil {
		panic(err)
	}
}
