import type { USBDevice, USBPort } from '../types/usb';
import './USBHub.css';

interface USBHubProps {
  device: USBDevice;
  depth?: number;
  onPortClick?: (device: USBDevice, port: USBPort) => void;
}

export function USBHub({ device, depth = 0, onPortClick }: USBHubProps) {
  const isHub = (device.ports && device.ports.length > 0) || device.aggregated;
  const isAggregated = device.aggregated && device.physicalPorts;
  
  // Use physical ports for aggregated hubs, otherwise use regular ports
  const displayPorts = isAggregated ? device.physicalPorts! : (device.ports || []);
  const hasConnectedDevices = displayPorts.some(p => p.device);

  // Check if we have a custom grid layout
  const hasGridLayout = device.gridLayout && device.gridLayout.length > 0;

  // Create a map of port number -> port data for grid layout rendering
  const portMap = new Map<number, USBPort>();
  displayPorts.forEach(port => {
    const portNum = port.mappedPort || port.port;
    portMap.set(portNum, port);
  });

  const getSpeedClass = (speed: string) => {
    if (speed.includes('10000') || speed.includes('5000')) return 'speed-super';
    if (speed.includes('480')) return 'speed-high';
    if (speed.includes('12')) return 'speed-full';
    return 'speed-low';
  };

  const getDeviceIcon = (dev?: USBDevice) => {
    const d = dev || device;
    const className = d.class?.toLowerCase() || '';
    const name = d.name?.toLowerCase() || '';
    
    // Return CSS class for device type indicator
    if (className.includes('hub') || name.includes('hub')) return 'icon-hub';
    if (className.includes('audio') || name.includes('audio') || name.includes('microphone')) return 'icon-audio';
    if (className.includes('video') || name.includes('camera')) return 'icon-video';
    if (className.includes('hid') || name.includes('keyboard')) return 'icon-input';
    if (name.includes('mouse') || name.includes('logitech')) return 'icon-input';
    if (className.includes('wireless') || name.includes('bluetooth')) return 'icon-wireless';
    if (name.includes('ethernet') || name.includes('network')) return 'icon-network';
    if (className.includes('storage') || name.includes('storage')) return 'icon-storage';
    return 'icon-device';
  };

  const getPortTooltip = (port: USBPort) => {
    const parts: string[] = [];
    
    // Show mapped port number if different from display port
    if (port.mappedPort && port.mappedPort !== port.port) {
      parts.push(`Physical Port ${port.mappedPort}`);
    } else {
      parts.push(`Port ${port.port}`);
    }
    
    // Show port key for mapping discovery
    if (port.portKey) {
      parts.push(`Key: ${port.portKey}`);
    }
    
    if (port.device) {
      parts.push(port.device.name);
    } else {
      parts.push('Empty');
    }
    
    if (isAggregated && port.location) {
      parts.push(`Location: ${port.location}`);
    }
    
    return parts.join(' | ');
  };

  // Render a single port cell (used in both grid modes)
  const renderPort = (port: USBPort, key: string | number) => {
    const displayNumber = port.mappedPort || port.port;
    return (
      <div 
        key={key} 
        className={`port ${port.device ? 'occupied' : 'empty'}`}
        onClick={() => onPortClick?.(device, port)}
        title={getPortTooltip(port)}
      >
        <span className="port-number">{displayNumber}</span>
        {port.device ? (
          <span className={`port-device-icon ${getDeviceIcon(port.device)}`}></span>
        ) : (
          <span className="port-empty-indicator"></span>
        )}
      </div>
    );
  };

  // Render a spacer cell for grid layout
  const renderSpacer = (key: string) => (
    <div key={key} className="port-spacer"></div>
  );

  // Render the custom grid layout
  const renderGridLayout = () => {
    if (!device.gridLayout) return null;
    
    const maxCols = Math.max(...device.gridLayout.map(row => row.length));
    
    return (
      <div 
        className="ports-grid-custom"
        style={{ gridTemplateColumns: `repeat(${maxCols}, minmax(42px, 60px))` }}
      >
        {device.gridLayout.map((row, rowIdx) => 
          row.map((portNum, colIdx) => {
            const key = `${rowIdx}-${colIdx}`;
            if (portNum === -1) {
              return renderSpacer(key);
            }
            const port = portMap.get(portNum);
            if (port) {
              return renderPort(port, key);
            }
            // Port number in layout but not found in ports - render as missing
            return (
              <div key={key} className="port missing" title={`Port ${portNum} not found`}>
                <span className="port-number">{portNum}</span>
                <span className="port-empty-indicator"></span>
              </div>
            );
          })
        )}
      </div>
    );
  };

  // Render the default auto grid
  const renderAutoGrid = () => (
    <div className={`ports-grid ${displayPorts.length > 12 ? 'large-grid' : ''}`}>
      {displayPorts.map((port) => renderPort(port, port.port))}
    </div>
  );

  return (
    <div className={`usb-device depth-${Math.min(depth, 5)}`}>
      <div className={`device-header ${isHub ? 'hub' : 'endpoint'} ${getSpeedClass(device.speed)} ${isAggregated ? 'aggregated' : ''}`}>
        <span className={`device-icon ${getDeviceIcon()}`}></span>
        <div className="device-info">
          <span className="device-name">
            {device.name || 'Unknown Device'}
            {isAggregated && (
              <span className="aggregated-badge" title={`${device.subHubCount} hubs combined`}>
                COMBINED
              </span>
            )}
          </span>
          <span className="device-details">
            {device.vendorId && device.productId && (
              <span className="device-id">{device.vendorId}:{device.productId}</span>
            )}
            <span className="device-speed">{device.speed}</span>
            {device.class && <span className="device-class">{device.class}</span>}
            {isAggregated && device.subHubCount && (
              <span className="sub-hub-count">{device.subHubCount} hubs</span>
            )}
          </span>
        </div>
        <span className="bus-info">Bus {device.bus} Dev {device.device}</span>
      </div>
      
      {isHub && displayPorts.length > 0 && (
        <div className="ports-container">
          <div className="ports-header">
            <span>{displayPorts.length} Ports</span>
            {hasConnectedDevices && (
              <span className="connected-count">
                ({displayPorts.filter(p => p.device).length} connected)
              </span>
            )}
          </div>
          
          {hasGridLayout ? renderGridLayout() : renderAutoGrid()}
          
          {/* Render connected devices */}
          <div className="connected-devices">
            {displayPorts
              .filter(port => port.device)
              .map(port => {
                const displayNumber = port.mappedPort || port.port;
                return (
                  <div key={port.port} className="port-connection">
                    <div className="connection-line">
                      <span className="port-label">
                        Port {displayNumber}
                        {port.portKey && <span className="port-key"> ({port.portKey})</span>}
                      </span>
                    </div>
                    <USBHub 
                      device={port.device!} 
                      depth={depth + 1} 
                      onPortClick={onPortClick}
                    />
                  </div>
                );
              })}
          </div>
        </div>
      )}
    </div>
  );
}
