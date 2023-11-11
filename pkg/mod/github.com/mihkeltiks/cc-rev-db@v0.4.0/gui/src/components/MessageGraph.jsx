import React, { useEffect, useState } from 'react';
import { Stage, Layer, Circle, Text, Line } from 'react-konva';

const NODE_COLUMN_WIDTH = 120;
const NODE_SIZE = 30;
const NODE_ROW_GAP = 55;

const MessageNode = ({
	checkpoint,
	nodeIndex,
	checkpointIndex,
	onRollbackSubmit,
	isOriginalRollbackCheckpint,
	pendingRollback,
	pendingRollbackNodes,
}) => {
	const { x, y } = getNodeCoordinates(nodeIndex, checkpointIndex);

	const { OpName, CanBeRestored, CurrentLocation } = checkpoint;

	let color = 'blue';

	if (pendingRollback) {
		if (pendingRollbackNodes?.includes(checkpoint.Id)) {
			color = 'orange';
		} else if (isOriginalRollbackCheckpint) {
			color = 'red';
		}
	} else if (CurrentLocation) {
		color = 'white';
	} else if (CanBeRestored) {
		color = 'green';
	}

	return (
		<>
			<Circle
				x={nodeIndex * NODE_COLUMN_WIDTH + 60}
				y={checkpointIndex * NODE_ROW_GAP + 55}
				width={NODE_SIZE}
				height={NODE_SIZE}
				fill={color}
				stroke='black'
				perfectDrawEnabled
				onClick={() =>
					!pendingRollback &&
					CanBeRestored &&
					!CurrentLocation &&
					onRollbackSubmit(checkpoint)
				}
			/>
			<Text
				x={x}
				y={y}
				width={120}
				text={OpName}
				fontSize={13}
				fontFamily='Ubuntu'
				fill='black'
				align='center'
				perfectDrawEnabled
			/>
		</>
	);
};

const MessageEdge = ({ checkpointIndex, nodeIndex, matchingNode }) => {
	const { x, y } = getNodeCoordinates(nodeIndex, checkpointIndex);
	const matchingNodeCoords = getNodeCoordinates(
		matchingNode.nodeIndex,
		matchingNode.checkpointIndex
	);

	return (
		<Line
			points={[
				x + 60,
				y + 30,
				matchingNodeCoords.x + 60,
				matchingNodeCoords.y + 30,
			]}
			fill='black'
			stroke='black'
		/>
	);
};

const MessageGraph = ({
	checkpointLog,
	onRollbackSubmit,
	pendingRollbackOriginalCheckpoint,
	pendingRollbackNodes,
}) => {
	if (!Object.entries(checkpointLog).length) {
		return <div>No checkpoints recorded</div>;
	}

	const [rankOrder, setRankOrder] = useState(null);

	const [orderedNodeData, setOrderedNodeData] = useState([]);

	useEffect(() => {
		if (!rankOrder) {
			let newRankOrder = [];

			Object.entries(checkpointLog).forEach(([nodeId, nodeCheckpoints]) => {
				for (const checkpoint of nodeCheckpoints) {
					if (checkpoint.NodeRank !== null) {
						newRankOrder.push({
							rank: checkpoint.NodeRank,
							nodeId,
						});

						break;
					}
				}
			});

			if (
				newRankOrder.length > 0 &&
				newRankOrder.length == Object.keys(checkpointLog).length
			) {
				setRankOrder(
					newRankOrder.sort((a, b) => Number(a.rank) - Number(b.rank))
				);
			}
		}
	}, [checkpointLog]);

	useEffect(() => {
		checkpointLog = computeVectorClocks(checkpointLog);

		if (rankOrder) {
			setOrderedNodeData(
				rankOrder.reduce((acc, { nodeId }) => {
					acc.push(checkpointLog[nodeId]);
					return acc;
				}, [])
			);
		} else setOrderedNodeData(Object.values(checkpointLog));
	}, [rankOrder, checkpointLog]);

	const computeVectorClocks = (checkpointLog) => {
		const nodeIds = Object.keys(checkpointLog);

		for (const [nodeIdx, nodeCheckpoints] of Object.entries(checkpointLog)) {
			for (const [checkpointIdx, checkpoint] of nodeCheckpoints.entries()) {
				checkpoint.vectorClocks = {};

				for (const nodeId of nodeIds) {
					checkpoint.vectorClocks[nodeId] =
						nodeId == nodeIdx ? checkpointIdx + 1 : 0;
				}
			}
		}

		let changesMade = true;

		while (changesMade) {
			changesMade = false;

			for (const [nodeIdx, nodeCheckpoints] of Object.entries(checkpointLog)) {
				for (const [checkpointIdx, checkpoint] of nodeCheckpoints.entries()) {
					if (checkpointIdx > 0) {
						const previousCheckpoint = nodeCheckpoints[checkpointIdx - 1];

						for (const [nodeId, nodeVecClock] of Object.entries(
							previousCheckpoint.vectorClocks
						)) {
							if (checkpoint.vectorClocks[nodeId] < nodeVecClock) {
								checkpoint.vectorClocks[nodeId] = nodeVecClock;
							}
						}

						let positionIndex = Math.max(
							...Object.values(checkpoint.vectorClocks)
						);

						const previousPosIndex = Math.max(
							...Object.values(nodeCheckpoints[checkpointIdx - 1].vectorClocks)
						);

						if (positionIndex <= previousPosIndex) {
							for (const nodeClock of Object.keys(checkpoint.vectorClocks)) {
								checkpoint.vectorClocks[nodeClock] =
									checkpoint.vectorClocks[nodeClock] + 1;
							}

							changesMade = true;
						}
					}

					const { MatchingEventId, IsSend } = checkpoint;

					if (MatchingEventId && !IsSend) {
						const {
							checkpoint: matchingCheckpoint,
							nodeIndex: matchingNodeIdx,
						} = getCheckpointById(MatchingEventId, checkpointLog);

						if (matchingCheckpoint) {
							for (const [nodeId, nodeVecClock] of Object.entries(
								matchingCheckpoint.vectorClocks
							)) {
								if (checkpoint.vectorClocks[nodeId] <= nodeVecClock) {
									checkpoint.vectorClocks[nodeId]++;
									changesMade = true;
								}
							}
						}
					}
				}
			}
		}

		for (const [nodeIdx, nodeCheckpoints] of Object.entries(checkpointLog)) {
			for (const [checkpointIdx, checkpoint] of nodeCheckpoints.entries()) {
				const maxIndex = Math.max(...Object.values(checkpoint.vectorClocks));

				checkpoint.positionIndex = maxIndex;
			}
		}

		console.log('vector clocks complete');
		console.log(checkpointLog);

		return checkpointLog;
	};

	return (
		<>
			<Stage
				draggable
				width={window.innerWidth}
				height={window.innerHeight - 20}
			>
				<Layer>
					{orderedNodeData.map((nodeCheckpoints, nodeIndex) =>
						nodeCheckpoints.map((checkpoint, checkpointIndex) => {
							const { MatchingEventId, IsSend, positionIndex } = checkpoint;

							const receiveNode =
								MatchingEventId && IsSend
									? getNodePosition(MatchingEventId, orderedNodeData)
									: null;

							return (
								receiveNode && (
									<MessageEdge
										key={checkpoint.Id}
										checkpointIndex={positionIndex}
										nodeIndex={nodeIndex}
										matchingNode={receiveNode}
									/>
								)
							);
						})
					)}

					{rankOrder &&
						rankOrder.map(({ rank, nodeId }, idx) => {
							const { x, y } = getNodeCoordinates(idx, 0);
							return (
								<Text
									key={rank}
									text={`Rank: ${rank}\n Id:${nodeId}`}
									x={x}
									y={y}
									fontSize={13}
									fontFamily='Ubuntu'
									fill='black'
									align='center'
									width={120}
									perfectDrawEnabled
								/>
							);
						})}

					{orderedNodeData.map((nodeCheckpoints, nodeIndex) => {

						return nodeCheckpoints.map((checkpoint, checkpointIndex) => {


							return (
								<MessageNode
									key={checkpoint.Id}
									checkpointIndex={checkpoint.positionIndex}
									nodeIndex={nodeIndex}
									checkpoint={checkpoint}
									onRollbackSubmit={onRollbackSubmit}
									pendingRollback={!!pendingRollbackOriginalCheckpoint}
									isOriginalRollbackCheckpint={
										pendingRollbackOriginalCheckpoint === checkpoint.Id
									}
									pendingRollbackNodes={pendingRollbackNodes}
								/>
							);
						});
					})}
				</Layer>
			</Stage>
		</>
	);
};

function getNodeCoordinates(nodeIndex, checkpointIndex) {
	return {
		x: nodeIndex * NODE_COLUMN_WIDTH,
		y: checkpointIndex * NODE_ROW_GAP + 25,
	};
}

function getCheckpointById(id, checkpointLog) {
	for (const [nodeIdx, nodeCheckpoints] of Object.entries(checkpointLog)) {
		for (const [checkpointIdx, checkpoint] of nodeCheckpoints.entries()) {
			if (checkpoint.Id == id) {
				return {
					nodeIndex: nodeIdx,
					checkpoint,
				};
			}
		}
	}

	return { checkpoint: null };
}

function getNodePosition(id, orderedNodeData) {
	// @ts-ignore
	for (const [nodeIndex, nodeCheckpoints] of orderedNodeData.entries()) {
		for (const [checkpointIndex, checkpoint] of nodeCheckpoints.entries()) {
			if (checkpoint.Id == id) {
				return {
					nodeIndex,
					checkpointIndex: checkpoint.positionIndex,
				};
			}
		}
	}

	return null;
}

export default MessageGraph;
