package checkpointmanager

import (
	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
)

// holds the checkpoints to be restored, obtained from the submission of a rollback command
type RollbackMap map[nodeId]checkpointRecord

// returns which (additional) checkpoints need to be rolled back
// if the supplied checkpoint is to be restored
// in order to maintain causal consistency
func SubmitForRollback(cmd *command.Command) *RollbackMap {

	checkpointId := cmd.Argument.(string)
	originalCheckpoint := findCheckpointById(checkpointId)

	if originalCheckpoint == nil {
		logger.Warn("Cannot find checkpoint with id %v", checkpointId)
		return nil
	}

	if !originalCheckpoint.canBeRestored {
		logger.Warn("Checkpoint of type %v cannot be restored", originalCheckpoint.opNname)
		return nil
	}

	logger.Debug("Finding related checkpoints for rollback, original checkpoint: %v", originalCheckpoint)

	rollbackPointsPerNode := make(RollbackMap)

	rollbackPointsPerNode[originalCheckpoint.nodeId] = *originalCheckpoint

	// TEMP
	for nodeId, nodeCheckpoints := range checkpointLog {
		for _, cp := range nodeCheckpoints {
			if cp.canBeRestored {
				rollbackPointsPerNode[nodeId] = cp
				break
			}
		}

	}

	return &rollbackPointsPerNode
}
