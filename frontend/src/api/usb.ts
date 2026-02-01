import type { USBTopology, PowerControlRequest, PowerControlResponse } from '../types/usb';

const API_BASE = '/api';

export async function fetchTopology(aggregate: boolean = false): Promise<USBTopology> {
  const url = aggregate ? `${API_BASE}/topology?aggregate=true` : `${API_BASE}/topology`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch USB topology');
  }
  return response.json();
}

export async function controlPower(request: PowerControlRequest): Promise<PowerControlResponse> {
  const response = await fetch(`${API_BASE}/power`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });
  if (!response.ok) {
    throw new Error('Failed to control power');
  }
  return response.json();
}

export async function fetchUhubctlInfo(): Promise<{ available: boolean; output: string }> {
  const response = await fetch(`${API_BASE}/uhubctl`);
  if (!response.ok) {
    throw new Error('Failed to fetch uhubctl info');
  }
  return response.json();
}
