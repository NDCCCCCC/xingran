package operlog

// DEBT-04（v1.51）：operlog.exclude_paths 配置驱动白名单 —— 缺省行为补充验证。
//
// 功能主体（Configure/IsExcludedPath + Record 入口早退 + filepath.Match 匹配
// 语义 + Record-skips 端到端）已由 Phase 35 OPERLOG-01..05 落地，覆盖见
// TestIsExcludedPath 与 TestExcludedPathsEarlyReturn。本文件只补任务验收点
// 「配置缺省行为」：Configure(nil)（生产未配置白名单）时任何路径都不被排除，
// 行为与白名单特性引入之前完全一致。

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsExcludedPath_DefaultNoConfig 未配置（Configure(nil)）时任何路径都不排除。
func TestIsExcludedPath_DefaultNoConfig(t *testing.T) {
	defer Configure(nil) // 恢复缺省，避免污染其它用例
	Configure(nil)

	for _, path := range []string{
		"/api/v1/system/user/list",
		"/api/v1/rpa/workers/w-1/heartbeat",
		"/healthz",
		"/swagger/index.html",
	} {
		assert.False(t, IsExcludedPath(path), "未配置白名单时 %q 不应被排除", path)
	}
}

// TestIsExcludedPath_ExclusiveReconfigure 每次.Configure 整体替换（非追加）——
// 运维收缩白名单后旧模式不应继续命中。
func TestIsExcludedPath_ExclusiveReconfigure(t *testing.T) {
	defer Configure(nil)
	Configure([]string{"/healthz"})
	assert.True(t, IsExcludedPath("/healthz"))

	Configure([]string{"/metrics"}) // 整体替换
	assert.False(t, IsExcludedPath("/healthz"), "重配置后旧条目应失效（replace 语义）")
	assert.True(t, IsExcludedPath("/metrics"))
}
