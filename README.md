# USB Hub Control

A visual tool to monitor and control USB hubs and ports. Built with Go backend and React frontend.

## Features

- Visual tree display of all USB buses, hubs, and devices
- **Hub aggregation**: Multi-hub setups (e.g., 20-port hub = 7-port + 6x4-port) shown as single unit
- Real-time topology scanning using `lsusb`
- Port power control via `uhubctl` (requires root/sudo)
- Device information display (vendor ID, product ID, speed, class)
- Configurable port hiding for internal/inaccessible ports

## Requirements

- Go 1.21+
- Node.js 18+
- `lsusb` (usually pre-installed on Linux)
- `uhubctl` (optional, for power control)

### Installing uhubctl

```bash
# Ubuntu/Debian
sudo apt install uhubctl

# From source
git clone https://github.com/mvp/uhubctl
cd uhubctl && make && sudo make install
```

## Quick Start

### Development Mode

Terminal 1 - Backend:
```bash
cd backend && go run .
```

Terminal 2 - Frontend:
```bash
cd frontend && npm run dev
```

Then open http://localhost:5173

### Production Build

```bash
make all
./bin/hubcontrol
```

Then open http://localhost:8080

## Configuration

Create a `config.toml` file to customize hub display:

```toml
[[hubs]]
vendor_id = "1a40"
product_id = "0201"
name = "My 20-Port Hub"
physical_ports = 20

# Hide internal/inaccessible ports
# Format: "child_hub_index.port_number"
hidden_ports = [
  "5.4",  # Child hub 5, port 4
  "6.3",  # Child hub 6, port 3
  "6.4",  # Child hub 6, port 4
  "1.4",  # Child hub 1, port 4
]
```

The config file is searched in:
1. `./config.toml`
2. `../config.toml`
3. `/etc/hubcontrol/config.toml`

## API Endpoints

- `GET /api/topology` - Returns USB topology as JSON
- `GET /api/topology?aggregate=true` - Returns aggregated topology (hubs combined)
- `POST /api/power` - Control port power (requires uhubctl + sudo)
- `GET /api/uhubctl` - Check uhubctl availability

## Project Structure

```
hubcontrol/
├── backend/           # Go backend
│   ├── main.go        # Server and API handlers
│   └── go.mod         # Go module
├── frontend/          # React frontend
│   ├── src/
│   │   ├── components/  # React components
│   │   ├── api/         # API client
│   │   └── types/       # TypeScript types
│   └── package.json
├── config.toml        # Hub configuration
└── Makefile           # Build commands
```

## Notes

- Power control requires `uhubctl` and sudo access
- Not all USB hubs support power control
- The tool uses `lsusb -t` for topology discovery
- Toggle "Combine Hubs" in the UI to switch between tree and flat view
