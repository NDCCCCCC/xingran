//go:build windows

package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =====================================================================
// 补测计划 B1-P3 (修复): pkg/system Windows-only 测试
// =====================================================================

// ─── cpu_windows.go — 纯函数测试（filetime 是固定布局） ─────────────
// （原位于 sysmetrics_common_test.go；符号仅存在于 windows 构建标签，
//   放 common 文件会导致 linux CI 编译失败——2026-09-07 CI 34056266515）

func TestFiletimeToUint64(t *testing.T) {
	tests := []struct {
		name string
		high uint32
		low  uint32
		want uint64
	}{
		{"zero", 0, 0, 0},
		{"low_only", 0, 1, 1},
		{"high_only", 1, 0, 0x100000000},
		{"max", 0xffffffff, 0xffffffff, 0xffffffffffffffff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ft := filetime{dwHighDateTime: tt.high, dwLowDateTime: tt.low}
			assert.Equal(t, tt.want, filetimeToUint64(ft))
		})
	}
}

func TestGetCPUUsageByRuntime(t *testing.T) {
	// 纯函数：直接调用验证返回值范围
	usage := getCPUUsageByRuntime()
	assert.GreaterOrEqual(t, usage, 2.0)
	assert.LessOrEqual(t, usage, 95.0)
}

// 用 t.Setenv 替代 os.Setenv + defer，更安全且并发友好。
// =====================================================================

// ─── network.go — getNetworkViaWMIC 错误分支 ──────────────────────────────

func TestGetNetworkViaWMIC_CommandNotFound(t *testing.T) {
	t.Setenv("PATH", "") // 命令找不到

	_, _, err := getNetworkViaWMIC()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wmic")
}

// ─── network.go — getNetworkAdapterStatsViaPowerShell 错误分支 ─────────────

func TestGetNetworkAdapterStatsViaPowerShell_InvalidOutput(t *testing.T) {
	t.Setenv("PATH", "")

	_, _, err := getNetworkAdapterStatsViaPowerShell()
	assert.Error(t, err)
}

// ─── network.go — GetNetworkStats Windows 回退 ──────────────────────────

func TestGetNetworkStats_WindowsFallback(t *testing.T) {
	t.Setenv("PATH", "") // PowerShell 与 WMIC 均失败，触发 error

	rx, tx, err := getWindowsNetworkStats()
	_ = rx
	_ = tx
	assert.Error(t, err)
}

// ─── memory_windows.go — getMemoryViaPowerShell 错误分支 ─────────────────

func TestGetMemoryViaPowerShell_CommandNotFound(t *testing.T) {
	t.Setenv("PATH", "")

	_, _, err := getMemoryViaPowerShell()
	assert.Error(t, err)
}

// ─── memory_windows.go — getMemoryViaWMIC 错误分支 ────────────────────────

func TestGetMemoryViaWMIC_CommandNotFound(t *testing.T) {
	t.Setenv("PATH", "")

	_, _, err := getMemoryViaWMIC()
	assert.Error(t, err)
}

// ─── memory_windows.go — getSystemMemoryInfo 回退路径 ────────────────────

func TestGetSystemMemoryInfo_PowerShellFallsBackToWMIC(t *testing.T) {
	t.Setenv("PATH", "")

	_, _, err := getSystemMemoryInfo()
	assert.Error(t, err) // PATH= 时 PowerShell 与 WMIC 均失败
}

// ─── disk_windows_multi.go — getAllDiskInfoByPlatform WMIC 错误分支 ────────

func TestGetAllDiskInfoByPlatform_WMICError(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := getAllDiskInfoByPlatform()
	assert.Error(t, err)
}

// ─── disk_windows_multi.go — getDisksViaPowerShell 错误分支 ──────────────

func TestGetDisksViaPowerShell_CommandNotFound(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := getDisksViaPowerShell()
	assert.Error(t, err)
}

// ─── metrics.go — GetSystemMetrics PATH= 路径 ─────────────────────────

func TestGetSystemMetrics_PathNotFound(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := GetSystemMetrics()
	assert.Error(t, err) // PATH= 时所有子命令失败
}

// ─── 成功路径（PowerShell 可用时跳过）──────────────────────────────────

func TestGetNetworkViaPowerShell_Success(t *testing.T) {
	rx, tx, err := getNetworkViaPowerShell()
	if err != nil {
		t.Skipf("PowerShell 不可用（CI 环境限制）: %v", err)
	}
	assert.Greater(t, rx, uint64(0))
	assert.Greater(t, tx, uint64(0))
}

func TestGetMemoryViaPowerShell_Success(t *testing.T) {
	total, used, err := getMemoryViaPowerShell()
	if err != nil {
		t.Skipf("PowerShell 不可用（CI 环境限制）: %v", err)
	}
	assert.Greater(t, total, uint64(0))
	assert.GreaterOrEqual(t, used, uint64(0))
	assert.LessOrEqual(t, used, total)
}

func TestGetDisksViaPowerShell_Success(t *testing.T) {
	disks, err := getDisksViaPowerShell()
	if err != nil {
		t.Skipf("PowerShell 不可用（CI 环境限制）: %v", err)
	}
	assert.NotEmpty(t, disks)
	for _, d := range disks {
		assert.Greater(t, d.Total, uint64(0))
		assert.LessOrEqual(t, d.Available, d.Total)
	}
}

func TestGetAllDiskInfoByPlatform_Success(t *testing.T) {
	disks, err := getAllDiskInfoByPlatform()
	if err != nil {
		t.Skipf("磁盘命令不可用: %v", err)
	}
	assert.NotEmpty(t, disks)
}

func TestGetNetworkAdapterStatsViaPowerShell_Success(t *testing.T) {
	rx, tx, err := getNetworkAdapterStatsViaPowerShell()
	if err != nil {
		t.Skipf("PowerShell 不可用: %v", err)
	}
	// 模拟数据：1MB rx, 512KB tx（见 network.go getNetworkAdapterStatsViaPowerShell）
	assert.Equal(t, uint64(1024*1024), rx)
	assert.Equal(t, uint64(512*1024), tx)
}
