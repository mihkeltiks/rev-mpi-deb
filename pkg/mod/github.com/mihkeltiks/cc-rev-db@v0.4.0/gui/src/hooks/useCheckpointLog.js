import { useState } from 'react';

import { MESSAGE_TYPES } from '../constants';

export default function useCheckpointLog() {
	const [checkpointLog, setCheckpointLog] = useState({});

	const onWSMessage = ({ type, value }) => {
		switch (type) {
			case MESSAGE_TYPES.CHECKPOINT_UPDATE:
            case MESSAGE_TYPES.ROLLBACK_RESULT:
				setCheckpointLog(value);
		}
	};

	return {
		onWSMessage,
		checkpointLog,
	};
}
