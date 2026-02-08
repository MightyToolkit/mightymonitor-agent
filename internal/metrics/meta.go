package metrics

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "unknown"
	}
	return hostname
}

func GetUptime() (*int64, error) {
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return nil, err
	}
	parts := strings.Fields(string(content))
	if len(parts) < 1 {
		return nil, errors.New("unexpected /proc/uptime format")
	}
	uptimeFloat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, err
	}
	if uptimeFloat < 0 {
		uptimeFloat = 0
	}
	uptime := int64(uptimeFloat)
	return &uptime, nil
}
