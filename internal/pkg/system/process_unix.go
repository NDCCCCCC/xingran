//go:build !windows
// +build !windows

package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetProcessCount 获取真实系统进程数量
func GetProcessCount() (int, error) {
	// 使用ps命令获取进程数量
	cmd := exec.Command("ps", "-eo", "pid")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("执行ps命令失败: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	processCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "PID") {
			continue
		}

		processCount++
	}

	if processCount == 0 {
		return 0, fmt.Errorf("未获取到任何进程信息")
	}

	return processCount, nil
}
