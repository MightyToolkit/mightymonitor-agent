package metrics

import (
	"log"
	"time"
)

func Collect() (*Payload, error) {
	payload := &Payload{
		Hostname: GetHostname(),
		TS:       time.Now().Unix(),
	}

	cpu, err := CollectCPU()
	if err != nil {
		log.Printf("WARN metrics cpu collection failed: %v", err)
	} else {
		payload.CPU = CPUPayload{
			Load1:  cpu.Load1,
			Load5:  cpu.Load5,
			Load15: cpu.Load15,
			Cores:  cpu.Cores,
		}
	}

	memory, err := CollectMemory()
	if err != nil {
		log.Printf("WARN metrics memory collection failed: %v", err)
	} else {
		payload.Memory = MemoryPayload{
			TotalBytes:     memory.TotalBytes,
			AvailableBytes: memory.AvailableBytes,
			SwapUsedBytes:  memory.SwapUsedBytes,
		}
	}

	disk, err := CollectDisk()
	if err != nil {
		log.Printf("WARN metrics disk collection failed: %v", err)
	} else {
		payload.Disk = DiskPayload{
			TotalBytes: disk.TotalBytes,
			FreeBytes:  disk.FreeBytes,
		}
	}

	network, err := CollectNetwork("/var/lib/mightymonitor")
	if err != nil {
		log.Printf("WARN metrics network collection failed: %v", err)
	} else if network != nil {
		payload.Network = &NetworkPayload{
			RxBytesPerSec: network.RxBytesPerSec,
			TxBytesPerSec: network.TxBytesPerSec,
		}
	}

	uptime, err := GetUptime()
	if err != nil {
		log.Printf("WARN metrics uptime collection failed: %v", err)
	} else {
		payload.UptimeSeconds = uptime
	}

	return payload, nil
}
