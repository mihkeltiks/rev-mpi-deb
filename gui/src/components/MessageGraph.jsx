import React, { useEffect, useState } from 'react';
import { Stage, Layer, Circle, Text, Line } from 'react-konva';

const NODE_COLUMN_WIDTH = 120;
const NODE_SIZE = 40;
const NODE_ROW_GAP = 80;

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
				y={checkpointIndex * NODE_ROW_GAP + 90}
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
				y={y + 30}
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
				y + 65,
				matchingNodeCoords.x + 60,
				matchingNodeCoords.y + 65,
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
		if (rankOrder) {
			setOrderedNodeData(
				rankOrder.reduce((acc, { nodeId }) => {
					acc.push(checkpointLog[nodeId]);
					return acc;
				}, [])
			);
		} else setOrderedNodeData(Object.values(checkpointLog));
	}, [rankOrder, checkpointLog]);

	return (
		<>
			<Stage width={window.innerWidth} height={window.innerHeight - 20}>
				<Layer>
					{orderedNodeData.map((nodeCheckpoints, nodeIndex) =>
						nodeCheckpoints.map((checkpoint, checkpointIndex) => {
							const { MatchingEventId, IsSend } = checkpoint;

							const matchingNode =
								MatchingEventId && IsSend
									? getNodePosition(MatchingEventId, orderedNodeData)
									: null;

							return (
								matchingNode && (
									<MessageEdge
										key={checkpoint.Id}
										checkpointIndex={checkpointIndex}
										nodeIndex={nodeIndex}
										matchingNode={matchingNode}
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
									y={y - 10}
									fontSize={13}
									fontFamily='Ubuntu'
									fill='black'
									align='center'
									width={120}
									perfectDrawEnabled
								/>
							);
						})}

					{orderedNodeData.map((nodeCheckpoints, nodeIndex) =>
						nodeCheckpoints.map((checkpoint, checkpointIndex) => {
							return (
								<MessageNode
									key={checkpoint.Id}
									checkpointIndex={checkpointIndex}
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
						})
					)}
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

function getNodePosition(id, orderedNodeData) {
	// @ts-ignore
	for (const [nodeIndex, nodeCheckpoints] of orderedNodeData.entries()) {
		for (const [checkpointIndex, checkpoint] of nodeCheckpoints.entries()) {
			if (checkpoint.Id == id) {
				return { nodeIndex, checkpointIndex };
			}
		}
	}

	return null;
}

export default MessageGraph;
