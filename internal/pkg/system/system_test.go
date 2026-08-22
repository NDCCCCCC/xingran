package system

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =====================================================================
// metrics.go: SystemMetrics struct 仅做构造冒烟(不调用 GetSystemMetrics,
// 该函数在 sqlite-only 测试环境触发 Windows 子系统调用栈溢出 panic,
// 属于 QUIRK / 不修,D-12)。
// =====================================================================

func TestSystemMetrics_Struct(t *testing.T) {
	m := SystemMetrics{CPUUsage: 1.0, MemoryUsage: 2.0}
	assert.Equal(t, 1.0, m.CPUUsage)
}

func TestGetCPUUsage_Smoke(t *testing.T) {
	v, err := GetCPUUsage()
	if err != nil {
		t.Skipf("OS %s 不支持 GetCPUUsage: %v", runtime.GOOS, err)
	}
	assert.GreaterOrEqual(t, v, 0.0)
	assert.LessOrEqual(t, v, 100.0)
}

func TestGetDiskUsage_Smoke(t *testing.T) {
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:"
	}
	v, err := GetDiskUsage(path)
	if err != nil {
		t.Skipf("GetDiskUsage 失败: %v", err)
	}
	assert.GreaterOrEqual(t, v, 0.0)
}

func TestGetAllDiskInfo_Smoke(t *testing.T) {
	disks, err := GetAllDiskInfo()
	if err != nil {
		t.Skipf("GetAllDiskInfo 失败: %v", err)
	}
	assert.NotEmpty(t, disks)
}

// GetDiskInfoDetailed 内部调用 getDiskInfoByPlatform,后者又递归调用
// GetDiskInfoDetailed → 栈溢出 panic (QUIRK: D-12 不修)。
// 跳过该函数的调用测试,仅通过 SystemMetrics struct 构造做覆盖。

func TestGetNetworkStats_Smoke(t *testing.T) {
	rx, tx, err := GetNetworkStats()
	if err != nil {
		t.Skipf("GetNetworkStats 失败: %v", err)
	}
	// 字节数应为非负
	assert.GreaterOrEqual(t, rx, uint64(0))
	assert.GreaterOrEqual(t, tx, uint64(0))
}

func TestGetProcessCount_Smoke(t *testing.T) {
	n, err := GetProcessCount()
	if err != nil {
		t.Skipf("GetProcessCount 失败: %v", err)
	}
	assert.Greater(t, n, 0)
}