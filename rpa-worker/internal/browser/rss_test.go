package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Phase 154 / ENGINE-01：RSS 增长基线（N=1 vs N=8）
//
// 断言口径：Chrome 进程树 RSS 在 8 个 page 时的总量必须远小于「8 个独立
// Chrome」——每多一个 page 的增量 < 60MB（实测约 15–40MB），总量 < 2GB。
// 实测数字用 t.Logf 输出，SUMMARY 引用同一次运行的数。
// ============================================================================

// measureTreeRSSBytes 测量以 rootPID 为根的进程树 RSS。
func measureTreeRSSBytes(rootPID int) (int64, error) {
	if rootPID <= 0 {
		return 0, fmt.Errorf("chrome pid 未就绪")
	}
	switch runtime.GOOS {
	case "linux":
		return measureLinuxTreeRSS(rootPID)
	case "windows":
		return measureWindowsTreeRSS(rootPID)
	default:
		return 0, fmt.Errorf("当前操作系统不支持 RSS 采样")
	}
}

// measureLinuxTreeRSS 读 /proc 计算进程树 RSS。
func measureLinuxTreeRSS(rootPID int) (int64, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	type procInfo struct {
		ppid int
		rss  int64
	}
	infos := map[int]procInfo{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		// stat 的 comm 字段可能含空格，先剥掉括号内容再切分。
		s := string(data)
		if i, j := strings.Index(s, "("), strings.LastIndex(s, ")"); i >= 0 && j > i {
			s = s[:i] + "x" + s[j+1:]
		}
		fields := strings.Fields(s)
		if len(fields) < 24 {
			continue
		}
		ppid, _ := strconv.Atoi(fields[3])
		rssPages, _ := strconv.ParseInt(fields[23], 10, 64)
		infos[pid] = procInfo{ppid: ppid, rss: rssPages * int64(os.Getpagesize())}
	}
	tree := map[int]bool{rootPID: true}
	for changed := true; changed; {
		changed = false
		for pid, info := range infos {
			if !tree[pid] && tree[info.ppid] {
				tree[pid] = true
				changed = true
			}
		}
	}
	var total int64
	for pid := range tree {
		total += infos[pid].rss
	}
	return total, nil
}

// measureWindowsTreeRSS 用 PowerShell CIM 求和进程树 WorkingSetSize。
func measureWindowsTreeRSS(rootPID int) (int64, error) {
	script := fmt.Sprintf(`
$root = %d
$procs = Get-CimInstance Win32_Process | Select-Object ProcessId,ParentProcessId,WorkingSetSize
$ids = @($root)
for ($i = 0; $i -lt 8; $i++) {
  $children = $procs | Where-Object { $ids -contains $_.ParentProcessId } | Select-Object -ExpandProperty ProcessId
  if (-not $children) { break }
  $ids += $children
}
$total = 0
foreach ($p in $procs) { if ($ids -contains $p.ProcessId) { $total += $p.WorkingSetSize } }
Write-Output $total
`, rootPID)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return 0, fmt.Errorf("powershell 采样失败: %w", err)
	}
	val := strings.TrimSpace(string(out))
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析 RSS 数值失败 %q: %w", val, err)
	}
	return n, nil
}

// TestManager_RSSBaseline ENGINE-01 验收：N=1 与 N=8 的 RSS 增长。
func TestManager_RSSBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过真 Chromium RSS 基线（-short）")
	}
	m := newTestManager(t, 8)
	ctx := context.Background()

	// 触发 Chrome 启动并拿 PID。
	warm, err := m.Acquire(ctx)
	if err != nil {
		t.Fatalf("首次 Acquire（启动 Chrome）失败: %v", err)
	}
	m.Release(warm)
	for i := 0; i < 50 && m.PID() == 0; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	pid := m.PID()
	if pid == 0 {
		t.Fatal("拿不到 Chrome PID")
	}

	measureWith := func(n int) (int64, error) {
		pages := make([]Page, 0, n)
		for i := 0; i < n; i++ {
			p, err := m.Acquire(ctx)
			if err != nil {
				t.Fatalf("Acquire #%d: %v", i+1, err)
			}
			pages = append(pages, p)
		}
		// 让 Chrome 完成渲染与内存分配（多个 page 有生命周期抖动）。
		time.Sleep(2 * time.Second)
		rss, err := measureTreeRSSBytes(pid)
		for _, p := range pages {
			m.Release(p)
		}
		// 回收后等待 GC 与 page 销毁。
		time.Sleep(1500 * time.Millisecond)
		return rss, err
	}

	idle, err := measureTreeRSSBytes(pid)
	if err != nil {
		t.Skipf("本机无法测量进程树 RSS（%v）——需在 Linux/Windows 实测环境补测", err)
	}
	rss1, err := measureWith(1)
	if err != nil {
		t.Skipf("RSS 采样失败: %v", err)
	}
	rss8, err := measureWith(8)
	if err != nil {
		t.Skipf("RSS 采样失败: %v", err)
	}

	perPage := float64(rss8-rss1) / 7.0
	t.Logf("RSS 基线（%s，root pid=%d）: 空载=%.1fMB | N=1=%.1fMB | N=8=%.1fMB | 每 page 增量≈%.1fMB",
		testChromePath(), pid,
		float64(idle)/(1<<20), float64(rss1)/(1<<20), float64(rss8)/(1<<20), perPage/(1<<20))

	if rss8 > 2<<30 {
		t.Fatalf("N=8 page 的 Chrome 进程树 RSS=%.1fMB 超过 2GB，承载模型异常",
			float64(rss8)/(1<<20))
	}
	if perPage > 60<<20 {
		t.Fatalf("每 page 增量 %.1fMB > 60MB，疑似退化为「每 page 一个 Chrome」",
			perPage/(1<<20))
	}
}
