package metrics

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

func CollectMemory() (*MemoryMetrics, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	valuesKB := map[string]int64{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		value, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		valuesKB[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	memTotalKB, ok := valuesKB["MemTotal"]
	if !ok || memTotalKB <= 0 {
		return nil, errors.New("MemTotal not found in /proc/meminfo")
	}

	memAvailKB, ok := valuesKB["MemAvailable"]
	if !ok {
		memFree := valuesKB["MemFree"]
		buffers := valuesKB["Buffers"]
		cached := valuesKB["Cached"]
		memAvailKB = memFree + buffers + cached
	}
	if memAvailKB < 0 {
		memAvailKB = 0
	}

	result := &MemoryMetrics{
		TotalBytes:     memTotalKB * 1024,
		AvailableBytes: memAvailKB * 1024,
	}

	swapTotalKB := valuesKB["SwapTotal"]
	swapFreeKB := valuesKB["SwapFree"]
	if swapTotalKB > 0 {
		swapUsedKB := swapTotalKB - swapFreeKB
		if swapUsedKB < 0 {
			swapUsedKB = 0
		}
		swapUsedBytes := swapUsedKB * 1024
		result.SwapUsedBytes = &swapUsedBytes
	}

	return result, nil
}
