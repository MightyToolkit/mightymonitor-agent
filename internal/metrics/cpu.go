package metrics

import (
	"bufio"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func CollectCPU() (*CPUMetrics, error) {
	content, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	parts := strings.Fields(string(content))
	if len(parts) < 3 {
		return nil, errors.New("unexpected /proc/loadavg format")
	}

	load1, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, err
	}
	load5, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, err
	}
	load15, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, err
	}

	cores := collectCoresFromCPUInfo()
	if cores < 1 {
		cores = runtime.NumCPU()
	}
	if cores < 1 {
		cores = 1
	}

	return &CPUMetrics{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
		Cores:  cores,
	}, nil
}

func collectCoresFromCPUInfo() int {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "processor") {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0
	}
	return count
}
