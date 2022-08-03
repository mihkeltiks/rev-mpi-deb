package websocket

import (
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/orchestrator/checkpointmanager"
	"github.com/ottmartens/cc-rev-db/orchestrator/cli"
	nodeconnection "github.com/ottmartens/cc-rev-db/orchestrator/nodeConnection"
)

type MessageType string

const (
	CheckpointUpdate MessageType = "checkpointUpdate"
	RollbackSubmit   MessageType = "rollbackSubmit"
	RollbackConfirm  MessageType = "rollbackConfirm"
	RollbackResult   MessageType = "rollbackResult"
)

type CheckpointUpdateMessage struct {
	Type  MessageType
	Value checkpointmanager.CheckpointLog
}

type RollbackSubmitMessage struct {
	Type  MessageType
	Value string
}

type RollbackConfirmMessage struct {
	Type  MessageType
	Value checkpointmanager.RollbackMap
}

type RollbackCommitMessage struct {
	Type  MessageType
	Value bool
}

type RollbackResultMessage struct {
	Type  MessageType
	Value checkpointmanager.CheckpointLog
}

func SendCheckpointUpdateMessage(checkpointLog checkpointmanager.CheckpointLog) {
	SendMessage(CheckpointUpdateMessage{
		Type:  CheckpointUpdate,
		Value: checkpointLog,
	})
}

func sendRollbackConfirm(rollbackMap checkpointmanager.RollbackMap) {
	SendMessage(RollbackConfirmMessage{
		Type:  RollbackConfirm,
		Value: rollbackMap,
	})
}

func handleRollbackSubmit(checkpointId string) {
	rollbackMap := checkpointmanager.SubmitForRollback(checkpointId)
	sendRollbackConfirm(*rollbackMap)
}

func handleRollbackCommit(execute bool) {
	if execute {
		nodeconnection.ExecutePendingRollback()
		SendMessage(RollbackResultMessage{
			Type:  RollbackResult,
			Value: checkpointmanager.GetCheckpointLog(),
		})
	} else {
		logger.Verbose("Cancelling pending rollback")
		checkpointmanager.ResetPendingRollback()
	}

	cli.PrintPrompt()
}
