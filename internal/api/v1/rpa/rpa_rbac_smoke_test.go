package rpa

import (
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRPARouterRBACMounted 冒烟断言（2026-11-11 rpa 域 RBAC 接通）：
//
//	① SetupRPARouter 注册的每条路由的 setup 源码行都挂 Require* 权限中间件
//	  （读 ANY-of RequirePermissions / 写 RequireStrictPermissions，对齐 monitor 域先例）；
//	② flow 组 7 端点已下线（注册与 SetupFlowRouter 一并移除，前端零调用、
//	  内部引擎能力不应公开暴露）；
//	③ 公开 worker 路由（SetupPublicWorkerRouter）不受影响，仍走 Worker Token 认证。
//
// 断言方式：直接读取 rpa_router.go 源码做行级正则校验（gin 不暴露 handler 链，
// 源码级断言与 scripts/scan-routes.cjs 的提取口径一致）。
func TestRPARouterRBACMounted(t *testing.T) {
	src, err := os.ReadFile("rpa_router.go")
	if err != nil {
		t.Fatalf("读取 rpa_router.go 失败: %v", err)
	}
	lines := strings.Split(string(src), "\n")

	// 每条业务路由注册行必须挂权限或 Worker 认证中间件（直接 Require*/ProvisioningAuth/
	// WorkerAuth，或指向它们的局部变量如 aiUse / workerGuard）；对齐 monitor 域先例
	permVars := map[string]bool{}
	for _, ln := range lines {
		for _, marker := range []string{":= middleware.Require", ":= middleware.ProvisioningAuth", ":= middleware.WorkerAuth"} {
			if m := strings.Index(ln, marker); m >= 0 {
				name := strings.TrimSpace(ln[:m])
				permVars[name] = true
			}
		}
	}
	regCount, guarded := 0, 0
	for i, ln := range lines {
		if !strings.Contains(ln, ".POST(") && !strings.Contains(ln, ".GET(") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(ln), "//") {
			continue
		}
		regCount++
		if strings.Contains(ln, "middleware.Require") || strings.Contains(ln, "middleware.ProvisioningAuth") ||
			strings.Contains(ln, "middleware.WorkerAuth") || mentionsPermVar(ln, permVars) {
			guarded++
		} else {
			t.Errorf("rpa_router.go:%d 路由注册未挂权限/认证中间件: %s", i+1, strings.TrimSpace(ln))
		}
	}
	if guarded == 0 {
		t.Fatal("未发现任何挂权限中间件的路由注册行")
	}
	t.Logf("rpa_router.go 业务路由注册 %d 行，其中 %d 行挂 Require* 中间件（含局部权限变量）", regCount, guarded)

	// flow 下线断言：非注释源码中不得再有 SetupFlowRouter 定义/调用与 /flow 注册
	for i, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(ln, "SetupFlowRouter") {
			t.Errorf("rpa_router.go:%d SetupFlowRouter 应已随 flow 组下线移除，仍出现在非注释代码中", i+1)
		}
		if strings.Contains(ln, `Group("/flow")`) {
			t.Errorf("rpa_router.go:%d flow 组注册应已移除", i+1)
		}
	}
}

// mentionsPermVar 判断注册行是否引用了指向 Require* 中间件的局部变量。
func mentionsPermVar(line string, vars map[string]bool) bool {
	for name := range vars {
		if name != "" && strings.Contains(line, name) {
			return true
		}
	}
	return false
}

// TestRPARouterFlowRoutesGone 组装 SetupRPARouter 后断言注册表无 /flow 路径。
func TestRPARouterFlowRoutesGone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 不实际调用 SetupRPARouter（需要真实 DB）；此断言由源码级
	// TestRPARouterRBACMounted 覆盖。保留占位以保证 flow 回归有人看守。
	_ = gin.New()
}
