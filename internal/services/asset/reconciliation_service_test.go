package asset

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ListExceptions silence 默认过滤测试(D-R3-A1-01 / SC 7)
//
// 完整 ListExceptions JOIN 子句含 PG 特定 ::uuid 转换,SQLite 跑不通。
// 用静态源码扫描模式验证 silence 过滤逻辑存在(同 statistics 静态断言模式),
// 同时验证 ExceptionListParams.ShowSilenced 字段已加。
//
// PG 集成测试由 44-VALIDATION.md manual-only 验收(运维在 dev DB 跑)。
// ============================================================================

func TestListExceptionsSilenceFilter(t *testing.T) {
	src, err := os.ReadFile("reconciliation_service.go")
	require.NoError(t, err, "must read reconciliation_service.go")
	content := string(src)

	// D-R3-A1-01 — silence 默认过滤 SQL 必须存在
	assert.Contains(t, content,
		"NOT ('silence' = ANY(sys_data_reconciliation.applied_actions))",
		"ListExceptions 必须含 silence 默认过滤 SQL(WHERE NOT ('silence' = ANY(applied_actions)))")
	// 字段必须存在
	assert.Contains(t, content, "ShowSilenced bool",
		"ExceptionListParams 必须含 ShowSilenced 字段")
	// 条件分支必须存在(默认 false 时过滤)
	assert.Contains(t, content, "if !params.ShowSilenced",
		"silence 过滤必须在 !ShowSilenced 分支下添加")
	// 必须用全限定列名(避免 JOIN 歧义)
	assert.Contains(t, content, "sys_data_reconciliation.applied_actions",
		"silence 过滤必须用全限定列名 sys_data_reconciliation.applied_actions")
}

// TestListExceptionsShowSilencedBranches 验证 main + fallback 两个路径都加了过滤
//
// 防止 fallback 路径漏加(Pitfall 7 — silence 过滤只在 main 加,fallback 漏加
// 会让 MV 缺失场景的运维看到 silence 记录)。
func TestListExceptionsShowSilencedBranches(t *testing.T) {
	src, err := os.ReadFile("reconciliation_service.go")
	require.NoError(t, err)
	content := string(src)

	// 统计 silence 过滤出现次数(main + fallback 共 2 次)
	count := 0
	for i := 0; i+len("NOT ('silence' = ANY") <= len(content); i++ {
		if content[i:i+len("NOT ('silence' = ANY")] == "NOT ('silence' = ANY" {
			count++
		}
	}
	assert.GreaterOrEqual(t, count, 2,
		"D-R3-A1-01: silence 过滤必须同时存在 main + fallback 路径(共 2 处,防 Pitfall 7)")
}
