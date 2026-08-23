//go:build windows
// +build windows

package system

import (
	"fmt"
	"syscall"
	"unsafe"
)

// getDiskInfoByPlatform 获取磁盘总容量和可用容量（Windows 实现）
func getDiskInfoByPlatform(path string) (uint64, uint64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}

	var freeBytes, totalBytes, freeBytesAvailable uint64

	ret, _, err := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&freeBytes)),
	)

	if ret == 0 {
		return 0, 0, fmt.Errorf("GetDiskFreeSpaceExW failed: %v", err)
	}

	return totalBytes, freeBytesAvailable, nil
}
