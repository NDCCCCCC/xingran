package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildDefaultPreferences_HardcodedDefaults 是 v1.22 D-10 cleanup 后的
// 回归保护测试：buildDefaultPreferences() 已不再合并 sys.theme.default
// （该特性随默认主题页面被一并删除），改为直接返回硬编码默认值。
//
// 保护目的：任何对默认值的无意识改动（字段顺序、值变化、字段删除）都必须
// 有意识地修改本测试。CI backend job 跑 `go test ./internal/...` 自动覆盖；
// filter `-run TestBuildDefaultPreferences` 也能在 <100ms 内快速验证。
//
// 失败即视为 bug：新登录用户会突然看到 light→dark、SidebarWidth 跳变等
// 行为差异，没有任何配置入口能解释。
func TestBuildDefaultPreferences_HardcodedDefaults(t *testing.T) {
	// buildDefaultPreferences() 不访问 s.db，nil db 字段安全；省掉 sqlite
	// 依赖和 setup 成本，让这个测试只验证纯函数行为。
	svc := &settingsService{}
	prefs := svc.buildDefaultPreferences()

	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
	assert.Equal(t, "classic", prefs.LayoutType)
	assert.Equal(t, "comfortable", prefs.LayoutDensity)
	assert.Equal(t, 280, prefs.SidebarWidth)
	assert.Equal(t, 64, prefs.SidebarCollapsedWidth)
	assert.Equal(t, false, prefs.SidebarCollapsed)
	assert.Equal(t, 10, prefs.PageSize)
	assert.Equal(t, "zh-CN", prefs.Language)
}
