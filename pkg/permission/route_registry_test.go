package permission

// route_registry_test.go —— 路由登记双层守卫之 go test 侧（Phase 85 Plan 01 / RBAC-03）。
//
// 守卫目标（接线选型 = CI 校验，GetRoutePermissions 不进请求链路）：
//   1. TestRouteRegistryCoverage  —— 每条 isWrite=true 路由必须被 GetRoutePermissions()
//      登记（GetPermissionByPath 返回非空码，:id 占位走 matchPath 段匹配语义），
//      或显式出现在 scripts/route-registry.json 的 deferred.items 冻结清单
//      （85-02/03 拆分同批收缩）。任一未登记且未冻结 → t.Fatalf 列出缺失清单。
//   2. TestRouteRegistryFreshness —— 重跑 scripts/scan-routes.cjs（--out 临时文件，
//      不触碰入库产物），与入库 json 的 routes 数组逐条比对；路由注册代码变了
//      而产物未重生成 → FAIL。注入未登记假写路由时本测试同步 FAIL（红绿验证）。
//
// node 不可用的环境（无前端的纯 Go CI）下 Freshness 退化为 t.Skip，
// Coverage 仍基于入库 json 全量执行。

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type registryFile struct {
	Totals   struct {
		Mounts              int `json:"mounts"`
		Routes              int `json:"routes"`
		WriteRoutes         int `json:"writeRoutes"`
		DeferredWriteRoutes int `json:"deferredWriteRoutes"`
	} `json:"totals"`
	Routes []struct {
		Method               string   `json:"method"`
		Path                 string   `json:"path"`
		MountGroup           string   `json:"mountGroup"`
		Setup                string   `json:"setup"`
		IsWrite              bool     `json:"isWrite"`
		Handler              string   `json:"handler"`
		PermKind             *string  `json:"permKind"`
		PermCodes            []string `json:"permCodes"`
		RegisteredDeadTable  bool     `json:"registeredDeadTable"`
		File                 string   `json:"file"`
		Line                 int      `json:"line"`
	} `json:"routes"`
	Deferred struct {
		Note  string `json:"note"`
		Items []struct {
			Path    string `json:"path"`
			Method  string `json:"method"`
			Handler string `json:"handler"`
		} `json:"items"`
	} `json:"deferred"`
}

// repoRoot 从本文件位置向上定位仓库根（pkg/permission → 仓库根）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("无法定位测试文件路径")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

func loadRegistry(t *testing.T) registryFile {
	t.Helper()
	path := filepath.Join(repoRoot(t), "scripts", "route-registry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 route-registry.json 失败: %v", err)
	}
	var reg registryFile
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("解析 route-registry.json 失败: %v", err)
	}
	if len(reg.Routes) == 0 {
		t.Fatalf("route-registry.json routes 为空，扫描器产物异常")
	}
	return reg
}

// TestRouteRegistryCoverage 未登记写路由守卫（deferred 冻结清单豁免制）。
func TestRouteRegistryCoverage(t *testing.T) {
	reg := loadRegistry(t)

	deferredSet := make(map[string]bool)
	for _, d := range reg.Deferred.Items {
		deferredSet[d.Method+" "+d.Path] = true
	}

	var missing, staleDeferred []string
	checked := 0
	for _, r := range reg.Routes {
		if !r.IsWrite {
			continue
		}
		checked++
		key := r.Method + " " + r.Path
		if !r.RegisteredDeadTable {
			if !deferredSet[key] {
				missing = append(missing, key)
			}
			continue
		}
		// 已登记：断言 GetPermissionByPath 经 matchPath 语义命中非空权限码（路径+方法+码三对齐）
		if code := GetPermissionByPath(r.Path, r.Method); code == "" {
			missing = append(missing, key+" (登记表存在但 GetPermissionByPath 未命中，检查登记路径/方法)")
		}
		if deferredSet[key] {
			staleDeferred = append(staleDeferred, key)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("发现 %d 条未登记写路由（须登记 GetRoutePermissions 或显式扩 deferred 冻结清单）:\n%s",
			len(missing), strings.Join(missing, "\n"))
	}
	if len(staleDeferred) > 0 {
		t.Fatalf("deferred 冻结清单中 %d 条已登记，应随拆分批次收缩并重生成 route-registry.json:\n%s",
			len(staleDeferred), strings.Join(staleDeferred, "\n"))
	}
	t.Logf("coverage 守卫通过: 写路由 %d 条, deferred %d 条", checked, len(reg.Deferred.Items))
}

// strictWriteGroups 85-02（RBAC-04）拆分为逐动作严格单一权限点的 system 4 组写路由
// （path → 期望死表权限码）。写组若被改回挂 ANY-of 读组（或做成读组子组），
// 扫描到的 permCodes 会合并父组 ANY-of 多码，本测试立即 FAIL。
var strictWriteGroups = map[string]PermissionCode{
	// roles
	"/system/roles": RoleAdd,
	"/system/roles/:id/update": RoleEdit,
	"/system/roles/:id/status": RoleEdit,
	"/system/roles/:id/delete": RoleRemove,
	"/system/roles/batch":      RoleRemove,
	// menus
	"/system/menus": MenuAdd,
	"/system/menus/:id/update": MenuEdit,
	"/system/menus/:id/status": MenuEdit,
	"/system/menus/:id/delete": MenuRemove,
	"/system/menus/batch-delete": MenuRemove,
	// departments
	"/system/departments": DeptAdd,
	"/system/departments/:id/update": DeptEdit,
	"/system/departments/:id/status": DeptEdit,
	"/system/departments/:id/delete": DeptRemove,
	"/system/departments/batch":      DeptRemove,
	"/system/departments/import":     DeptImport,
	"/system/departments/export":     DeptExport,
	// posts
	"/system/posts": PostAdd,
	"/system/posts/:id/update": PostEdit,
	"/system/posts/:id/delete": PostRemove,
	"/system/posts/batch-delete": PostRemove,

	// 85-03 批次 A：ops 域 13 组写路由（internal/api/router.go 内联兄弟写组，
	// 形态 var := strictUserRouteGroup(parent, permission.OpsXxx, core)）
	"/ops/building":              OpsBuildingAdd,
	"/ops/building/:id/update":   OpsBuildingEdit,
	"/ops/building/:id/delete":   OpsBuildingDelete,
	"/ops/building/batch":        OpsBuildingDelete,
	"/ops/building/import":       OpsBuildingImport,
	"/ops/building/export":       OpsBuildingExport,
	"/ops/floor":                 OpsFloorAdd,
	"/ops/floor/:id/update":      OpsFloorEdit,
	"/ops/floor/:id/delete":      OpsFloorDelete,
	"/ops/floor/batch":           OpsFloorDelete,
	"/ops/floor/import":          OpsFloorImport,
	"/ops/floor/export":          OpsFloorExport,
	"/ops/workstation":           OpsWorkstationAdd,
	"/ops/workstation/:id/update":  OpsWorkstationEdit,
	"/ops/workstation/positions":   OpsWorkstationEdit,
	"/ops/workstation/:id/delete":  OpsWorkstationDelete,
	"/ops/workstation/batch":       OpsWorkstationDelete,
	"/ops/workstation/import":      OpsWorkstationImport,
	"/ops/workstation/export":      OpsWorkstationExport,
	"/ops/workstation-device/manual":               OpsWorkstationAdd,
	"/ops/workstation-device/sync-ad":              OpsWorkstationEdit,
	"/ops/workstation-device/sync-asset":           OpsWorkstationEdit,
	"/ops/workstation-device/:id/update":           OpsWorkstationEdit,
	"/ops/workstation-device/:id/set-primary":      OpsWorkstationEdit,
	"/ops/workstation-device/:id/set-primary-and-save": OpsWorkstationEdit,
	"/ops/workstation-device/:id/delete":           OpsWorkstationDelete,
	"/ops/asset":               OpsAssetAdd,
	"/ops/asset/:id/update":    OpsAssetEdit,
	"/ops/asset/:id/delete":    OpsAssetDelete,
	"/ops/asset/batch":         OpsAssetDelete,
	"/ops/asset/import":        OpsAssetImport,
	"/ops/asset/export":        OpsAssetExport,
	"/ops/infoPoint":           OpsInfoPointAdd,
	"/ops/infoPoint/:id/update": OpsInfoPointEdit,
	"/ops/infoPoint/:id/delete": OpsInfoPointDelete,
	"/ops/infoPoint/batch":     OpsInfoPointDelete,
	"/ops/infoPoint/import":    OpsInfoPointImport,
	"/ops/infoPoint/export":    OpsInfoPointExport,
	"/ops/serverRoom":          OpsServerRoomAdd,
	"/ops/serverRoom/:id/update": OpsServerRoomEdit,
	"/ops/serverRoom/:id/delete": OpsServerRoomDelete,
	"/ops/serverRoom/batch":    OpsServerRoomDelete,
	"/ops/serverRoom/import":   OpsServerRoomImport,
	"/ops/serverRoom/export":   OpsServerRoomExport,
	"/ops/dedicatedLine":       OpsDedicatedLineAdd,
	"/ops/dedicatedLine/:id/update": OpsDedicatedLineEdit,
	"/ops/dedicatedLine/:id/delete": OpsDedicatedLineDelete,
	"/ops/dedicatedLine/batch": OpsDedicatedLineDelete,
	"/ops/dedicatedLine/import": OpsDedicatedLineImport,
	"/ops/dedicatedLine/export": OpsDedicatedLineExport,
	"/ops/roomDevice":          OpsRoomDeviceAdd,
	"/ops/roomDevice/:id/update": OpsRoomDeviceEdit,
	"/ops/roomDevice/:id/delete": OpsRoomDeviceDelete,
	"/ops/roomDevice/batch":    OpsRoomDeviceDelete,
	"/ops/roomDevice/import":   OpsRoomDeviceImport,
	"/ops/roomDevice/export":   OpsRoomDeviceExport,
	"/ops/location-alias":          OpsLocationAliasAdd,
	"/ops/location-alias/:id/update": OpsLocationAliasEdit,
	"/ops/location-alias/:id/delete": OpsLocationAliasDelete,
	"/ops/walls":             OpsWallAdd,
	"/ops/walls/:id/update":  OpsWallEdit,
	"/ops/walls/:id/delete":  OpsWallDelete,
	"/ops/walls/batch":       OpsWallDelete,
	"/ops/doors":             OpsDoorAdd,
	"/ops/doors/:id/update":  OpsDoorEdit,
	"/ops/doors/:id/delete":  OpsDoorDelete,
	"/ops/doors/batch":       OpsDoorDelete,
	"/ops/floor-plan-texts":          OpsFloorPlanEdit,
	"/ops/floor-plan-texts/:id/update": OpsFloorPlanEdit,
	"/ops/floor-plan-texts/:id/delete": OpsFloorPlanEdit,
	"/ops/floor-plan-texts/batch":      OpsFloorPlanEdit,

	// 85-03 批次 B：network 域 8 组写路由（network_router.go strictGroup 兄弟写组；
	// 拓扑/mac/ports 读组为 PLAN 覆盖清单外，不在守护范围）
	"/network/devices":               NetworkDeviceAdd,
	"/network/devices/quick-create":  NetworkDeviceAdd,
	"/network/devices/:id/update":    NetworkDeviceEdit,
	"/network/devices/:id/delete":    NetworkDeviceDelete,
	"/network/devices/batch-delete":  NetworkDeviceDelete,
	"/network/devices/export":        NetworkDeviceExport,
	"/network/credentials":           NetworkCredentialAdd,
	"/network/credentials/:id/update":    NetworkCredentialEdit,
	"/network/credentials/:id/set-default": NetworkCredentialEdit,
	"/network/credentials/:id/delete":    NetworkCredentialDelete,
	"/network/credentials/batch-delete":  NetworkCredentialDelete,
	"/network/credentials/export":        NetworkCredentialExport,
	"/network/templates":            NetworkTemplateAdd,
	"/network/templates/:id/clone":  NetworkTemplateAdd,
	"/network/templates/:id/update": NetworkTemplateEdit,
	"/network/templates/:id/delete": NetworkTemplateDelete,
	"/network/templates/batch-delete": NetworkTemplateDelete,
	"/network/templates/:id/preview":  NetworkTemplateView,
	"/network/templates/export":       NetworkTemplateExport,
	"/network/command/dispatch": NetworkCommandExecute,
	"/network/command/quick":    NetworkCommandExecute,
	"/network/command/export":   NetworkCommandExport,
	"/network/executions/template/execute": NetworkCommandExecute,
	"/network/executions/:id/delete":       NetworkCommandExecute,
	"/network/executions/batch-delete":     NetworkCommandExecute,
	"/network/executions/export":           NetworkCommandExecute,
	"/network/backups":              NetworkBackupAdd,
	"/network/backups/batch":        NetworkBackupAdd,
	"/network/backups/diff":         NetworkBackupDiff,
	"/network/backups/:id/delete":   NetworkBackupDelete,
	"/network/backups/batch-delete": NetworkBackupDelete,
	"/network/backups/export":       NetworkBackupExport,
	"/network/backups/:id/restore":  NetworkBackupRestore,
	"/network/discoveries/create":    NetworkDiscoveryAdd,
	"/network/discoveries/:id/execute": NetworkDiscoveryAdd,
	"/network/discoveries/:id/import":  NetworkDiscoveryAdd,
	"/network/discoveries/:id/delete":  NetworkDiscoveryDelete,
	"/network/discoveries/batch-delete": NetworkDiscoveryDelete,
	"/network/discoveries/export":       NetworkDiscoveryExport,

	// 85-03 批次 C：system 其余组（apikeys/configs/dicts/notices 子 router 拆分 +
	// manager/ad 内联 strict 兄弟组 + ad accounts Require→Strict 升级）
	"/system/apikeys":           APIKeyAdd,
	"/system/apikeys/:id/update": APIKeyEdit,
	"/system/apikeys/:id/toggle": APIKeyEdit,
	"/system/apikeys/:id/delete": APIKeyRemove,
	"/system/configs":               ConfigAdd,
	"/system/configs/:id/update":    ConfigEdit,
	"/system/configs/:id/delete":    ConfigRemove,
	"/system/configs/batch-delete":  ConfigRemove,
	"/system/configs/refresh-cache": ConfigEdit,
	"/system/dicts/types":            DictTypeAdd,
	"/system/dicts/types/:id/update": DictTypeEdit,
	"/system/dicts/types/:id/delete": DictTypeRemove,
	"/system/dicts/data":             DictTypeAdd,
	"/system/dicts/data/:id/update":  DictTypeEdit,
	"/system/dicts/data/:id/delete":  DictTypeRemove,
	"/system/notices":               NoticeAdd,
	"/system/notices/:id/update":    NoticeEdit,
	"/system/notices/:id/delete":    NoticeRemove,
	"/system/notices/batch-delete":  NoticeRemove,
	"/system/notices/:id/publish":   NoticeEdit,
	"/system/notices/:id/withdraw":  NoticeEdit,
	"/system/notices/:id/channels":  NoticeEdit,
	"/system/manager-rules":               OpsManagerRuleEdit,
	"/system/manager-rules/:id/update":    OpsManagerRuleEdit,
	"/system/manager-rules/:id/delete":    OpsManagerRuleDelete,
	"/system/manager-orgs":                OpsManagerOrgEdit,
	"/system/manager-orgs/:id/update":     OpsManagerOrgEdit,
	"/system/manager-orgs/:id/delete":     OpsManagerOrgDelete,
	"/system/manager-orgs/generate-from-dept": OpsManagerOrgEdit,
	"/ad-domain/configs":            OpsADConfigAdd,
	"/ad-domain/configs/:id/update": OpsADConfigEdit,
	"/ad-domain/configs/:id/delete": OpsADConfigEdit,
	"/ad-domain/configs/:id/test":   OpsADConfigTest,
	"/ad-domain/configs/:id/sync":   OpsADConfigSync,
	"/ad-domain/groups/:id/update":  OpsADGroupEdit,
	"/ad-domain/users/:id/update":   OpsADUserEdit,
	"/ad-domain/users/:id/move":     OpsADUserEdit,
	"/ad-domain/users/:id/enable":   OpsADUserEdit,
	"/ad-domain/users/:id/disable":  OpsADUserEdit,
	"/ad-domain/users/batch-sync":   OpsADUserEdit,
	"/ad-domain/accounts/create": PermissionCode("ops:ad:config:account:edit"),
	"/ad-domain/accounts/delete": PermissionCode("ops:ad:config:account:delete"),
}

// TestStrictWriteGroupsSingleCode 85-02 拆分守护：system 4 组写路由必须保持
// RequireStrictPermissions 单一权限点，且死表登记码与实际挂载码一致（三对齐）。
func TestStrictWriteGroupsSingleCode(t *testing.T) {
	reg := loadRegistry(t)
	byKey := make(map[string]struct {
		PermKind  *string
		PermCodes []string
	})
	for _, r := range reg.Routes {
		byKey[r.Method+" "+r.Path] = struct {
			PermKind  *string
			PermCodes []string
		}{r.PermKind, r.PermCodes}
	}

	var violations []string
	for path, wantCode := range strictWriteGroups {
		r, ok := byKey["POST "+path]
		if !ok {
			violations = append(violations, "POST "+path+" 路由消失（不得移除拆分后的写路由）")
			continue
		}
		kind := ""
		if r.PermKind != nil {
			kind = *r.PermKind
		}
		if !strings.HasPrefix(kind, "RequireStrictPermissions") {
			violations = append(violations, "POST "+path+" 权限中间件退化为 "+kind+"（必须是 RequireStrictPermissions 单一权限点）")
			continue
		}
		if len(r.PermCodes) != 1 {
			violations = append(violations, fmt.Sprintf("POST %s permCodes=%v（严格写路由必须恰为单一权限点）", path, r.PermCodes))
			continue
		}
		if got := PermissionCode(r.PermCodes[0]); got != wantCode {
			violations = append(violations, fmt.Sprintf("POST %s 挂载码 %s ≠ 死表码 %s", path, got, wantCode))
		}
		if dead := GetPermissionByPath(path, "POST"); dead != wantCode {
			violations = append(violations, fmt.Sprintf("POST %s 死表命中 %s ≠ 期望 %s", path, dead, wantCode))
		}
	}
	if len(violations) > 0 {
		t.Fatalf("system 4 组严格写路由守护失败 %d 项:\n%s", len(violations), strings.Join(violations, "\n"))
	}
	t.Logf("严格写路由守护通过: %d 条逐动作单一权限点", len(strictWriteGroups))
}

// TestRouteRegistryFreshness 重扫比对守卫（防「代码改了表没重生成」）。
func TestRouteRegistryFreshness(t *testing.T) {
	root := repoRoot(t)
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node 不在 PATH，freshness 重扫退化为 CI 侧 check-route-registry.mjs 承担")
	}

	outFile := filepath.Join(t.TempDir(), "route-registry-fresh.json")
	cmd := exec.Command(node, filepath.Join("scripts", "scan-routes.cjs"), "--out", outFile)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("重跑 scan-routes.cjs 失败: %v\n%s", err, out)
	}

	freshData, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("读取重扫产物失败: %v", err)
	}
	var fresh registryFile
	if err := json.Unmarshal(freshData, &fresh); err != nil {
		t.Fatalf("解析重扫产物失败: %v", err)
	}

	committed := loadRegistry(t)

	committedSet := make(map[string]bool)
	for _, r := range committed.Routes {
		committedSet[r.Method+" "+r.Path] = true
	}
	freshSet := make(map[string]bool)
	for _, r := range fresh.Routes {
		freshSet[r.Method+" "+r.Path] = true
	}

	var added, removed []string
	for k := range freshSet {
		if !committedSet[k] {
			added = append(added, k)
		}
	}
	for k := range committedSet {
		if !freshSet[k] {
			removed = append(removed, k)
		}
	}
	if len(added) > 0 || len(removed) > 0 {
		t.Fatalf("route-registry.json 已过期（路由注册代码与入库产物不一致），运行 node scripts/scan-routes.cjs 重生成.\n新增未入库:\n%s\n已消失未移除:\n%s",
			strings.Join(added, "\n"), strings.Join(removed, "\n"))
	}

	// 登记状态变化同样要求重生成
	var regChanged []string
	for _, r := range fresh.Routes {
		for _, c := range committed.Routes {
			if r.Method == c.Method && r.Path == c.Path && r.RegisteredDeadTable != c.RegisteredDeadTable {
				regChanged = append(regChanged, r.Method+" "+r.Path)
			}
		}
	}
	if len(regChanged) > 0 {
		t.Fatalf("死表登记状态变化未同步到 route-registry.json，运行 node scripts/scan-routes.cjs 重生成:\n%s",
			strings.Join(regChanged, "\n"))
	}
	t.Logf("freshness 守卫通过: %d 条路由与入库产物一致", len(fresh.Routes))
}
