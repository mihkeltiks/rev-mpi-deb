const SERVER_URL = 'ws://127.0.0.1:3496';
let socket;

export const connect = (onMessage) => {
	socket = new WebSocket(SERVER_URL);


	socket.onerror = (err) => {
		console.log('socket error', err);
	};

	socket.onopen = () => {
		console.log('socket connected');
	};

	socket.onmessage = (message) => {
		try {
			const data = JSON.parse(message.data);

			const messageType = data.Type;
			const messageValue = data.Value;

			console.log('new message of type', messageType);
			onMessage({ type: messageType, value: messageValue });
		} catch (err) {
			console.warn(`Error receiving ws message: ${err}`);
		}
	};

	socket.onclose = () => {
		window.close();
	};
};

export const sendMessage = (messageType, payload) => {
	if (socket && socket.OPEN) {
		console.log('sending socket message', messageType);
		socket.send(
			JSON.stringify({
				Type: messageType,
				Value: payload,
			})
		);
	}
};
