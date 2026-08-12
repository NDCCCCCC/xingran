package system

import (
	"fmt"
	"runtime"
	"time"
)

// SystemMetrics 系统指标结构
type SystemMetrics struct {
	CPUUsage    float64   `json:"cpuUsage"`    // CPU使用率 (0-100)
	MemoryUsage float64   `json:"memoryUsage"` // 内存使用率 (0-100)
	DiskUsage   float64   `json:"diskUsage"`   // 磁盘使用率 (0-100)
	NetworkRx   uint64    `json:"networkRx"`   // 网络接收字节数
	NetworkTx   uint64    `json:"networkTx"`   // 网络发送字节数
	ProcessNum  int       `json:"processNum"`  // 进程/协程数量
	TotalMemory uint64    `json:"totalMemory"` // 总内存字节数
	UsedMemory  uint64    `json:"usedMemory"`  // 已用内存字节数
	Timestamp   time.Time `json:"timestamp"`   // 时间戳
}

// GetSystemMetrics 获取真实系统指标
func GetSystemMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// 获取内存信息（真实系统内存，不是Go进程内存）
	totalMemory, usedMemory, err := getSystemMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("无法获取系统内存信息: %w", err)
	}

	// 验证获取的数据有效性
	if totalMemory == 0 {
		return nil, fmt.Errorf("获取到的系统总内存为0")
	}
	if usedMemory > totalMemory {
		return nil, fmt.Errorf("已用内存(%d)超过总内存(%d)", usedMemory, totalMemory)
	}

	metrics.TotalMemory = totalMemory
	metrics.UsedMemory = usedMemory
	// 计算真实内存使用率
	metrics.MemoryUsage = float64(metrics.UsedMemory) / float64(metrics.TotalMemory) * 100

	// 获取CPU使用率（真实数据）
	cpuUsage, err := GetCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("无法获取CPU使用率: %w", err)
	}
	metrics.CPUUsage = cpuUsage

	// 获取磁盘使用率（真实数据）
	// 根据操作系统选择正确的磁盘路径
	diskPath := "/"
	if runtime.GOOS == "windows" {
		diskPath = "C:" // Windows系统盘
	}
	diskUsage, err := GetDiskUsage(diskPath)
	if err != nil {
		return nil, fmt.Errorf("无法获取磁盘使用率: %w", err)
	}
	metrics.DiskUsage = diskUsage

	// 获取网络统计（真实数据）
	networkRx, networkTx, err := GetNetworkStats()
	if err != nil {
		return nil, fmt.Errorf("无法获取网络统计数据: %w", err)
	}
	metrics.NetworkRx = networkRx
	metrics.NetworkTx = networkTx

	// 获取真实进程数量（不是Go协程数）
	processCount, err := GetProcessCount()
	if err != nil {
		return nil, fmt.Errorf("无法获取进程数量: %w", err)
	}
	metrics.ProcessNum = processCount

	return metrics, nil
}
