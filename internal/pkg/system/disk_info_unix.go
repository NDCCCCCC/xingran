//go:build !windows
// +build !windows

package system

import "syscall"

// getDiskInfoByPlatform 获取磁盘总容量和可用容量（Unix 实现）
func getDiskInfoByPlatform(path string) (uint64, uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	return st.Blocks * uint64(st.Bsize), st.Bavail * uint64(st.Bsize), nil
}
