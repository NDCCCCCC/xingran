//go:build linux
// +build linux

package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CPUStats CPU统计信息
type CPUStats struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	Iowait  uint64
	Irq     uint64
	Softirq uint64
}

var (
	lastCPUStats CPUStats
	firstCall    = true
	cpuStatsMu   sync.Mutex
)

// GetCPUUsage 获取Linux系统的CPU使用率
func GetCPUUsage() (float64, error) {
	cpuStatsMu.Lock()
	defer cpuStatsMu.Unlock()

	stats, err := readCPUStats()
	if err != nil {
		return 0, err
	}

	if firstCall {
		// 第一次调用，初始化统计信息并间隔 100ms 后重新采样
		lastCPUStats = *stats
		firstCall = false
		time.Sleep(100 * time.Millisecond)
		stats, err = readCPUStats()
		if err != nil {
			return 0, err
		}
	}

	// 计算差值
	userDiff := stats.User - lastCPUStats.User
	niceDiff := stats.Nice - lastCPUStats.Nice
	systemDiff := stats.System - lastCPUStats.System
	idleDiff := stats.Idle - lastCPUStats.Idle
	iowaitDiff := stats.Iowait - lastCPUStats.Iowait
	irqDiff := stats.Irq - lastCPUStats.Irq
	softirqDiff := stats.Softirq - lastCPUStats.Softirq

	// 更新上次统计
	lastCPUStats = *stats

	// 计算总时间和空闲时间
	total := userDiff + niceDiff + systemDiff + idleDiff + iowaitDiff + irqDiff + softirqDiff
	idle := idleDiff + iowaitDiff

	if total == 0 {
		return 0, fmt.Errorf("CPU时间差值计算为0，无法获取使用率")
	}

	// CPU使用率 = (总时间 - 空闲时间) / 总时间 * 100
	usage := float64(total-idle) / float64(total) * 100

	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}

// readCPUStats 读取 /proc/stat 中的CPU统计信息
func readCPUStats() (*CPUStats, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("无法打开/proc/stat文件: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return nil, fmt.Errorf("invalid cpu stats format")
			}

			stats := &CPUStats{}
			stats.User, _ = strconv.ParseUint(fields[1], 10, 64)
			stats.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
			stats.System, _ = strconv.ParseUint(fields[3], 10, 64)
			stats.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
			stats.Iowait, _ = strconv.ParseUint(fields[5], 10, 64)
			stats.Irq, _ = strconv.ParseUint(fields[6], 10, 64)
			stats.Softirq, _ = strconv.ParseUint(fields[7], 10, 64)

			return stats, nil
		}
	}

	return nil, fmt.Errorf("cpu stats not found")
}
