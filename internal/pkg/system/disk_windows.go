//go:build windows
// +build windows

package system

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32Disk            = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceExW = kernel32Disk.NewProc("GetDiskFreeSpaceExW")
)

// getDiskUsageByPlatform 获取Windows系统的磁盘使用率
func getDiskUsageByPlatform(path string) (float64, error) {
	// 将路径转换为UTF-16指针
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	var freeBytes, totalBytes, freeBytesAvailable uint64

	// 调用Windows API获取磁盘空间信息
	ret, _, err := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&freeBytes)),
	)

	if ret == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW failed: %v", err)
	}

	if totalBytes == 0 {
		return 0, fmt.Errorf("invalid disk size")
	}

	// 计算使用率
	used := totalBytes - freeBytes
	usage := float64(used) / float64(totalBytes) * 100

	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}
