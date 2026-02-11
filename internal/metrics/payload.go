package metrics

type Payload struct {
	HostID        string          `json:"hostId"`
	Hostname      string          `json:"hostname"`
	AgentVersion  string          `json:"agentVersion"`
	TS            int64           `json:"ts"`
	CPU           CPUPayload      `json:"cpu"`
	Memory        MemoryPayload   `json:"memory"`
	Disk          DiskPayload     `json:"disk"`
	Network       *NetworkPayload `json:"network,omitempty"`
	UptimeSeconds *int64          `json:"uptimeSeconds,omitempty"`
}

type CPUPayload struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
	Cores  int     `json:"cores"`
}

type MemoryPayload struct {
	TotalBytes     int64  `json:"totalBytes"`
	AvailableBytes int64  `json:"availableBytes"`
	SwapUsedBytes  *int64 `json:"swapUsedBytes,omitempty"`
}

type DiskPayload struct {
	TotalBytes int64 `json:"totalBytes"`
	FreeBytes  int64 `json:"freeBytes"`
}

// Internal collector metrics

type CPUMetrics struct {
	Load1  float64
	Load5  float64
	Load15 float64
	Cores  int
}

type MemoryMetrics struct {
	TotalBytes     int64
	AvailableBytes int64
	SwapUsedBytes  *int64
}

type DiskMetrics struct {
	TotalBytes int64
	FreeBytes  int64
}

type NetworkPayload struct {
	RxBytesPerSec float64 `json:"rxBytesPerSec"`
	TxBytesPerSec float64 `json:"txBytesPerSec"`
}
