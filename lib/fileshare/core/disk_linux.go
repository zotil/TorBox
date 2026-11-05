//go:build linux

package core

import "syscall"

// GetDiskUsage returns total, free (available to unprivileged users), and used bytes.
func GetDiskUsage(path string) (total uint64, free uint64, used uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, err
	}
	bsize := uint64(stat.Bsize)
	total = uint64(stat.Blocks) * bsize
	free = uint64(stat.Bavail) * bsize
	used = (uint64(stat.Blocks) - uint64(stat.Bfree)) * bsize
	return total, free, used, nil
}
