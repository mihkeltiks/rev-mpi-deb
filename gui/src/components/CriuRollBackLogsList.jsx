import React, { useState } from 'react';
import '../styles/CriuRollbackLogsList.css'; // Import the CSS file

function CriuRollbackLogsList({ criuRollbackLogs, setCheckpointLog, currentCheckpointLog, selectedIndex, setSelectedIndex }) {
  const handleListClick = (index) => {
    setSelectedIndex(index);
    if (index === 0) {
      if (currentCheckpointLog !== undefined){
        setCheckpointLog(currentCheckpointLog);
      }
    } else {
      setCheckpointLog(criuRollbackLogs[index-1]);
    }
  };

  const sortedLogs = [currentCheckpointLog, ...criuRollbackLogs];

  return (
    <div className="list-container">
      {sortedLogs.map((log, index) => (
        <div key={index} className={`list-item ${index === selectedIndex ? 'selected' : ''}`} onClick={() => handleListClick(index)}>
          {index == 0 ? 'Current' : `Checkpoint ${index}`}
        </div>
      ))}
    </div>
  );
}

export default CriuRollbackLogsList;
