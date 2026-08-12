//go:build windows
// +build windows

package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getSystemMemoryInfo 获取Windows系统的内存信息
func getSystemMemoryInfo() (uint64, uint64, error) {
	// 首先尝试使用PowerShell获取更可靠的内存信息
	if total, used, err := getMemoryViaPowerShell(); err == nil {
		return total, used, nil
	}

	// 如果PowerShell失败，回退到WMIC
	return getMemoryViaWMIC()
}

// getMemoryViaPowerShell 使用PowerShell获取内存信息
func getMemoryViaPowerShell() (uint64, uint64, error) {
	// 使用PowerShell获取计算机系统内存信息
	cmd := exec.Command("powershell", "-Command",
		"Get-CimInstance -ClassName Win32_ComputerSystem | Select-Object @{Name='TotalMemory'; Expression={$_.TotalPhysicalMemory}} | ConvertTo-Json")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("执行PowerShell获取总内存命令失败: %w", err)
	}

	// 解析总内存
	var totalMemory struct {
		TotalMemory uint64 `json:"TotalMemory"`
	}

	// 简单的JSON解析（避免引入json包依赖）
	totalStr := strings.TrimSpace(string(output))
	if !strings.Contains(totalStr, "TotalMemory") {
		return 0, 0, fmt.Errorf("PowerShell输出格式异常: %s", totalStr)
	}

	// 提取数字
	lines := strings.Split(totalStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "TotalMemory") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				numStr := strings.TrimSpace(strings.Replace(parts[1], ",", "", -1))
				numStr = strings.Trim(numStr, `" `)
				if totalBytes, err := strconv.ParseUint(numStr, 10, 64); err == nil {
					totalMemory.TotalMemory = totalBytes
					break
				}
			}
		}
	}

	if totalMemory.TotalMemory == 0 {
		return 0, 0, fmt.Errorf("无法解析PowerShell总内存输出")
	}

	// 获取可用内存
	cmd = exec.Command("powershell", "-Command",
		"(Get-CimInstance -ClassName Win32_OperatingSystem).FreePhysicalMemory | ConvertTo-Json")
	output, err = cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("执行PowerShell获取可用内存命令失败: %w", err)
	}

	// 解析可用内存（单位是KB）
	freeKBStr := strings.TrimSpace(string(output))
	freeKB, err := strconv.ParseUint(freeKBStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("无法解析PowerShell可用内存输出: %w", err)
	}

	// 转换为字节
	freeBytes := freeKB * 1024
	totalBytes := totalMemory.TotalMemory

	// 安全检查
	if totalBytes < 1024*1024*1024 || totalBytes > 1024*1024*1024*1024 {
		return 0, 0, fmt.Errorf("PowerShell返回的总内存值(%d)不合理，应在1GB-1TB范围内", totalBytes)
	}

	if freeBytes > totalBytes {
		return 0, 0, fmt.Errorf("PowerShell返回的可用内存(%d)超过总内存(%d)，数据异常", freeBytes, totalBytes)
	}

	usedBytes := totalBytes - freeBytes
	return totalBytes, usedBytes, nil
}

// getMemoryViaWMIC 使用WMIC获取内存信息（备用方案）
func getMemoryViaWMIC() (uint64, uint64, error) {
	// 使用wmic命令获取内存信息
	cmd := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("执行wmic内存命令失败: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("wmic输出格式异常")
	}

	// 跳过标题行，解析数据行
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "TotalVisibleMemorySize,FreePhysicalMemory") {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) >= 3 {
			// 解析总内存（KB）和可用内存（KB）
			totalKB, err1 := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64)
			freeKB, err2 := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64)

			if err1 == nil && err2 == nil {
				// 转换为字节
				totalBytes := totalKB * 1024
				freeBytes := freeKB * 1024

				// 多重安全检查
				// 1. 检查总内存是否合理（至少1GB，不超过1TB）
				if totalBytes < 1024*1024*1024 || totalBytes > 1024*1024*1024*1024 {
					return 0, 0, fmt.Errorf("WMIC返回的总内存值(%d)不合理，应在1GB-1TB范围内", totalBytes)
				}

				// 2. 检查可用内存不超过总内存
				if freeBytes > totalBytes {
					return 0, 0, fmt.Errorf("WMIC返回的可用内存(%d)超过总内存(%d)，数据异常", freeBytes, totalBytes)
				}

				// 3. 检查可用内存比例是否合理（不超过99%）
				if freeBytes > totalBytes*99/100 {
					return 0, 0, fmt.Errorf("WMIC返回的可用内存比例(%d/%d)过高，数据异常", freeBytes, totalBytes)
				}

				usedBytes := totalBytes - freeBytes

				// 4. 再次验证已用内存不超过总内存（冗余检查）
				if usedBytes > totalBytes {
					return 0, 0, fmt.Errorf("计算出的已用内存(%d)超过总内存(%d)，计算异常", usedBytes, totalBytes)
				}

				return totalBytes, usedBytes, nil
			}
		}
	}

	// 如果解析失败，返回错误而不是默认值
	return 0, 0, fmt.Errorf("无法解析wmic内存命令输出")
}


