package asset

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// task-3/4 权限接线新增的 Setup*WriteRouter / Setup*ExportRouter 此前无测试
// 触达，diff 覆盖率缺口。模式与 internal/api/v1/system/setup_routers_full_test.go
// 相同：内存 sqlite core + gin engine，真实调用 Setup 注册路由，
// 断言关键路由已挂到 engine.Routes()。复用同包 newTestCore fixture。

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func routeSet(t *testing.T, engine *gin.Engine) map[string]bool {
	t.Helper()
	set := make(map[string]bool)
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	return set
}

// TestSetupFixSuggestionWriteRouter_RegistersActionRoutes 修复建议 4 个写动作
// 由 strict 兄弟组逐动作接线（task-3 G2）。
func TestSetupFixSuggestionWriteRouter_RegistersActionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newTestCore(t)
	engine := gin.New()
	root := engine.Group("/asset/reconciliation")

	groups := FixSuggestionWriteRouteGroups{
		Accept:   root.Group(""),
		Reject:   root.Group(""),
		Apply:    root.Group(""),
		Rollback: root.Group(""),
	}
	require.NotPanics(t, func() {
		SetupFixSuggestionWriteRouter(groups, c)
	})

	routes := routeSet(t, engine)
	for _, want := range []string{
		http.MethodPost + " /asset/reconciliation/fix-suggestion/:id/accept",
		http.MethodPost + " /asset/reconciliation/fix-suggestion/:id/reject",
		http.MethodPost + " /asset/reconciliation/fix-suggestion/:id/apply",
		http.MethodPost + " /asset/reconciliation/fix-suggestion/:id/rollback",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupReconciliationWriteRouter_RegistersWriteRoutes 对账写端点
// resolve / refresh（task-3 G2 拆分）。
func TestSetupReconciliationWriteRouter_RegistersWriteRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newTestCore(t)
	engine := gin.New()
	rg := engine.Group("/asset/reconciliation")

	require.NotPanics(t, func() {
		SetupReconciliationWriteRouter(rg, c)
	})

	routes := routeSet(t, engine)
	assert.True(t, routes[http.MethodPost+" /asset/reconciliation/exception/:id/resolve"])
	assert.True(t, routes[http.MethodPost+" /asset/reconciliation/refresh"])
}

// TestSetupReconciliationExceptionExportRouter_RegistersExportRoutes
// 例外规则导出/模板下载 strict 兄弟组（task-3 G2）。
func TestSetupReconciliationExceptionExportRouter_RegistersExportRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newTestCore(t)
	engine := gin.New()
	rg := engine.Group("/asset/reconciliation")

	require.NotPanics(t, func() {
		SetupReconciliationExceptionExportRouter(rg, c)
	})

	routes := routeSet(t, engine)
	assert.True(t, routes[http.MethodPost+" /asset/reconciliation/exception-rule/export"])
	assert.True(t, routes[http.MethodPost+" /asset/reconciliation/exception-rule/template"])
}
