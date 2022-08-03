package checkpointmanager

import (
	"github.com/ottmartens/cc-rev-db/logger"
)

// holds the checkpoints to be restored, obtained from the submission of a rollback command
type RollbackMap map[NodeId]checkpointRecord

var pendingRollback *RollbackMap

// returns which (additional) checkpoints need to be rolled back
// if the supplied checkpoint is to be restored
// in order to maintain causal consistency
func SubmitForRollback(checkpointId string) *RollbackMap {
	originalCheckpoint := findCheckpointById(checkpointId)

	if originalCheckpoint == nil {
		logger.Warn("Cannot find checkpoint with id %v", checkpointId)
		return nil
	}

	if !originalCheckpoint.CanBeRestored {
		logger.Warn("Checkpoint of type %v cannot be restored", originalCheckpoint.OpName)
		return nil
	}

	logger.Debug("Finding related checkpoints for rollback, original checkpoint: %v", originalCheckpoint)

	rollbackPointsPerNode := RollbackMap{
		originalCheckpoint.nodeId: *originalCheckpoint,
	}

	for {
		updated := false

		for nodeId, nodeRollbackCheckpoint := range rollbackPointsPerNode {

			for i := checkpointIndex(nodeId, nodeRollbackCheckpoint.Id); i < len(checkpointLog[nodeId]); i++ {

				checkpoint := checkpointLog[nodeId][i]

				matchingEvent := checkpoint.matchingEvent

				if matchingEvent != nil {

					existingRollbackEvent, hasExistingRollbackEvent := rollbackPointsPerNode[matchingEvent.nodeId]

					if !hasExistingRollbackEvent || isBefore(matchingEvent.Id, existingRollbackEvent.Id, matchingEvent.nodeId) {
						rollbackPointsPerNode[matchingEvent.nodeId] = *matchingEvent
						updated = true
					}
				}
			}
		}

		if !updated {
			break
		}
	}

	// // TEMP
	// for nodeId, nodeCheckpoints := range checkpointLog {
	// 	for _, cp := range nodeCheckpoints {
	// 		if cp.CanBeRestored {
	// 			rollbackPointsPerNode[nodeId] = *cp
	// 			break
	// 		}
	// 	}

	// }

	pendingRollback = &rollbackPointsPerNode

	return pendingRollback
}

// Returns whether checkpoint 1 happened before checkpoint 2 on the specified node
func isBefore(checkpointId1 string, checkpointId2 string, nodeId NodeId) bool {
	var idx1, idx2 int
	for idx, checkpoint := range checkpointLog[nodeId] {
		if checkpoint.Id == checkpointId1 {
			idx1 = idx
		}
		if checkpoint.Id == checkpointId2 {
			idx2 = idx
		}
	}

	return idx1 < idx2
}

func checkpointIndex(nodeId NodeId, checkpointId string) int {
	for idx, checkpoint := range checkpointLog[nodeId] {
		if checkpoint.Id == checkpointId {
			return idx
		}
	}

	logger.Warn("Cannot find checkpoint on node")
	return 0
}

func GetPendingRollback() *RollbackMap {
	return pendingRollback
}

func ResetPendingRollback() {
	pendingRollback = nil
}
