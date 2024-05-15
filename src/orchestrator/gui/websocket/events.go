package websocket

import (
	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/checkpointmanager"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/cli"
	nodeconnection "github.com/mihkeltiks/rev-mpi-deb/orchestrator/nodeConnection"
)

type MessageType string

const (
	CheckpointUpdate MessageType = "checkpointUpdate"
	RollbackSubmit   MessageType = "rollbackSubmit"
	RollbackConfirm  MessageType = "rollbackConfirm"
	RollbackResult   MessageType = "rollbackResult"
	CriuCheckpoint   MessageType = "criuCheckpoint"
	CriuRestore      MessageType = "criuRestore"
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

func HandleCriuRestore(index int) {
	// logger.Verbose("Handling criu rollback")
	SendMessage(RollbackResultMessage{
		Type:  CriuRestore,
		Value: checkpointmanager.GetCheckpointLogIndex(index),
	})
}

func HandleCriuCheckpoint() {
	// logger.Verbose("Handling criu Checkpoint")
	SendMessage(RollbackResultMessage{
		Type:  CriuCheckpoint,
		Value: checkpointmanager.GetCheckpointLog(),
	})
}
