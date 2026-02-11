package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type NetworkMetrics struct {
	RxBytesPerSec float64
	TxBytesPerSec float64
}

type netState struct {
	RxBytes   uint64 `json:"rxBytes"`
	TxBytes   uint64 `json:"txBytes"`
	Timestamp int64  `json:"timestamp"`
}

// CollectNetwork reads /proc/net/dev, sums rx/tx bytes for all non-loopback
// interfaces, computes bytes/sec from the previous state, and persists state.
// Returns nil on first run, counter reset, or elapsed > 300s.
func CollectNetwork(stateDir string) (*NetworkMetrics, error) {
	rx, tx, err := readProcNetDev()
	if err != nil {
		return nil, fmt.Errorf("read /proc/net/dev: %w", err)
	}

	now := time.Now().Unix()
	stateFile := filepath.Join(stateDir, "net_state.json")

	prev, err := loadNetState(stateFile)

	current := netState{RxBytes: rx, TxBytes: tx, Timestamp: now}
	if saveErr := saveNetState(stateFile, current); saveErr != nil {
		return nil, fmt.Errorf("save net state: %w", saveErr)
	}

	if err != nil {
		// First run or corrupt state file
		return nil, nil
	}

	elapsed := float64(now - prev.Timestamp)
	if elapsed <= 0 || elapsed > 300 {
		return nil, nil
	}

	// Counter reset detection
	if rx < prev.RxBytes || tx < prev.TxBytes {
		return nil, nil
	}

	return &NetworkMetrics{
		RxBytesPerSec: float64(rx-prev.RxBytes) / elapsed,
		TxBytesPerSec: float64(tx-prev.TxBytes) / elapsed,
	}, nil
}

func readProcNetDev() (rxTotal uint64, txTotal uint64, err error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		rx, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		tx, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}

		rxTotal += rx
		txTotal += tx
	}

	return rxTotal, txTotal, scanner.Err()
}

func loadNetState(path string) (*netState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state netState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveNetState(path string, state netState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
