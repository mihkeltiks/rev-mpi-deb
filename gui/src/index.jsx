import React from 'react';
import { createRoot } from 'react-dom/client';

import MessageGraph from './components/MessageGraph';
import useCheckpointLog from './hooks/useCheckpointLog';
import useRollback from './hooks/useRollback';

import * as websocket from './websocket';

const App = () => {
	const { onWSMessage, checkpointLog } = useCheckpointLog();

	const {
		onWSMessage: onRollbackMessage,

		pendingRollbackOriginalCheckpoint,
		pendingRollbackNodes,

		onRollbackSubmit,

		onRollbackCommit,
		onRollbackCancel,
	} = useRollback();

	websocket.connect((message) => {
		onWSMessage(message);
		onRollbackMessage(message);
	});

	return (
		<>
			{pendingRollbackNodes ? (
				<>
					<span>Orange checkpoints will be restored</span>
					<button onClick={onRollbackCommit}>Confirm</button>
					<button onClick={onRollbackCancel}>Cancel</button>
				</>
			) : (
				!!Object.values(checkpointLog).length && (
					<span>Click a green node to roll back to this checkpoint</span>
				)
			)}
			<MessageGraph
				pendingRollbackOriginalCheckpoint={pendingRollbackOriginalCheckpoint}
				onRollbackSubmit={onRollbackSubmit}
				checkpointLog={checkpointLog}
				pendingRollbackNodes={pendingRollbackNodes}
			/>
		</>
	);
};

window.resizeTo(400, 400);

createRoot(document.getElementById('root')).render(<App />);
