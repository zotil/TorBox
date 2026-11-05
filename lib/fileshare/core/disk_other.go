//go:build !darwin && !linux

package core

// GetDiskUsage is a fallback implementation for non-Darwin, non-Linux platforms.
func GetDiskUsage(path string) (total uint64, free uint64, used uint64, err error) {
	return 0, 0, 0, nil
}
