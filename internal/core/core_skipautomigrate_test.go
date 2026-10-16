// Package core — SKIP_AUTOMIGRATE 生产模式 fatal 守卫的源码断言测试 (CDX-H2)。
//
// 背景(codex HIGH): SKIP_AUTOMIGRATE=true 跳过范围超预期 —— 除 AutoMigrate 外还跳过
// Migrate175/176/202-205,BootstrapMissingTables 仅保证 sys_api_keys +
// sys_api_key_usage_logs 两表。新库 + SKIP_AUTOMIGRATE=true 会得到半初始化系统
// (无 reconciliation 视图、无端口审计索引、无 dot1x_user_limit 列等)。
// 该开关是 Supabase pooler 的 dev 应急旁路,生产误设必须 fail-fast。
package core

import (
	"os"
	"strings"
	"testing"
)

// TestSkipAutomigrateReleaseModeFatalGuard 源码断言:initDBAndData 的 SKIP_AUTOMIGRATE
// 分支在最前面(Warnf 之前)存在 server.mode=release 判定,命中时 return fmt.Errorf
// 终止启动,错误文案指明"半初始化"风险与处置(移除环境变量)。
func TestSkipAutomigrateReleaseModeFatalGuard(t *testing.T) {
	src, err := os.ReadFile("core.go")
	if err != nil {
		t.Fatalf("read core.go: %v", err)
	}
	s := string(src)

	needle := `os.Getenv("SKIP_AUTOMIGRATE") == "true"`
	idx := strings.Index(s, needle)
	if idx < 0 {
		t.Fatalf("SKIP_AUTOMIGRATE branch not found in core.go initDBAndData")
	}
	// 取分支体窗口(分支体 + 后续 1200 字符足够覆盖守卫位置)
	window := s[idx:min(idx+1200, len(s))]

	// 1) 生产模式判定必须存在
	if !strings.Contains(window, `c.Config.Server.Mode == "release"`) {
		t.Fatalf("SKIP_AUTOMIGRATE branch must guard with c.Config.Server.Mode == \"release\" (CDX-H2)")
	}
	// 2) release 命中时必须 return 错误(启动终止),错误文案含"半初始化"指明风险
	if !strings.Contains(window, "半初始化") {
		t.Fatalf("SKIP_AUTOMIGRATE release-mode guard error message must mention 半初始化 (CDX-H2), window:\n%s", window)
	}
	// 3) 守卫必须在 WARN 旁路日志之前(先 fatal 再旁路)
	guardIdx := strings.Index(window, `c.Config.Server.Mode == "release"`)
	warnIdx := strings.Index(window, "Warnf")
	if warnIdx >= 0 && guardIdx > warnIdx {
		t.Fatalf("release-mode fatal guard must precede the WARN bypass log (CDX-H2)")
	}
	// 4) 守卫命中必须 return fmt.Errorf 而非仅日志
	if !strings.Contains(window, "return fmt.Errorf") {
		t.Fatalf("SKIP_AUTOMIGRATE release-mode guard must return fmt.Errorf (fail-fast), window:\n%s", window)
	}
}

// TestSkipAutomigrateDebugBypassRetained 源码断言:非 release 模式的 WARN 旁路保留
// (Warnf + BootstrapMissingTables 调用),dev 应急能力不丢。
func TestSkipAutomigrateDebugBypassRetained(t *testing.T) {
	src, err := os.ReadFile("core.go")
	if err != nil {
		t.Fatalf("read core.go: %v", err)
	}
	s := string(src)

	for _, required := range []string{
		"[SKIP_AUTOMIGRATE=true]",       // WARN 旁路日志保留
		"c.DB.BootstrapMissingTables()", // dev 旁路补建调用保留
	} {
		if !strings.Contains(s, required) {
			t.Fatalf("core.go missing required fragment %q (CDX-H2: dev bypass must be retained)", required)
		}
	}
}

// min 辅助函数(Go 1.21+ 有内置 min,但保持与 db 包测试一致的显式实现避免版本歧义)。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
