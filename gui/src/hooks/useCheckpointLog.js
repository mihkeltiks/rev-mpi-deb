import { useState } from 'react';
import { MESSAGE_TYPES } from '../constants';

export default function useCheckpointLog() {
	const [checkpointLog, setCheckpointLog] = useState({});
	const [criuRollbackLogs, setCriuRollbackLogs] = useState([]);
	const [currentCheckpointLog, setCurrentCheckpointLog] = useState({});
	const [selectedIndex, setSelectedIndex] = useState(0); 

	const onWSMessage = ({ type, value }) => {
		switch (type) {
			case MESSAGE_TYPES.CHECKPOINT_UPDATE:
			case MESSAGE_TYPES.ROLLBACK_RESULT:
				setSelectedIndex(0);
				// setCheckpointLog(currentCheckpointLog);
				console.log(value);
				setCheckpointLog(value);
				setCurrentCheckpointLog(value);
				break;
			case MESSAGE_TYPES.CRIU_RESTORE:
				setSelectedIndex(0);	
				setCheckpointLog([]);
				setCurrentCheckpointLog([]);
				break;
			case MESSAGE_TYPES.CRIU_CHECKPOINT:
				setSelectedIndex(0);
				setCriuRollbackLogs(prevLogs => [...prevLogs, value]);
				setCurrentCheckpointLog(value);
				setCheckpointLog(value);
				break;
		}
	};

	return {
		onWSMessage,
		checkpointLog,
		setCheckpointLog,
		criuRollbackLogs,
		currentCheckpointLog,
		selectedIndex,
		setSelectedIndex
	};
}
