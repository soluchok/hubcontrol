// USB Device types matching the Go backend

export interface USBDevice {
  bus: number;
  device: number;
  vendorId: string;
  productId: string;
  name: string;
  class: string;
  driver: string;
  speed: string;
  ports?: USBPort[];
  // Aggregation fields
  aggregated?: boolean;
  totalPorts?: number;
  subHubCount?: number;
  physicalPorts?: USBPort[];
  gridLayout?: number[][]; // 2D layout for visual display, -1 = spacer
}

export interface USBPort {
  port: number;
  device?: USBDevice;
  // For aggregated view
  hubDevice?: number;
  hubPort?: number;
  location?: string;
  mappedPort?: number;  // Physical port number from config mapping
  portKey?: string;     // Key used for port mapping (e.g., "1.3")
}

export interface USBBus {
  bus: number;
  device: USBDevice;
}

export interface USBTopology {
  buses: USBBus[];
  aggregated?: boolean;
}

export interface PowerControlRequest {
  bus: number;
  port: number;
  action: 'on' | 'off' | 'cycle';
  location?: string;
}

export interface PowerControlResponse {
  success: boolean;
  message: string;
}
