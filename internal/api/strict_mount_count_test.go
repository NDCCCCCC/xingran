package api

// strict_mount_count_test.go —— Strict 挂载总量防回退守卫（Phase 85 Plan 04 / W3 收口）。
//
// 与 pkg/permission/route_registry_test.go 互补：那边管"登记对齐 + 逐组单码"，
// 本文件管"总量下限"——任何 RequireStrictPermissions 挂载被悄悄改回
// RequirePermissions / RequirePermissionsWithQuery（strict → ANY-of 回退），
// 总量跌破写死下限即 FAIL。
//
// 三道断言：
//  1. TestStrictMountSourceCount   —— internal/api/**/*.go 源码文本中
//     `RequireStrictPermissions(` 出现次数 ≥ 16（85-03 完成态实测，含
//     strictUserRouteGroup / strictGroup 两处 helper 复用点）。
//  2. TestStrictMountRegistryCount —— route-registry.json 中
//     RequireStrictPermissions 挂载路由 ≥ 188 条（其中写路由 ≥ 164 条）。
//     json 与源码的同步由 TestRouteRegistryFreshness 保证，故此计数等价于
//     源码真实挂载状态，且粒度细到路由条（helper 内一处回退也能拦）。
//  3. TestSystemFourGroupsWritePurity —— system 4 组（roles/menus/departments
//     /posts）21 条已知写路由必须全部保持 RequireStrictPermissions（读组纯化
//     防回退：绝不允许挂回 ANY-of 读组或 WithQuery 子组）。读语义的
//     POST /system/roles/all、/system/posts/enabled 不在断言范围。
//
// 测试只读源码文本与入库 json，不启动 gin、不连 DB。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 85-03 完成态实测写死下限（2026-10-29，85-03 b5d40bc3+c7fbeae2 之后的收口基线）。
const (
	minStrictSourceRefs  = 16  // internal/api 下 RequireStrictPermissions( 源码引用次数
	minStrictMountsTotal = 188 // registry 中 Strict 挂载路由总数
	minStrictMountsWrite = 164 // registry 中 Strict 挂载写路由数
)

func strictMountRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("无法定位测试文件路径")
	}
	// internal/api → 仓库根（与 route_registry_test.go 同深：上溯三级）
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// TestStrictMountSourceCount 源码文本总量守卫。
func TestStrictMountSourceCount(t *testing.T) {
	root := strictMountRepoRoot(t)
	count := 0
	err := filepath.WalkDir(filepath.Join(root, "internal", "api"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		count += strings.Count(string(data), "RequireStrictPermissions(")
		return nil
	})
	if err != nil {
		t.Fatalf("扫描 internal/api 源码失败: %v", err)
	}
	if count < minStrictSourceRefs {
		t.Fatalf("Strict 挂载源码引用数 %d < 下限 %d：检测到 strict→ANY-of 回退（RequireStrictPermissions 被改回 RequirePermissions / WithQuery 或写组被删除），逐处核对 internal/api 下的挂载改动", count, minStrictSourceRefs)
	}
	t.Logf("源码守卫通过: RequireStrictPermissions( 引用 %d 处 ≥ %d", count, minStrictSourceRefs)
}

type strictMountRoute struct {
	Method   string   `json:"method"`
	Path     string   `json:"path"`
	IsWrite  bool     `json:"isWrite"`
	PermKind *string  `json:"permKind"`
}

func (r strictMountRoute) kind() string {
	if r.PermKind == nil {
		return ""
	}
	return *r.PermKind
}

func loadStrictMountRegistry(t *testing.T) []strictMountRoute {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(strictMountRepoRoot(t), "scripts", "route-registry.json"))
	if err != nil {
		t.Fatalf("读取 route-registry.json 失败: %v", err)
	}
	var reg struct {
		Routes []strictMountRoute `json:"routes"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("解析 route-registry.json 失败: %v", err)
	}
	if len(reg.Routes) == 0 {
		t.Fatalf("route-registry.json routes 为空，扫描器产物异常")
	}
	return reg.Routes
}

// TestStrictMountRegistryCount registry 总量守卫（细粒度：任何一条 Strict 路由
// 退化为 ANY-of 即跌破下限）。
func TestStrictMountRegistryCount(t *testing.T) {
	routes := loadStrictMountRegistry(t)
	total, write := 0, 0
	for _, r := range routes {
		if strings.HasPrefix(r.kind(), "RequireStrictPermissions") {
			total++
			if r.IsWrite {
				write++
			}
		}
	}
	if total < minStrictMountsTotal || write < minStrictMountsWrite {
		t.Fatalf("Strict 挂载路由 total=%d(下限 %d) write=%d(下限 %d)：检测到挂载回退，重扫核对 node scripts/scan-routes.cjs", total, minStrictMountsTotal, write, minStrictMountsWrite)
	}
	t.Logf("registry 守卫通过: Strict 挂载 %d 条（写路由 %d 条）≥ (%d, %d)", total, write, minStrictMountsTotal, minStrictMountsWrite)
}

// systemFourGroupsWritePaths system 4 组 21 条写路由（POST 方法但读语义的
// /system/roles/all、/system/posts/enabled 不在断言范围）。
var systemFourGroupsWritePaths = []string{
	// roles 5
	"/system/roles",
	"/system/roles/:id/update",
	"/system/roles/:id/status",
	"/system/roles/:id/delete",
	"/system/roles/batch",
	// menus 5
	"/system/menus",
	"/system/menus/:id/update",
	"/system/menus/:id/status",
	"/system/menus/:id/delete",
	"/system/menus/batch-delete",
	// departments 7（含 Excel import/export 独立子组）
	"/system/departments",
	"/system/departments/:id/update",
	"/system/departments/:id/status",
	"/system/departments/:id/delete",
	"/system/departments/batch",
	"/system/departments/import",
	"/system/departments/export",
	// posts 4
	"/system/posts",
	"/system/posts/:id/update",
	"/system/posts/:id/delete",
	"/system/posts/batch-delete",
}

// TestSystemFourGroupsWritePurity system 4 组写路由必须全部保持
// RequireStrictPermissions（读组纯化防回退）。
func TestSystemFourGroupsWritePurity(t *testing.T) {
	routes := loadStrictMountRegistry(t)
	byKey := make(map[string]string, len(routes)) // "POST path" → permKind
	for _, r := range routes {
		byKey[r.Method+" "+r.Path] = r.kind()
	}

	var violations []string
	for _, path := range systemFourGroupsWritePaths {
		kind, ok := byKey["POST "+path]
		if !ok {
			violations = append(violations, fmt.Sprintf("POST %s 路由消失（不得移除拆分后的写路由）", path))
			continue
		}
		if !strings.HasPrefix(kind, "RequireStrictPermissions") {
			violations = append(violations, fmt.Sprintf("POST %s 权限中间件退化为 %q（必须保持 RequireStrictPermissions 单一权限点，不得挂回 ANY-of 读组/WithQuery）", path, kind))
		}
	}
	if len(violations) > 0 {
		t.Fatalf("system 4 组写路由纯化守卫失败 %d 项:\n%s", len(violations), strings.Join(violations, "\n"))
	}
	t.Logf("纯化守卫通过: system 4 组 %d 条写路由全部保持 RequireStrictPermissions", len(systemFourGroupsWritePaths))
}
