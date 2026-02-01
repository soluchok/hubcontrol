package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
)

// Config represents the application configuration
type Config struct {
	Hubs []HubConfig `toml:"hubs"`
}

// HubConfig represents configuration for a specific hub
type HubConfig struct {
	VendorID      string         `toml:"vendor_id"`
	ProductID     string         `toml:"product_id"`
	Name          string         `toml:"name"`
	PhysicalPorts int            `toml:"physical_ports"`
	HiddenPorts   []string       `toml:"hidden_ports"` // Format: "child_index.port"
	PortMap       map[string]int `toml:"port_map"`     // Maps "child_index.port" -> physical port number
	GridLayout    [][]int        `toml:"grid_layout"`  // 2D array for visual layout, -1 = empty space
}

var config Config

// USBDevice represents a USB device connected to a port
type USBDevice struct {
	Bus       int       `json:"bus"`
	Device    int       `json:"device"`
	VendorID  string    `json:"vendorId"`
	ProductID string    `json:"productId"`
	Name      string    `json:"name"`
	Class     string    `json:"class"`
	Driver    string    `json:"driver"`
	Speed     string    `json:"speed"`
	Ports     []USBPort `json:"ports,omitempty"`
	// For aggregated hubs
	Aggregated    bool      `json:"aggregated,omitempty"`    // True if this is an aggregated hub
	TotalPorts    int       `json:"totalPorts,omitempty"`    // Total ports across all sub-hubs
	SubHubCount   int       `json:"subHubCount,omitempty"`   // Number of sub-hubs aggregated
	PhysicalPorts []USBPort `json:"physicalPorts,omitempty"` // All ports from sub-hubs flattened
	GridLayout    [][]int   `json:"gridLayout,omitempty"`    // 2D layout for visual display, -1 = spacer
}

// USBPort represents a port on a USB hub
type USBPort struct {
	Port   int        `json:"port"`
	Device *USBDevice `json:"device,omitempty"`
	// For aggregated view - track which physical hub this port belongs to
	HubDevice  int    `json:"hubDevice,omitempty"`  // Device number of the physical hub
	HubPort    int    `json:"hubPort,omitempty"`    // Original port number on the physical hub
	Location   string `json:"location,omitempty"`   // USB path for uhubctl
	MappedPort int    `json:"mappedPort,omitempty"` // Physical port number from config mapping
	PortKey    string `json:"portKey,omitempty"`    // Key used for port mapping (e.g., "1.3")
}

// USBBus represents a USB bus (root hub)
type USBBus struct {
	Bus    int        `json:"bus"`
	Device *USBDevice `json:"device"`
}

// USBTopology represents the complete USB topology
type USBTopology struct {
	Buses      []USBBus `json:"buses"`
	Aggregated bool     `json:"aggregated"` // Whether this is the aggregated view
}

// PowerControlRequest represents a request to control port power
type PowerControlRequest struct {
	Bus      int    `json:"bus"`
	Port     int    `json:"port"`
	Action   string `json:"action"`             // "on", "off", "cycle"
	Location string `json:"location,omitempty"` // uhubctl location parameter
}

// PowerControlResponse represents the response from power control
type PowerControlResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func main() {
	// Load configuration
	loadConfig()

	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/topology", getTopology).Methods("GET")
	api.HandleFunc("/power", controlPower).Methods("POST")
	api.HandleFunc("/uhubctl", getUhubctlInfo).Methods("GET")

	// Serve static files for frontend
	spa := spaHandler{staticPath: "../frontend/dist", indexPath: "index.html"}
	r.PathPrefix("/").Handler(spa)

	// CORS middleware for development
	handler := corsMiddleware(r)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// loadConfig loads the configuration from config.toml
func loadConfig() {
	configPaths := []string{"config.toml", "../config.toml", "/etc/hubcontrol/config.toml"}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &config); err != nil {
				log.Printf("Warning: Failed to parse config file %s: %v", path, err)
			} else {
				log.Printf("Loaded configuration from %s", path)
				return
			}
		}
	}
	log.Println("No config file found, using defaults")
}

// getHubConfig returns the configuration for a specific hub, or nil if not configured
func getHubConfig(vendorID, productID string) *HubConfig {
	for i := range config.Hubs {
		if config.Hubs[i].VendorID == vendorID && config.Hubs[i].ProductID == productID {
			return &config.Hubs[i]
		}
	}
	return nil
}

// isPortHidden checks if a port should be hidden based on configuration
func isPortHidden(hubConfig *HubConfig, childIndex, portNum int) bool {
	if hubConfig == nil {
		return false
	}
	portKey := fmt.Sprintf("%d.%d", childIndex, portNum)
	for _, hidden := range hubConfig.HiddenPorts {
		if hidden == portKey {
			return true
		}
	}
	return false
}

// getMappedPort returns the physical port number for a logical port, or 0 if not mapped
func getMappedPort(hubConfig *HubConfig, childIndex, portNum int) int {
	if hubConfig == nil || hubConfig.PortMap == nil {
		return 0
	}
	portKey := fmt.Sprintf("%d.%d", childIndex, portNum)
	if mapped, ok := hubConfig.PortMap[portKey]; ok {
		return mapped
	}
	return 0
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// spaHandler serves the single-page application
type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

// getTopology returns the USB topology by parsing lsusb output
func getTopology(w http.ResponseWriter, r *http.Request) {
	topology, err := parseUSBTopology()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if aggregated view is requested
	if r.URL.Query().Get("aggregate") == "true" {
		topology = aggregateTopology(topology)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topology)
}

// parseUSBTopology parses lsusb -t and lsusb output to build topology
func parseUSBTopology() (*USBTopology, error) {
	// Get tree structure
	treeCmd := exec.Command("lsusb", "-t")
	treeOutput, err := treeCmd.Output()
	if err != nil {
		return nil, err
	}

	// Get device details
	listCmd := exec.Command("lsusb")
	listOutput, err := listCmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse device details into a map
	deviceMap := parseDeviceList(string(listOutput))

	// Parse tree structure
	topology := parseTreeOutput(string(treeOutput), deviceMap)

	return topology, nil
}

// DeviceInfo holds parsed device information from lsusb
type DeviceInfo struct {
	VendorID  string
	ProductID string
	Name      string
}

func parseDeviceList(output string) map[string]DeviceInfo {
	devices := make(map[string]DeviceInfo)
	// Pattern: Bus 001 Device 009: ID 1a40:0201 Terminus Technology Inc. FE 2.1 7-port Hub
	re := regexp.MustCompile(`Bus (\d+) Device (\d+): ID ([0-9a-f]+):([0-9a-f]+) (.+)`)

	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if matches != nil {
			key := matches[1] + "-" + matches[2] // bus-device
			devices[key] = DeviceInfo{
				VendorID:  matches[3],
				ProductID: matches[4],
				Name:      strings.TrimSpace(matches[5]),
			}
		}
	}
	return devices
}

func parseTreeOutput(output string, deviceMap map[string]DeviceInfo) *USBTopology {
	topology := &USBTopology{
		Buses: make([]USBBus, 0),
	}

	lines := strings.Split(output, "\n")
	var currentBusIdx int = -1
	var parentStack []*USBDevice         // Stack to track parent devices at each depth
	seenDevices := make(map[string]bool) // Track seen devices to avoid duplicates

	// Pattern for root hub: /:  Bus 001.Port 001: Dev 001, Class=root_hub, Driver=xhci_hcd/6p, 480M
	busRe := regexp.MustCompile(`^/:  Bus (\d+)\.Port (\d+): Dev (\d+), Class=([^,]+), Driver=([^,]+), (\d+M?)`)

	// Pattern for device: |__ Port 003: Dev 009, If 0, Class=Hub, Driver=hub/7p, 480M
	// or:                     |__ Port 003: Dev 009, 480M (no interface info)
	deviceRe := regexp.MustCompile(`^(\s*)\|__ Port (\d+): Dev (\d+)(?:, If (\d+))?, (?:Class=([^,]+), Driver=([^,]+), )?(\d+M?)`)

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Check for root hub
		if matches := busRe.FindStringSubmatch(line); matches != nil {
			bus, _ := strconv.Atoi(matches[1])
			dev, _ := strconv.Atoi(matches[3])

			key := matches[1] + "-" + matches[3]
			info := deviceMap[key]

			numPorts := extractNumPorts(matches[5])

			device := &USBDevice{
				Bus:       bus,
				Device:    dev,
				VendorID:  info.VendorID,
				ProductID: info.ProductID,
				Name:      info.Name,
				Class:     matches[4],
				Driver:    matches[5],
				Speed:     matches[6],
				Ports:     make([]USBPort, numPorts),
			}

			// Initialize ports
			for i := 0; i < numPorts; i++ {
				device.Ports[i] = USBPort{Port: i + 1}
			}

			topology.Buses = append(topology.Buses, USBBus{
				Bus:    bus,
				Device: device,
			})
			currentBusIdx = len(topology.Buses) - 1
			parentStack = []*USBDevice{device}
			seenDevices = make(map[string]bool) // Reset for new bus
			continue
		}

		// Check for device
		if matches := deviceRe.FindStringSubmatch(line); matches != nil && currentBusIdx >= 0 {
			indent := len(matches[1])
			depth := indent / 4 // Each level is 4 spaces (first level = 4 spaces = depth 1)

			port, _ := strconv.Atoi(matches[2])
			dev, _ := strconv.Atoi(matches[3])
			ifNum := matches[4] // Interface number (may be empty)
			class := matches[5]
			driver := matches[6]
			speed := matches[7]

			busNum := topology.Buses[currentBusIdx].Bus

			// Skip duplicate interfaces of the same device (only process If 0 or first occurrence)
			deviceKey := fmt.Sprintf("%d-%d-%d", busNum, port, dev)
			if seenDevices[deviceKey] {
				continue
			}
			// Only mark as seen if this is interface 0 or no interface specified
			if ifNum == "" || ifNum == "0" {
				seenDevices[deviceKey] = true
			} else {
				continue // Skip non-zero interfaces
			}

			busStr := strconv.Itoa(busNum)
			devStr := matches[3]
			// Pad with zeros for deviceMap lookup
			if len(busStr) < 3 {
				busStr = strings.Repeat("0", 3-len(busStr)) + busStr
			}
			if len(devStr) < 3 {
				devStr = strings.Repeat("0", 3-len(devStr)) + devStr
			}
			infoKey := busStr + "-" + devStr
			info := deviceMap[infoKey]

			numPorts := 0
			if strings.Contains(class, "Hub") || strings.Contains(driver, "hub") {
				numPorts = extractNumPorts(driver)
			}

			device := &USBDevice{
				Bus:       busNum,
				Device:    dev,
				VendorID:  info.VendorID,
				ProductID: info.ProductID,
				Name:      info.Name,
				Class:     class,
				Driver:    driver,
				Speed:     speed,
			}

			if numPorts > 0 {
				device.Ports = make([]USBPort, numPorts)
				for i := 0; i < numPorts; i++ {
					device.Ports[i] = USBPort{Port: i + 1}
				}
			}

			// Find parent at depth-1 and attach device to port
			parentDepth := depth - 1
			if parentDepth >= 0 && parentDepth < len(parentStack) {
				parent := parentStack[parentDepth]
				// Find the port and attach device
				for i := range parent.Ports {
					if parent.Ports[i].Port == port && parent.Ports[i].Device == nil {
						parent.Ports[i].Device = device
						break
					}
				}

				// Update parent stack for hubs (at current depth)
				if numPorts > 0 {
					if depth >= len(parentStack) {
						parentStack = append(parentStack, device)
					} else {
						parentStack[depth] = device
					}
					// Trim stack to avoid stale entries at deeper levels
					parentStack = parentStack[:depth+1]
				}
			}
		}
	}

	return topology
}

func extractNumPorts(driver string) int {
	re := regexp.MustCompile(`/(\d+)p`)
	matches := re.FindStringSubmatch(driver)
	if matches != nil {
		n, _ := strconv.Atoi(matches[1])
		return n
	}
	return 0
}

// aggregateTopology creates a view where child hubs with the same vendor ID as their parent
// are merged into a single virtual hub showing all ports
func aggregateTopology(topology *USBTopology) *USBTopology {
	result := &USBTopology{
		Buses:      make([]USBBus, len(topology.Buses)),
		Aggregated: true,
	}

	for i, bus := range topology.Buses {
		result.Buses[i] = USBBus{
			Bus:    bus.Bus,
			Device: aggregateDevice(bus.Device, ""),
		}
	}

	return result
}

// aggregateDevice recursively processes a device and aggregates child hubs with matching vendor ID
func aggregateDevice(device *USBDevice, parentPath string) *USBDevice {
	if device == nil {
		return nil
	}

	// Build the USB path for this device
	currentPath := parentPath
	if parentPath != "" {
		currentPath = parentPath + "."
	}

	// If this device is not a hub, just return a copy
	if len(device.Ports) == 0 {
		return &USBDevice{
			Bus:       device.Bus,
			Device:    device.Device,
			VendorID:  device.VendorID,
			ProductID: device.ProductID,
			Name:      device.Name,
			Class:     device.Class,
			Driver:    device.Driver,
			Speed:     device.Speed,
		}
	}

	// Get hub configuration if available
	hubConfig := getHubConfig(device.VendorID, device.ProductID)

	// This is a hub - check if we should aggregate child hubs
	aggregatedPorts := make([]USBPort, 0)
	regularPorts := make([]USBPort, 0)
	subHubCount := 0
	childIndex := 0
	nonAggregatedPorts := make([]USBPort, 0) // Ports that aren't child hubs (empty or devices)

	for _, port := range device.Ports {
		portPath := fmt.Sprintf("%d", port.Port)
		if currentPath != "" {
			portPath = currentPath + portPath
		}

		if port.Device == nil {
			// Empty port - track it but don't add to aggregated yet
			nonAggregatedPorts = append(nonAggregatedPorts, USBPort{
				HubDevice: device.Device,
				HubPort:   port.Port,
				Location:  portPath,
				PortKey:   fmt.Sprintf("0.%d", port.Port), // Main hub direct port
			})
			regularPorts = append(regularPorts, USBPort{Port: port.Port})
		} else if isHub(port.Device) && port.Device.VendorID == device.VendorID {
			// Child hub with same vendor - aggregate its ports
			childIndex++
			subHubCount++

			// Collect all ports from this child hub recursively
			childPorts := collectAllPorts(port.Device, portPath, device.VendorID, hubConfig, childIndex)
			for _, cp := range childPorts {
				cp.Port = len(aggregatedPorts) + 1
				aggregatedPorts = append(aggregatedPorts, cp)
			}

			// Don't add to regular ports - it's being aggregated
		} else {
			// Regular device or hub with different vendor - process recursively
			processedDevice := aggregateDevice(port.Device, portPath)
			portKey := fmt.Sprintf("0.%d", port.Port) // Main hub direct port
			nonAggregatedPorts = append(nonAggregatedPorts, USBPort{
				Device:     processedDevice,
				HubDevice:  device.Device,
				HubPort:    port.Port,
				Location:   portPath,
				PortKey:    portKey,
				MappedPort: getMappedPort(hubConfig, 0, port.Port), // Check mapping for main hub (index 0)
			})
			regularPorts = append(regularPorts, USBPort{
				Port:   port.Port,
				Device: processedDevice,
			})
		}
	}

	// Create the result device
	result := &USBDevice{
		Bus:       device.Bus,
		Device:    device.Device,
		VendorID:  device.VendorID,
		ProductID: device.ProductID,
		Name:      device.Name,
		Class:     device.Class,
		Driver:    device.Driver,
		Speed:     device.Speed,
	}

	if subHubCount > 0 {
		// This hub has aggregated child hubs
		// Only include the main hub's non-aggregated ports if there are NO child hubs
		// being aggregated (meaning all ports are from child hubs)
		// This gives us just the "leaf" ports that users can actually use

		// Add non-aggregated ports (devices connected directly to main hub) to the end
		for _, nap := range nonAggregatedPorts {
			if nap.Device != nil {
				// Only include ports with actual devices, not empty internal ports
				nap.Port = len(aggregatedPorts) + 1
				aggregatedPorts = append(aggregatedPorts, nap)
			}
			// Skip empty ports on the main hub - they're likely internal/inaccessible
		}

		// Sort ports by mapped port number if port mapping is configured
		hasMappedPorts := false
		for _, p := range aggregatedPorts {
			if p.MappedPort > 0 {
				hasMappedPorts = true
				break
			}
		}

		if hasMappedPorts {
			// Collect all explicitly mapped port numbers
			usedPositions := make(map[int]bool)
			for _, p := range aggregatedPorts {
				if p.MappedPort > 0 {
					usedPositions[p.MappedPort] = true
				}
			}

			// Assign unmapped ports to available positions
			nextAvailable := 1
			for i := range aggregatedPorts {
				if aggregatedPorts[i].MappedPort == 0 {
					// Find next available position not used by explicit mappings
					for usedPositions[nextAvailable] {
						nextAvailable++
					}
					aggregatedPorts[i].MappedPort = nextAvailable
					usedPositions[nextAvailable] = true
					nextAvailable++
				}
			}

			// Now sort all ports by their mapped port number
			sort.Slice(aggregatedPorts, func(i, j int) bool {
				return aggregatedPorts[i].MappedPort < aggregatedPorts[j].MappedPort
			})
		}

		// Re-number ports sequentially after sorting
		for i := range aggregatedPorts {
			aggregatedPorts[i].Port = i + 1
		}

		result.Aggregated = true
		result.SubHubCount = subHubCount + 1 // Include self
		result.TotalPorts = len(aggregatedPorts)
		result.PhysicalPorts = aggregatedPorts
		result.Ports = regularPorts // Keep original structure too

		// Include grid layout if configured
		if hubConfig != nil && len(hubConfig.GridLayout) > 0 {
			result.GridLayout = hubConfig.GridLayout
		}

		// Update name - use custom name from config if available
		if hubConfig != nil && hubConfig.Name != "" {
			result.Name = fmt.Sprintf("%s (%d ports)", hubConfig.Name, len(aggregatedPorts))
		} else {
			result.Name = fmt.Sprintf("%s (%d ports)", device.Name, len(aggregatedPorts))
		}
	} else {
		result.Ports = regularPorts
	}

	return result
}

// isHub checks if a device is a USB hub
func isHub(device *USBDevice) bool {
	if device == nil {
		return false
	}
	return len(device.Ports) > 0 ||
		strings.Contains(strings.ToLower(device.Class), "hub") ||
		strings.Contains(strings.ToLower(device.Driver), "hub")
}

// collectAllPorts recursively collects all ports from a hub and its child hubs with matching vendor
func collectAllPorts(device *USBDevice, basePath string, vendorID string, hubConfig *HubConfig, childIndex int) []USBPort {
	result := make([]USBPort, 0)

	for _, port := range device.Ports {
		portPath := fmt.Sprintf("%s.%d", basePath, port.Port)
		portKey := fmt.Sprintf("%d.%d", childIndex, port.Port)

		// Check if this port should be hidden
		if isPortHidden(hubConfig, childIndex, port.Port) {
			continue
		}

		// Get mapped physical port number if configured
		mappedPort := getMappedPort(hubConfig, childIndex, port.Port)

		if port.Device == nil {
			// Empty port
			result = append(result, USBPort{
				HubDevice:  device.Device,
				HubPort:    port.Port,
				Location:   portPath,
				MappedPort: mappedPort,
				PortKey:    portKey,
			})
		} else if isHub(port.Device) && port.Device.VendorID == vendorID {
			// Another child hub with same vendor - recurse (shouldn't happen for your hub)
			childPorts := collectAllPorts(port.Device, portPath, vendorID, hubConfig, childIndex)
			result = append(result, childPorts...)
		} else {
			// End device or different vendor hub
			processedDevice := aggregateDevice(port.Device, portPath)
			result = append(result, USBPort{
				Device:     processedDevice,
				HubDevice:  device.Device,
				HubPort:    port.Port,
				Location:   portPath,
				MappedPort: mappedPort,
				PortKey:    portKey,
			})
		}
	}

	return result
}

// getUhubctlInfo returns information from uhubctl
func getUhubctlInfo(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("sudo", "uhubctl")
	output, err := cmd.CombinedOutput()

	response := map[string]interface{}{
		"available": err == nil,
		"output":    string(output),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// controlPower controls power on a USB port using uhubctl
func controlPower(w http.ResponseWriter, r *http.Request) {
	var req PowerControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build uhubctl command
	args := []string{"uhubctl"}

	if req.Location != "" {
		args = append(args, "-l", req.Location)
	}

	args = append(args, "-p", strconv.Itoa(req.Port))

	switch req.Action {
	case "on":
		args = append(args, "-a", "on")
	case "off":
		args = append(args, "-a", "off")
	case "cycle":
		args = append(args, "-a", "cycle")
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()

	response := PowerControlResponse{
		Success: err == nil,
		Message: string(output),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
