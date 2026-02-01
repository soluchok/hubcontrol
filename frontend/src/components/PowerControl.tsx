import { useState } from 'react';
import type { USBDevice, USBPort } from '../types/usb';
import { controlPower } from '../api/usb';
import './PowerControl.css';

interface PowerControlProps {
  device: USBDevice;
  port: USBPort;
  onClose: () => void;
  onActionComplete: () => void;
}

export function PowerControl({ device, port, onClose, onActionComplete }: PowerControlProps) {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null);

  const displayNumber = port.mappedPort || port.port;
  
  // For uhubctl, we need the location (USB path) and the port number on that hub
  // The location should be the parent hub's path, and port is hubPort
  const getUhubctlParams = () => {
    // Location is the path to the hub (e.g., "1.2" for a hub at bus 1, port 2)
    // We need to strip the last segment from port.location to get the hub path
    let hubLocation = '';
    if (port.location) {
      const parts = port.location.split('.');
      if (parts.length > 1) {
        hubLocation = parts.slice(0, -1).join('.');
      } else {
        hubLocation = parts[0];
      }
    }
    return {
      location: hubLocation,
      port: port.hubPort || port.port
    };
  };

  const handleAction = async (action: 'on' | 'off' | 'cycle') => {
    setLoading(true);
    setResult(null);

    const params = getUhubctlParams();
    
    try {
      const response = await controlPower({
        bus: device.bus,
        port: params.port,
        action,
        location: params.location
      });
      setResult(response);
      if (response.success) {
        // Refresh topology after successful action
        setTimeout(() => {
          onActionComplete();
        }, 1000);
      }
    } catch (error) {
      setResult({
        success: false,
        message: error instanceof Error ? error.message : 'Unknown error'
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="power-control-overlay" onClick={onClose}>
      <div className="power-control-modal" onClick={e => e.stopPropagation()}>
        <div className="power-control-header">
          <h3>Port {displayNumber} Power Control</h3>
          <button className="close-btn" onClick={onClose}>&times;</button>
        </div>
        
        <div className="power-control-info">
          <div className="info-row">
            <span className="info-label">Port Key:</span>
            <span className="info-value">{port.portKey || 'N/A'}</span>
          </div>
          <div className="info-row">
            <span className="info-label">Location:</span>
            <span className="info-value">{port.location || 'N/A'}</span>
          </div>
          <div className="info-row">
            <span className="info-label">Hub Port:</span>
            <span className="info-value">{port.hubPort || port.port}</span>
          </div>
          {port.device && (
            <div className="info-row">
              <span className="info-label">Device:</span>
              <span className="info-value">{port.device.name}</span>
            </div>
          )}
        </div>

        <div className="power-control-actions">
          <button 
            className="power-btn power-on"
            onClick={() => handleAction('on')}
            disabled={loading}
          >
            Power ON
          </button>
          <button 
            className="power-btn power-off"
            onClick={() => handleAction('off')}
            disabled={loading}
          >
            Power OFF
          </button>
          <button 
            className="power-btn power-cycle"
            onClick={() => handleAction('cycle')}
            disabled={loading}
          >
            Cycle
          </button>
        </div>

        {loading && (
          <div className="power-control-status loading">
            Executing command...
          </div>
        )}

        {result && (
          <div className={`power-control-status ${result.success ? 'success' : 'error'}`}>
            <div className="status-header">
              {result.success ? 'Success' : 'Error'}
            </div>
            <pre className="status-output">{result.message}</pre>
          </div>
        )}
      </div>
    </div>
  );
}
