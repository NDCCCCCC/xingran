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
