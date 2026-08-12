//go:build windows
// +build windows

package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getAllDiskInfoByPlatform 获取Windows系统所有磁盘信息
func getAllDiskInfoByPlatform() ([]DiskInfo, error) {
	var disks []DiskInfo

	// 首先尝试使用PowerShell获取更可靠的磁盘信息
	if disks, err := getDisksViaPowerShell(); err == nil && len(disks) > 0 {
		return disks, nil
	}

	// 如果PowerShell失败，回退到WMIC
	cmd := exec.Command("wmic", "logicaldisk", "get", "DeviceID,Size,FreeSpace", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行wmic磁盘命令失败: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "DeviceID") || strings.Contains(line, "Node,") {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) >= 4 {
			// WMIC CSV格式：Node,DeviceID,FreeSpace,Size
			deviceID := strings.Trim(fields[1], `"`)
			freeStr := strings.Trim(fields[2], `"`)
			sizeStr := strings.Trim(fields[3], `"`)

			if deviceID == "" || sizeStr == "" || freeStr == "" {
				continue
			}

			// 转换大小（字节数）
			totalBytes, err1 := strconv.ParseUint(sizeStr, 10, 64)
			freeBytes, err2 := strconv.ParseUint(freeStr, 10, 64)

			if err1 == nil && err2 == nil && totalBytes > 0 {
				// 确保可用空间不超过总容量
				if freeBytes > totalBytes {
					continue // 跳过异常数据
				}

				usedBytes := totalBytes - freeBytes

				disk := DiskInfo{
					MountPoint: deviceID,
					Total:      totalBytes,
					Available:  freeBytes,
					Used:       usedBytes,
				}
				disks = append(disks, disk)
			}
		}
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("未获取到任何磁盘信息")
	}

	return disks, nil
}

// getDisksViaPowerShell 使用PowerShell获取磁盘信息
func getDisksViaPowerShell() ([]DiskInfo, error) {
	cmd := exec.Command("powershell", "-Command", "Get-WmiObject Win32_LogicalDisk | Where-Object {$_.DriveType -eq 3} | Select-Object DeviceID, Size, FreeSpace")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行PowerShell磁盘命令失败: %w", err)
	}

	var disks []DiskInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "DeviceID") || strings.Contains(line, "----") || !strings.Contains(line, ":") {
			continue
		}

		// PowerShell输出格式解析 (空格分隔的列)
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			deviceID := fields[0]
			sizeStr := fields[1]
			freeStr := fields[2]

			// 清理可能存在的空格和特殊字符
			deviceID = strings.TrimSpace(deviceID)
			sizeStr = strings.TrimSpace(sizeStr)
			freeStr = strings.TrimSpace(freeStr)

			if deviceID == "" || sizeStr == "" || freeStr == "" {
				continue
			}

			// 转换大小（字节数）
			totalBytes, err1 := strconv.ParseUint(sizeStr, 10, 64)
			freeBytes, err2 := strconv.ParseUint(freeStr, 10, 64)

			if err1 == nil && err2 == nil && totalBytes > 0 {
				// 确保可用空间不超过总容量
				if freeBytes > totalBytes {
					continue // 跳过异常数据
				}

				usedBytes := totalBytes - freeBytes

				disk := DiskInfo{
					MountPoint: deviceID,
					Total:      totalBytes,
					Available:  freeBytes,
					Used:       usedBytes,
				}
				disks = append(disks, disk)
			}
		}
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("PowerShell未获取到任何磁盘信息")
	}

	return disks, nil
}
