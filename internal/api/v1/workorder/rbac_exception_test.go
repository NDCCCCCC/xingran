package workorder

// rbac_exception_test.go —— 周期工单 4 条 86 新端点 RBAC 例外断言（#87 配套）。
//
// 背景：#87 全域治理口径是「写端点逐动作严格权限」；但 enable/disable/
// generate/logs 4 条端点按任务书豁免、仍挂 ANY-of 读组（RBAC-EXC-01）。
// 本测试做三件事，确保例外**不是静默保留**：
//  1. 4 条端点仍由 SetupPeriodicWorkOrderRouter 注册（不得被再次误下线，
//     防 PR #85 事故复发，见 scripts/FIX-PROTOCOL.md）；
//  2. strict 写组 SetupPeriodicWorkOrderWriteRouter 只含模板主体 CRUD，
//     不含这 4 条动作（锁定「例外仍存在」这一事实，迁移时本测试需同步更新）；
//  3. scripts/RBAC-EXCEPTIONS.md 存在、含 RBAC-EXC-01 / reason / window / review，
//     且本包源码注释引用了文档路径（代码↔文档双向可追溯）。
//
// 与 router_task3_smoke_test.go 共用 ci87WorkOrderCore / ci87RouteSet fixture。

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// periodicExceptionPaths 4 条按任务书豁免、仍挂 ANY-of 读组的动作端点。
var periodicExceptionPaths = []string{
	"/workorder/periodic/templates/:id/enable",
	"/workorder/periodic/templates/:id/disable",
	"/workorder/periodic/templates/:id/generate",
	"/workorder/periodic/templates/:id/logs",
}

// periodicStrictWritePaths 模板主体 CRUD（已迁 strict 写组，作为对照组）。
var periodicStrictWritePaths = []string{
	"/workorder/periodic/templates",
	"/workorder/periodic/templates/:id/update",
	"/workorder/periodic/templates/:id/delete",
}

func rbacExceptionRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "无法定位测试文件路径")
	// 从测试文件所在目录逐级上溯，直到找到 go.mod（仓库根），
	// 避免硬编码「上溯 N 级」在目录层级变动时算错。
	dir := filepath.Dir(thisFile)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("从测试文件上溯未找到 go.mod（仓库根）")
	return ""
}

// TestPeriodicActionRoutes_RegisteredInReadGroup 4 条动作端点仍注册在
// SetupPeriodicWorkOrderRouter（ANY-of 读组）内，防 PR #85 误下线复发。
func TestPeriodicActionRoutes_RegisteredInReadGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := ci87WorkOrderCore(t)
	engine := gin.New()
	root := engine.Group("/workorder/periodic")

	require.NotPanics(t, func() {
		SetupPeriodicWorkOrderRouter(root, c)
	})
	routes := ci87RouteSet(engine)

	for _, p := range periodicExceptionPaths {
		assert.True(t, routes[http.MethodPost+" "+p],
			"周期工单动作端点 %s 不应下线（PR #85 误判事故），须保持注册", p)
	}
	// 读组也含 list/statistics，确认挂载面完整。
	assert.True(t, routes[http.MethodPost+" /workorder/periodic/templates/list"])
	assert.True(t, routes[http.MethodPost+" /workorder/periodic/templates/statistics"])
}

// TestPeriodicException_NotInStrictWriteGroup 锁定例外事实：4 条动作端点
// 不在 SetupPeriodicWorkOrderWriteRouter 的 strict 写组里（若将来清偿
// RBAC-EXC-01 把它们迁入 strict，本测试会提醒同步更新文档与断言）。
func TestPeriodicException_NotInStrictWriteGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := ci87WorkOrderCore(t)
	engine := gin.New()
	root := engine.Group("/workorder/periodic/templates")

	groups := PeriodicWorkOrderWriteRouteGroups{
		Add:    root.Group(""),
		Edit:   root.Group(""),
		Delete: root.Group(""),
	}
	require.NotPanics(t, func() {
		SetupPeriodicWorkOrderWriteRouter(groups, c)
	})
	routes := ci87RouteSet(engine)

	for _, p := range periodicStrictWritePaths {
		assert.True(t, routes[http.MethodPost+" "+p],
			"模板主体 CRUD %s 应保持在 strict 写组", p)
	}
	for _, p := range periodicExceptionPaths {
		assert.False(t, routes[http.MethodPost+" "+p],
			"RBAC-EXC-01：%s 仍为 ANY-of 读组例外，不应出现在 strict 写组；"+
				"若已清偿例外，请同步更新 scripts/RBAC-EXCEPTIONS.md 与本断言", p)
	}
}

// TestPeriodicActionRoutes_RBACExceptionDeclared 文档 + 代码注释双向守卫：
// scripts/RBAC-EXCEPTIONS.md 必须存在且含三要素，源码注释必须引用该文档。
func TestPeriodicActionRoutes_RBACExceptionDeclared(t *testing.T) {
	root := rbacExceptionRepoRoot(t)

	docPath := filepath.Join(root, "scripts", "RBAC-EXCEPTIONS.md")
	data, err := os.ReadFile(docPath)
	require.NoError(t, err, "scripts/RBAC-EXCEPTIONS.md 必须存在（例外不得静默保留）")
	doc := string(data)
	for _, must := range []string{
		"RBAC-EXC-01",
		"reason",
		"window",
		"review",
		"enable",
		"disable",
		"generate",
		"logs",
	} {
		assert.Contains(t, doc, must, "例外文档应包含 %q", must)
	}

	// 代码注释引用文档：workorder_router.go 与 router.go 均须标注，避免只写文档不落代码。
	wrPath := filepath.Join(root, "internal", "api", "v1", "workorder", "workorder_router.go")
	wrSrc, err := os.ReadFile(wrPath)
	require.NoError(t, err)
	assert.Contains(t, string(wrSrc), "scripts/RBAC-EXCEPTIONS.md",
		"workorder_router.go 的 4 条例外端点注释必须引用 RBAC-EXCEPTIONS.md")

	rtPath := filepath.Join(root, "internal", "api", "router.go")
	rtSrc, err := os.ReadFile(rtPath)
	require.NoError(t, err)
	assert.Contains(t, string(rtSrc), "scripts/RBAC-EXCEPTIONS.md",
		"router.go 的 periodic 组注释必须引用 RBAC-EXCEPTIONS.md")

	// 例外端点注册处附近（4 条 POST 上方）确实带 RBAC-EXC-01 标注：
	// 取 SetupPeriodicWorkOrderRouter 函数体做局部校验。
	start := strings.Index(string(wrSrc), "func SetupPeriodicWorkOrderRouter")
	require.GreaterOrEqual(t, start, 0, "找不到 SetupPeriodicWorkOrderRouter")
	fnBody := string(wrSrc)[start:]
	if end := strings.Index(fnBody, "\nfunc "); end > 0 {
		fnBody = fnBody[:end]
	}
	assert.Contains(t, fnBody, "RBAC-EXC-01",
		"4 条例外路由注册处必须带 RBAC-EXC-01 标注")
}
