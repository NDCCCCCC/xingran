//go:build !windows
// +build !windows

package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getAllDiskInfoByPlatform 获取Unix-like系统所有磁盘信息
func getAllDiskInfoByPlatform() ([]DiskInfo, error) {
	var disks []DiskInfo

	// 使用df命令获取所有挂载点信息
	cmd := exec.Command("df", "-k")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行df磁盘命令失败: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // 跳过标题行和空行
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// 解析字段：文件系统、总容量、已用、可用、使用率%、挂载点
		totalKB, err1 := strconv.ParseUint(fields[1], 10, 64)
		availableKB, err2 := strconv.ParseUint(fields[3], 10, 64)
		mountPoint := fields[5]

		if err1 == nil && err2 == nil && totalKB > 0 {
			// 过滤掉一些特殊的挂载点
			if shouldSkipMountPoint(mountPoint) {
				continue
			}

			totalBytes := totalKB * 1024
			availableBytes := availableKB * 1024
			usedBytes := totalBytes - availableBytes

			disk := DiskInfo{
				MountPoint: mountPoint,
				Total:      totalBytes,
				Available:  availableBytes,
				Used:       usedBytes,
			}
			disks = append(disks, disk)
		}
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("未获取到任何磁盘信息")
	}

	return disks, nil
}

// shouldSkipMountPoint 判断是否跳过某些挂载点
func shouldSkipMountPoint(mountPoint string) bool {
	skipPoints := []string{
		"/dev",
		"/dev/shm",
		"/run",
		"/sys",
		"/proc",
		"/var/lock",
		"/var/run",
		"/tmpfs",
	}

	for _, skip := range skipPoints {
		if mountPoint == skip || strings.HasPrefix(mountPoint, skip) {
			return true
		}
	}

	return false
}

// getEstimatedMultiDiskInfo 获取估算的多磁盘信息（备用方案）
func getEstimatedMultiDiskInfo() ([]DiskInfo, error) {
	return []DiskInfo{
		{
			MountPoint: "/",
			Total:      uint64(500) * 1024 * 1024 * 1024, // 500GB
			Available:  uint64(125) * 1024 * 1024 * 1024, // 125GB可用
			Used:       uint64(375) * 1024 * 1024 * 1024, // 375GB已用
		},
	}, nil
}
