package api

// datascope_mount_test.go —— DS-02 挂载点防回退守卫（v1.52 Phase 130）。
//
// 守卫目标：ops 全域 13 个读组 + 13 个 strict 写组必须全部挂载
// middleware.DataScopePermission，且读组挂载必须位于权限校验之后、handler 之前。
// 与 pkg/permission/route_registry_test.go（Strict 双层守卫）互补：那边管权限码登记，
// 本文件管 data_scope/user_id 注入面的覆盖度与顺序。
//
// 方案 = 源码文本扫描（与 strict_mount_count_test.go 一致，不启动 gin、不连 DB）。
// 扫描器 scripts/scan-routes.cjs 只识别 middleware.Require*，看不到 DataScopePermission，
// 故本测试是 DS-02 注入面的唯一自动闸门。

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func datascopeMountRouterSource(t *testing.T) []string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("无法定位测试文件路径")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	data, err := os.ReadFile(filepath.Join(root, "internal", "api", "router.go"))
	if err != nil {
		t.Fatalf("读取 router.go 失败: %v", err)
	}
	return strings.Split(string(data), "\n")
}

// opsReadGroups / opsWriteGroups 是 DS-02 收口的 ops 全域组变量名（router.go 中
// `xxx := ops.Group("/...")` 的赋值名）。新增 ops 组时必须同步扩此清单并重挂中间件，
// 否则本测试立即 FAIL（防「新增端点漏注入」）。
var opsReadGroups = []string{
	"buildings", "floors", "workstations", "locationAlias", "workstationDevices",
	"assets", "infoPoints", "serverRooms", "dedicatedLines", "roomDevices",
	"walls", "doors", "texts",
}

var opsWriteGroups = []string{
	"buildingWrites", "floorWrites", "workstationWrites", "locationAliasWrites",
	"workstationDeviceWrites", "assetWrites", "infoPointWrites", "serverRoomWrites",
	"dedicatedLineWrites", "roomDeviceWrites", "wallsWrites", "doorsWrites", "textsWrites",
}

// findUseLines 返回 `<var>.Use(<prefix>` 命中的行号（1-based）。
func findUseLines(lines []string, varName, prefix string) []int {
	needle := varName + ".Use(" + prefix
	var out []int
	for i, l := range lines {
		if strings.Contains(l, needle) {
			out = append(out, i+1)
		}
	}
	return out
}

// TestDataScopeMount_OpsCoverage 断言 ops 全域 26 个组全部挂载 DataScopePermission，
// 且读组挂载顺序在权限校验之后（权限 abort 时不会白跑一次 getUserMaxDataScope）。
func TestDataScopeMount_OpsCoverage(t *testing.T) {
	lines := datascopeMountRouterSource(t)

	// 1) 既有 5 处（system usersRead/usersWrite/deptsRead/deptsWrite + vdi vms）不得被误删，
	//    加上 ops 26 处 = 31。
	total := 0
	for _, l := range lines {
		if strings.Contains(l, ".Use(middleware.DataScopePermission(core))") {
			total++
		}
	}
	const wantTotal = 5 + 26
	if total != wantTotal {
		t.Fatalf("DataScopePermission 挂载总数 = %d，期望 %d（既有 5 + ops 读组 13 + ops 写组 13）；新增/删除挂载必须同步本测试与 ops 域", total, wantTotal)
	}

	// 2) 26 个组逐一断言已挂载。
	var missing []string
	for _, g := range append(append([]string{}, opsReadGroups...), opsWriteGroups...) {
		if len(findUseLines(lines, g, "middleware.DataScopePermission(core)")) != 1 {
			missing = append(missing, g)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("以下 ops 组未挂载 DataScopePermission: %s", strings.Join(missing, ", "))
	}

	// 3) 读组：权限校验必须在 DataScopePermission 之前（"权限校验之后、handler 之前"）。
	var orderViolations []string
	for _, g := range opsReadGroups {
		req := findUseLines(lines, g, "middleware.Require")
		ds := findUseLines(lines, g, "middleware.DataScopePermission(core)")
		if len(req) == 0 {
			orderViolations = append(orderViolations, g+" 未找到 Require* 权限中间件")
			continue
		}
		if len(ds) == 0 || ds[0] < req[len(req)-1] {
			orderViolations = append(orderViolations, g+" 的 DataScopePermission 不在权限校验之后")
		}
	}
	if len(orderViolations) > 0 {
		t.Fatalf("读组注入顺序违规:\n%s", strings.Join(orderViolations, "\n"))
	}

	t.Logf("DS-02 挂载守卫通过: 总数 %d（5 既有 + 13 读组 + 13 写组），读组顺序全部为 Require* → DataScopePermission", total)
}

// TestDataScopeMount_NotOnOpsAncestor 防「挂到 ops 祖先组」的坑：
// 祖先组中间件会被所有子组继承（含显式豁免的 roomPhotos 几何/照片端点），
// 且会在各组权限校验之前运行。DataScopePermission 必须只挂在 26 个叶子组上。
func TestDataScopeMount_NotOnOpsAncestor(t *testing.T) {
	lines := datascopeMountRouterSource(t)
	for i, l := range lines {
		if strings.Contains(l, "ops.Use(middleware.DataScopePermission") ||
			strings.Contains(l, "r.Use(middleware.DataScopePermission") {
			t.Fatalf("router.go:%d 把 DataScopePermission 挂到了祖先组（会被所有子组继承且在权限校验之前）: %s", i+1, strings.TrimSpace(l))
		}
	}
	// 显式豁免组 roomPhotos（无权限中间件、几何/照片端点，REQUIREMENTS 映射总表登记豁免）不得挂。
	if n := len(findUseLines(lines, "roomPhotos", "middleware.DataScopePermission(core)")); n != 0 {
		t.Fatalf("豁免组 roomPhotos 不应挂载 DataScopePermission，实际 %d 处", n)
	}
}
