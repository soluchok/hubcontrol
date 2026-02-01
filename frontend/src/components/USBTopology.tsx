import type { USBBus, USBDevice, USBPort } from '../types/usb';
import { USBHub } from './USBHub';
import './USBTopology.css';

interface USBTopologyViewProps {
  buses: USBBus[];
  onRefresh: () => void;
  loading: boolean;
  aggregated: boolean;
  onToggleAggregated: (value: boolean) => void;
  onPortClick?: (device: USBDevice, port: USBPort) => void;
}

export function USBTopologyView({ 
  buses, 
  onRefresh, 
  loading, 
  aggregated,
  onToggleAggregated,
  onPortClick
}: USBTopologyViewProps) {
  const stats = calculateStats(buses);

  return (
    <div className="topology-view">
      <div className="topology-header">
        <h1>USB Hub Control</h1>
        <div className="header-controls">
          <label className="toggle-label">
            <input 
              type="checkbox" 
              checked={aggregated}
              onChange={(e) => onToggleAggregated(e.target.checked)}
            />
            <span className="toggle-switch"></span>
            <span className="toggle-text">Combine Hubs</span>
          </label>
          <button 
            className="refresh-btn" 
            onClick={onRefresh} 
            disabled={loading}
          >
            {loading ? 'Scanning...' : 'Refresh'}
          </button>
        </div>
      </div>
      
      <div className="topology-stats">
        <div className="stat">
          <span className="stat-value">{buses.length}</span>
          <span className="stat-label">USB Buses</span>
        </div>
        <div className="stat">
          <span className="stat-value">{stats.totalDevices}</span>
          <span className="stat-label">Devices</span>
        </div>
        <div className="stat">
          <span className="stat-value">{stats.totalHubs}</span>
          <span className="stat-label">Hubs</span>
        </div>
        <div className="stat">
          <span className="stat-value">{stats.totalPorts}</span>
          <span className="stat-label">Total Ports</span>
        </div>
        <div className="stat">
          <span className="stat-value">{stats.usedPorts}</span>
          <span className="stat-label">Used</span>
        </div>
      </div>

      {aggregated && (
        <div className="aggregation-notice">
          Showing combined view - hubs from the same manufacturer are merged into single units
        </div>
      )}

      <div className="buses-container">
        {buses.map((bus) => (
          <div key={bus.bus} className="bus-section">
            <div className="bus-header">
              <span className="bus-icon">🔲</span>
              <span className="bus-title">Bus {bus.bus}</span>
              <span className="bus-type">
                {bus.device.speed?.includes('10000') ? 'USB 3.1' :
                 bus.device.speed?.includes('5000') ? 'USB 3.0' :
                 bus.device.speed?.includes('480') ? 'USB 2.0' : 'USB 1.1'}
              </span>
            </div>
            <USBHub device={bus.device} depth={0} onPortClick={onPortClick} />
          </div>
        ))}
      </div>
      
      {buses.length === 0 && !loading && (
        <div className="no-devices">
          <p>No USB devices found. Make sure the backend is running.</p>
        </div>
      )}
    </div>
  );
}

interface Stats {
  totalDevices: number;
  totalHubs: number;
  totalPorts: number;
  usedPorts: number;
}

function calculateStats(buses: USBBus[]): Stats {
  let totalDevices = 0;
  let totalHubs = 0;
  let totalPorts = 0;
  let usedPorts = 0;

  function processDevice(device: USBDevice) {
    totalDevices++;
    
    // Use physicalPorts for aggregated hubs, otherwise use ports
    const ports = device.physicalPorts || device.ports;
    
    if (ports && ports.length > 0) {
      totalHubs++;
      totalPorts += ports.length;
      
      for (const port of ports) {
        if (port.device) {
          usedPorts++;
          processDevice(port.device);
        }
      }
    }
  }

  for (const bus of buses) {
    processDevice(bus.device);
  }

  return { totalDevices, totalHubs, totalPorts, usedPorts };
}
