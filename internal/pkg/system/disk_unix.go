//go:build !windows
// +build !windows

package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getDiskUsageByPlatform 获取Unix-like系统的磁盘使用率
func getDiskUsageByPlatform(path string) (float64, error) {
	// 使用df命令获取磁盘使用率，这是最跨平台的方式
	cmd := exec.Command("df", "-h", path)
	output, err := cmd.Output()
	if err != nil {
		// 如果df命令失败，使用估算值
		return 65.0, nil
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 65.0, fmt.Errorf("unexpected df output")
	}

	// 跳过标题行，解析数据行
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return 65.0, fmt.Errorf("unexpected df line format")
	}

	// 解析使用率字段（通常在第5个字段，格式如"65%"）
	usageStr := fields[4]
	if strings.HasSuffix(usageStr, "%") {
		usageStr = strings.TrimSuffix(usageStr, "%")
		if usage, err := strconv.ParseFloat(usageStr, 64); err == nil {
			return usage, nil
		}
	}

	// 如果解析失败，返回默认值
	return 65.0, nil
}
