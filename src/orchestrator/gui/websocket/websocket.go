package websocket

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"logger"
	"utils"
)

var connection *websocket.Conn

const ADDRESS = "localhost:3496"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func SendMessage(value interface{}) {
	if connection == nil {
		logger.Warn("Cannot send ws message: client not connected")
		return
	}

	err := connection.WriteJSON(value)
	if err != nil {
		logger.Warn("Error sending ws message: %v", err)
	}
}

func handler() func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		utils.Must(err)
		if connection == nil {
			logger.Verbose("client connected to websocket")
		}

		connection = c
		defer c.Close()

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if err == websocket.ErrCloseSent {
					logger.Verbose("Client disconnected from websocket")
					connection = nil
				} else {
					logger.Warn("ws read error: %v", err)
				}
				break
			}

			rollbackSubmitMessage := &RollbackSubmitMessage{}
			err = json.Unmarshal(message, rollbackSubmitMessage)
			if err == nil {
				logger.Verbose("received rollback submit message")
				handleRollbackSubmit(rollbackSubmitMessage.Value)

				continue
			}

			rollbackCommitMessage := &RollbackCommitMessage{}
			err = json.Unmarshal(message, rollbackCommitMessage)
			if err == nil {
				logger.Verbose("received rollback commit message")
				handleRollbackCommit(rollbackCommitMessage.Value)

				continue
			}

			logger.Info("unknown ws message: %s", message)

		}
	}

}

func InitServer() {
	http.HandleFunc("/", handler())

	logger.Verbose("starting websocket server for gui")
	go http.ListenAndServe(ADDRESS, nil)
}

func WaitForClientConnection() {
	for {
		if connection != nil {
			return
		}
	}
}
