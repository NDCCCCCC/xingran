//go:build !windows && !linux
// +build !windows,!linux

package system

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GetCPUUsage 获取其他平台的CPU使用率
func GetCPUUsage() (float64, error) {
	switch runtime.GOOS {
	case "darwin":
		return getDarwinCPUUsage()
	default:
		// 对于不支持的平台，使用系统命令
		return getCPUUsageByCommand()
	}
}

// getDarwinCPUUsage 获取macOS的CPU使用率
func getDarwinCPUUsage() (float64, error) {
	// 使用 iostat 命令获取CPU使用率
	cmd := exec.Command("iostat", "-c", "1", "-w", "1")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 4 {
		return 0, fmt.Errorf("unexpected iostat output")
	}

	// iostat 输出格式的最后一行包含CPU使用率
	cpuLine := lines[len(lines)-2]
	fields := strings.Fields(cpuLine)
	if len(fields) < 4 {
		return 0, fmt.Errorf("unexpected iostat CPU line format")
	}

	// iostat 输出格式：us sy id
	userUsage, _ := strconv.ParseFloat(fields[0], 64)
	sysUsage, _ := strconv.ParseFloat(fields[1], 64)
	idleUsage, _ := strconv.ParseFloat(fields[2], 64)

	// 总使用率 = 用户使用率 + 系统使用率
	totalUsage := userUsage + sysUsage

	return totalUsage, nil
}

// getCPUUsageByCommand 使用系统命令获取CPU使用率
func getCPUUsageByCommand() (float64, error) {
	// 尝试使用 top 命令
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "freebsd", "openbsd", "netbsd":
		cmd = exec.Command("top", "-b", "-d", "1")
	default:
		cmd = exec.Command("top", "-b", "-n", "1")
	}

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// 解析 top 命令输出中的CPU使用率
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "%Cpu(s)") || strings.Contains(line, "CPU:") {
			// 提取CPU使用率百分比
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.HasSuffix(field, "%us") || strings.HasSuffix(field, "%id") {
					valueStr := strings.TrimSuffix(field, "%us")
					valueStr = strings.TrimSuffix(valueStr, "%id")
					if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
						if strings.Contains(field, "us") {
							return value, nil
						}
					}
				}
			}
		}
	}

	// 如果无法解析，返回基于运行时的估算值
	return 0, fmt.Errorf("could not parse CPU usage from system command")
}
