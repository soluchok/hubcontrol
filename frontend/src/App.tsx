import { useState, useEffect, useCallback } from 'react';
import { USBTopologyView } from './components/USBTopology';
import { PowerControl } from './components/PowerControl';
import { fetchTopology } from './api/usb';
import type { USBTopology, USBDevice, USBPort } from './types/usb';
import './App.css';

interface SelectedPort {
  device: USBDevice;
  port: USBPort;
}

function App() {
  const [topology, setTopology] = useState<USBTopology | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [aggregated, setAggregated] = useState(true); // Default to aggregated view
  const [selectedPort, setSelectedPort] = useState<SelectedPort | null>(null);

  const loadTopology = useCallback(async (aggregate: boolean) => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchTopology(aggregate);
      setTopology(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load USB topology');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTopology(aggregated);
  }, [aggregated, loadTopology]);

  const handleToggleAggregated = (value: boolean) => {
    setAggregated(value);
  };

  const handleRefresh = () => {
    loadTopology(aggregated);
  };

  const handlePortClick = (device: USBDevice, port: USBPort) => {
    setSelectedPort({ device, port });
  };

  const handleClosePowerControl = () => {
    setSelectedPort(null);
  };

  const handlePowerActionComplete = () => {
    // Refresh topology after power action
    loadTopology(aggregated);
  };

  if (error) {
    return (
      <div className="app-error">
        <h2>Error</h2>
        <p>{error}</p>
        <button onClick={handleRefresh}>Retry</button>
      </div>
    );
  }

  return (
    <div className="app">
      <USBTopologyView 
        buses={topology?.buses || []} 
        onRefresh={handleRefresh}
        loading={loading}
        aggregated={aggregated}
        onToggleAggregated={handleToggleAggregated}
        onPortClick={handlePortClick}
      />
      
      {selectedPort && (
        <PowerControl
          device={selectedPort.device}
          port={selectedPort.port}
          onClose={handleClosePowerControl}
          onActionComplete={handlePowerActionComplete}
        />
      )}
    </div>
  );
}

export default App;
