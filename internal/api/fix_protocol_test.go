package api

// fix_protocol_test.go —— 防「误下线」流程协议文档守卫（PR #85/#86 事故配套）。
//
// 任务书要求 scripts/FIX-PROTOCOL.md 存在且含「先查 service 层」硬检查项。
// 本测试锁定文档存在、三原则、硬检查项与 PR 复盘锚点，防止文档被删或
// 关键检查项被稀释后无人察觉（源码文本扫描，不启动 gin、不连 DB）。

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixProtocolRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "无法定位测试文件路径")
	// internal/api → 仓库根：上溯三级
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// TestFixProtocolDocument_ExistsWithHardChecks 断言协议文档存在且含
// 「先查 service 层」硬检查项与三条原则、PR 复盘锚点。
func TestFixProtocolDocument_ExistsWithHardChecks(t *testing.T) {
	root := fixProtocolRepoRoot(t)
	path := filepath.Join(root, "scripts", "FIX-PROTOCOL.md")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "scripts/FIX-PROTOCOL.md 必须存在")
	doc := string(data)

	for _, must := range []string{
		// 硬检查项（任务书点名要求）
		"先查 service 层",
		// 三条硬性原则
		"下线用户可见功能前必须先查 service 层实现与历史路由",
		"必须由代码证据而非注释判定",
		"必须先与前端调用点交叉验证",
		// 事故复盘锚点，保证上下文不丢失
		"#85",
		"#86",
		"cd306b359",
		// 门禁脚本不被本条改动（任务书要求）
		"check-frontend-api-urls.mjs",
	} {
		assert.Contains(t, doc, must, "FIX-PROTOCOL.md 应包含 %q", must)
	}

	// 明确声明「不改门禁脚本」，避免后人误读成本文件授权改脚本。
	assert.Contains(t, doc, "不改", "文档应声明不改门禁脚本")
}
