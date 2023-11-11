import { useEffect, useState } from 'react';
import { MESSAGE_TYPES } from '../constants';
import { sendMessage } from '../websocket';

export default function useRollback() {
	const [
		pendingRollbackOriginalCheckpoint,
		setPendingRollbackOriginalCheckpoint,
	] = useState(null);

	const [pendingRollbackNodes, setPendingRollbackNodes] = useState(null);

	const onRollbackSubmit = (checkpoint) => {
		setPendingRollbackOriginalCheckpoint(checkpoint.Id);
		sendMessage(MESSAGE_TYPES.ROLLBACK_SUBMIT, checkpoint.Id);
	};

	const onRollbackCommit = () => {
		sendMessage(MESSAGE_TYPES.ROLLBACK_COMMIT, true);
	};

	const onRollbackCancel = () => {
		sendMessage(MESSAGE_TYPES.ROLLBACK_COMMIT, false);
		setPendingRollbackOriginalCheckpoint(null);
		setPendingRollbackNodes(null);
	};

	const onWSMessage = ({ type, value }) => {
		switch (type) {
			case MESSAGE_TYPES.ROLLBACK_CONFIRM:
				if (!value) {
					setPendingRollbackOriginalCheckpoint(null);
					setPendingRollbackNodes(null);
				} else {
					setPendingRollbackNodes(Object.values(value).map(({ Id }) => Id));
				}
                break
			case MESSAGE_TYPES.ROLLBACK_RESULT:
				setPendingRollbackNodes(null);
				setPendingRollbackOriginalCheckpoint(null);
                break
		}
	};

	return {
		onWSMessage,

		pendingRollbackOriginalCheckpoint,
		pendingRollbackNodes,

		onRollbackSubmit,

		onRollbackCommit,
		onRollbackCancel,
	};
}
