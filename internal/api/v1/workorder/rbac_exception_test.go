package workorder

// rbac_exception_test.go —— 周期工单 4 条动作端点 RBAC 例外清偿断言（#87 配套）。
//
// 历史背景：#87 全域治理口径是「写端点逐动作严格权限」；enable/disable/
// generate/logs 4 条端点曾按任务书豁免、挂在 ANY-of 读组（RBAC-EXC-01），
// 仅持只读 list 码的角色也能调用状态变更 / 生成工单等写操作。
//
// 2026-11-13 已清偿：4 条端点迁至逐动作严格权限点
// （ops:workorder:periodic:enable/disable/generate/logs），并同步落地
//   - pkg/permission/config.go 权限码常量 + 路由表登记；
//   - internal/core/db/migrations/migration_250_..._perm_seed.go F 型授权载体播种；
//   - 前端 permissionCodes.ts 常量 + 周期工单模板页按钮门控。
//
// 本测试锁定「清偿后的事实」，防止回退：
//  1. 4 条端点仍注册（不得被误下线，防 PR #85 事故复发，见 scripts/FIX-PROTOCOL.md）；
//  2. 4 条端点**必须**出现在动作 strict 写组内，且**不得**留在 ANY-of 读组；
//  3. scripts/RBAC-EXCEPTIONS.md 记录清偿结果（留痕，防静默删除历史）。
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

// periodicActionPaths 4 条已迁逐动作严格权限的动作端点。
var periodicActionPaths = []string{
	"/workorder/periodic/templates/:id/enable",
	"/workorder/periodic/templates/:id/disable",
	"/workorder/periodic/templates/:id/generate",
	"/workorder/periodic/templates/:id/logs",
}

// periodicStrictWritePaths 模板主体 CRUD（逐动作严格写组，作为对照组）。
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

// TestPeriodicActionRoutes_RegisteredInStrictActionGroup 4 条动作端点必须注册在
// SetupPeriodicWorkOrderActionRouter 的逐动作严格组内（清偿后的事实锁定）。
func TestPeriodicActionRoutes_RegisteredInStrictActionGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := ci87WorkOrderCore(t)
	engine := gin.New()
	root := engine.Group("/workorder/periodic/templates")

	groups := PeriodicWorkOrderActionRouteGroups{
		Enable:   root.Group(""),
		Disable:  root.Group(""),
		Generate: root.Group(""),
		Logs:     root.Group(""),
	}
	require.NotPanics(t, func() {
		SetupPeriodicWorkOrderActionRouter(groups, c)
	})
	routes := ci87RouteSet(engine)

	for _, p := range periodicActionPaths {
		assert.True(t, routes[http.MethodPost+" "+p],
			"周期工单动作端点 %s 不应下线（PR #85 误判事故），须保持注册", p)
	}
}

// TestPeriodicActionRoutes_NotInReadGroup 锁定清偿结果：4 条动作端点
// 不得再出现在 ANY-of 读组（SetupPeriodicWorkOrderRouter）内。
// 若将来因故回退到读组，本测试失败并提示同步更新权限口径。
func TestPeriodicActionRoutes_NotInReadGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := ci87WorkOrderCore(t)
	engine := gin.New()
	root := engine.Group("/workorder/periodic")

	require.NotPanics(t, func() {
		SetupPeriodicWorkOrderRouter(root, c)
	})
	routes := ci87RouteSet(engine)

	// 读组只剩 list/statistics 两个读端点。
	assert.True(t, routes[http.MethodPost+" /workorder/periodic/templates/list"])
	assert.True(t, routes[http.MethodPost+" /workorder/periodic/templates/statistics"])

	// 4 条动作端点必须已迁出读组。
	for _, p := range periodicActionPaths {
		assert.False(t, routes[http.MethodPost+" "+p],
			"RBAC-EXC-01 已清偿：%s 应迁至逐动作 strict 组，不应留在 ANY-of 读组", p)
	}
}

// TestPeriodicStrictWritePaths_StillInWriteGroup 模板主体 CRUD 保持在
// 逐动作 strict 写组（对照组，防止清偿动作误伤既有路由）。
func TestPeriodicStrictWritePaths_StillInWriteGroup(t *testing.T) {
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
}

// TestPeriodicActionRoutes_RBACExceptionCleared 清偿留痕守卫：
// scripts/RBAC-EXCEPTIONS.md 必须记录 RBAC-EXC-01 的清偿结果，
// 且源码注释引用该文档（代码↔文档双向可追溯，防静默删除历史）。
func TestPeriodicActionRoutes_RBACExceptionCleared(t *testing.T) {
	root := rbacExceptionRepoRoot(t)

	docPath := filepath.Join(root, "scripts", "RBAC-EXCEPTIONS.md")
	data, err := os.ReadFile(docPath)
	require.NoError(t, err, "scripts/RBAC-EXCEPTIONS.md 必须存在（清偿留痕）")
	doc := string(data)
	for _, must := range []string{
		"RBAC-EXC-01",
		"已清偿",
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
		"workorder_router.go 的动作路由注释必须引用 RBAC-EXCEPTIONS.md")

	rtPath := filepath.Join(root, "internal", "api", "router.go")
	rtSrc, err := os.ReadFile(rtPath)
	require.NoError(t, err)
	assert.Contains(t, string(rtSrc), "scripts/RBAC-EXCEPTIONS.md",
		"router.go 的 periodic 组注释必须引用 RBAC-EXCEPTIONS.md")

	// 清偿后的动作路由注册处必须带 RBAC-EXC-01 标注：
	// 取 SetupPeriodicWorkOrderActionRouter 函数体做局部校验。
	start := strings.Index(string(wrSrc), "func SetupPeriodicWorkOrderActionRouter")
	require.GreaterOrEqual(t, start, 0, "找不到 SetupPeriodicWorkOrderActionRouter")
	fnBody := string(wrSrc)[start:]
	if end := strings.Index(fnBody, "\nfunc "); end > 0 {
		fnBody = fnBody[:end]
	}
	assert.Contains(t, fnBody, "RBAC-EXC-01",
		"动作路由注册处必须带 RBAC-EXC-01 标注")
}
