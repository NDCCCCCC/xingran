//go:build !windows
// +build !windows

package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getSystemMemoryInfo 获取Unix-like系统的内存信息
func getSystemMemoryInfo() (uint64, uint64, error) {
	// 读取/proc/meminfo文件
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, fmt.Errorf("无法打开/proc/meminfo文件: %w", err)
	}
	defer file.Close()

	var totalKB, availableKB uint64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				totalKB = val
			}
		case "MemAvailable:":
			if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				availableKB = val
			}
		}

		// 如果已经获取到两个值，可以退出循环
		if totalKB > 0 && availableKB > 0 {
			break
		}
	}

	if totalKB == 0 {
		return 0, 0, fmt.Errorf("无法从/proc/meminfo获取内存总量信息")
	}

	// 转换为字节
	totalBytes := totalKB * 1024
	availableBytes := availableKB * 1024
	usedBytes := totalBytes - availableBytes

	return totalBytes, usedBytes, nil
}

// getEstimatedMemoryInfo 获取估算的内存信息（备用方案）
func getEstimatedMemoryInfo() (uint64, uint64, error) {
	// 假设系统有32GB内存，80%使用
	totalBytes := uint64(32) * 1024 * 1024 * 1024 // 32GB
	usedBytes := totalBytes * 80 / 100            // 80%使用

	return totalBytes, usedBytes, nil
}
