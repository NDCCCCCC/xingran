package system

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errMockNotFound is a sentinel error used to simulate file-not-found paths
// in tests that monkey-patch osOpen. Defined here (test-only) to keep the
// production package free of test artefacts.
var errMockNotFound = errors.New("mock: file not found")

// =====================================================================
// 补测计划 B1-P3 (修复): pkg/system 测试拆分 + 用 t.Setenv 替代 defer
// 跨平台纯函数测试（无平台依赖）
// =====================================================================

// ─── cpu_windows.go — 纯函数跨平台可测（filetime 是固定布局） ──────

func TestFiletimeToUint64(t *testing.T) {
	tests := []struct {
		name  string
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

// ─── osOpen monkey-patch 验证（osOpen 在 network.go 声明） ───────────────

func TestOSOpenPatch(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(tmp, []byte("hello"), 0644))

	f, err := osOpen(tmp)
	require.NoError(t, err)
	defer f.Close()

	data := make([]byte, 5)
	n, err := f.Read(data)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data[:n]))
}

// ─── Linux 分支（runtime.GOOS 跳过非 Linux） ─────────────────────────

func TestGetLinuxNetworkStats_ProcNetDevNotExist(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("需要 /proc/net/dev，只在 Linux 可测")
	}
	orig := osOpen
	osOpenCalled := false
	osOpen = func(name string) (*os.File, error) {
		osOpenCalled = true
		if name == "/proc/net/dev" {
			return nil, errMockNotFound
		}
		return orig(name)
	}
	t.Cleanup(func() { osOpen = orig })

	_, _, err := getLinuxNetworkStats()
	assert.True(t, osOpenCalled)
	assert.Error(t, err)
}

// TestGetLinuxNetworkStats_Parsing 测试 /proc/net/dev 解析路径
func TestGetLinuxNetworkStats_Parsing(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("需要 /proc/net/dev，只在 Linux 可测")
	}
	tmpFile := filepath.Join(t.TempDir(), "proc_net_dev")
	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo frame compressed colls carrier
  eth0: 12345678 1000 0 0 0 0 0 50 87654321 500 0 0 0 0 0 20
    lo: 100 1 1 0 0 0 0 0 100 1 0 0 0 0 0 0
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	orig := osOpen
	osOpen = func(name string) (*os.File, error) {
		if name == "/proc/net/dev" {
			return os.Open(tmpFile)
		}
		return orig(name)
	}
	t.Cleanup(func() { osOpen = orig })

	rx, tx, err := getLinuxNetworkStats()
	require.NoError(t, err)
	assert.Equal(t, uint64(12345678), rx)
	assert.Equal(t, uint64(87654321), tx)
}

// ─── Darwin 分支（PATH= 触发 netstat 找不到，runtime.GOOS 跳过非 Darwin） ──

func TestGetDarwinNetworkStats_CommandNotFound(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("只在 Darwin 可触发 netstat 找不到")
	}
	t.Setenv("PATH", "") // 自动在测试结束时还原

	_, _, err := getDarwinNetworkStats()
	assert.Error(t, err)
}

// ─── GetNetworkStats default 分支（未知 OS 时触发）────────────────────

func TestGetNetworkStats_UnsupportedPlatform(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "windows", "darwin":
		// 已知平台走对应分支，default 分支不可达。
		// 通过 Darwin/Windows 错误路径覆盖（见对应 OS 测试文件）。
	default:
		rx, tx, err := GetNetworkStats()
		_ = rx
		_ = tx
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	}
}