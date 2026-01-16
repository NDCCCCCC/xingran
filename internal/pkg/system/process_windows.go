//go:build windows
// +build windows

package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetProcessCount 获取真实系统进程数量
func GetProcessCount() (int, error) {
	// 使用wmic命令获取进程数量
	cmd := exec.Command("wmic", "process", "get", "ProcessId", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("执行wmic命令失败: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	processCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "ProcessId") || strings.Contains(line, "Node,") {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) >= 2 {
			processID := strings.Trim(fields[1], `"`)
			if processID != "" {
				processCount++
			}
		}
	}

	if processCount == 0 {
		return 0, fmt.Errorf("未获取到任何进程信息")
	}

	return processCount, nil
}
