package metrics

import "syscall"

func CollectDisk() (*DiskMetrics, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return nil, err
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	if total < 0 {
		total = 0
	}
	if free < 0 {
		free = 0
	}

	return &DiskMetrics{
		TotalBytes: total,
		FreeBytes:  free,
	}, nil
}
