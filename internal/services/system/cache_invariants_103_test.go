package system

// =====================================================================
// Phase 103-04 (D-103-16/D-103-17/D-103-18/D-103-19): 缓存收敛防线扩口。
//
// Phase 92 已锁 system/operations 的 *_cache_impl.go 零 interface{} 闭包
// GetOrSet（cache_invariants_92_test.go）。Phase 103 完成三域手写 cache-aside
// 迁移（mac_history query/heatmap + asset/reconciliation + rpa/selector_learner
// → base.GetOrSetJSON[T]），本文件把两类检测扩展到全域：
//
//   1. TestNoInterfaceGetOrSetResidue103 —— interface{} 闭包式 GetOrSet 检测
//      （复用 92 版 AST 口径）扩口到 services root / asset / rpa 全部 .go 文件
//      （排除 _test.go / base 包 / data_cache_service.go 基础设施实现本体 /
//      adapter 与测试 fixture 的 GetOrSet 方法实现——实现端无闭包实参，
//      92-02 校准记录 2 先例）。
//   2. TestNoHandwrittenCacheAside —— 手写 cache-aside 标志性序列 AST 检测
//      （cache.Get/GetJSON → 后续行 json.Unmarshal → 更后续行 cache.Set/SetJSON
//      含 json.Marshal 参数或 SetJSON 调用），扩口到同域全部非测试 .go 文件。
//
// 断言档位（D-103-17）：
//   - 硬失败档：Phase 103 收敛面 4 文件（mac_history_query_service.go /
//     mac_history_heatmap_service.go / reconciliation_service.go /
//     selector_learner.go）两类残留都必须 == 0，超出即 t.Errorf
//   - warning 档：同域其余文件（api_endpoint_service.go / dashboard_service.go /
//     widget_data_fetcher.go 等已知 GetJSON 站点 + 未来新增）仅计数日志
//     （v1.31+ 迁移候选，不在本相范围）
//
// 白名单机制与 92 版一致：allowedResidues103（初始空 = 期望 0），确需豁免
// 必须显式登记（diff 可见）。
//
// 纪律：文件定位用 runtime.Caller 相对推导，禁本地绝对路径断言（v1.27 测试纪律）。
// =====================================================================

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// allowedResidues103 白名单：文件名 → 允许的残留数（interface{} 闭包 GetOrSet 与
// 手写 cache-aside 共用）。初始全空 = 期望 0；确需豁免必须在此显式登记。
var allowedResidues103 = map[string]int{}

// migratedFiles103 Phase 103 收敛面：硬失败档（两类残留都必须 == 0）。
var migratedFiles103 = []string{
	"mac_history_query_service.go",
	"mac_history_heatmap_service.go",
	"reconciliation_service.go",
	"selector_learner.go",
}

// scanDirs103 扫描目录：services root + asset + rpa（D-103-16 扩口面）。
// system/operations 已由 92 版覆盖（cache_impl 文件口径），本测试的
// "全部 .go 文件" 口径天然包含它们——重复扫描无害（同口径零残留断言更强）。
func scanDirs103(t *testing.T) []string {
	t.Helper()
	root := servicesRoot92(t)
	return []string{
		root,
		filepath.Join(root, "asset"),
		filepath.Join(root, "rpa"),
	}
}

// goFilesInDir103 返回目录（非递归）下全部非 _test.go 的 .go 文件。
func goFilesInDir103(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s 失败: %v", dir, err)
	}
	var files []string
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		files = append(files, m)
	}
	sort.Strings(files)
	return files
}

// scanFiles103 汇总扫描目录全部目标文件（去重），返回 文件名 → 绝对路径。
// 跳过基础设施实现本体（data_cache_service.go 是 DataCacheService.GetOrSet
// 实现处，非调用残留）。
func scanFiles103(t *testing.T) map[string]string {
	t.Helper()
	seen := map[string]string{}
	for _, dir := range scanDirs103(t) {
		for _, path := range goFilesInDir103(t, dir) {
			base := filepath.Base(path)
			if base == "data_cache_service.go" {
				continue // 基础设施实现本体（infrastructure，D-06/D-07 in place）
			}
			seen[base] = path
		}
	}
	return seen
}

// isClosureGetOrSet103 判断 CallExpr 是否为 interface{} 闭包式 GetOrSet 调用
//（口径复用 cacheImplResidue92：selector .GetOrSet + FuncLit 实参签名
// (interface{}, error) / (any, error)）。
func isClosureGetOrSet103(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "GetOrSet" {
		return false
	}
	for _, arg := range call.Args {
		fl, ok := arg.(*ast.FuncLit)
		if !ok {
			continue
		}
		ft := fl.Type
		if ft.Results == nil || len(ft.Results.List) != 2 {
			continue
		}
		if !isEmptyInterface(ft.Results.List[0].Type) {
			continue
		}
		errIdent, ok := ft.Results.List[1].Type.(*ast.Ident)
		if ok && errIdent.Name == "error" {
			return true
		}
	}
	return false
}

// isCacheReadCall103 判断是否为缓存读取调用：receiver 链含 "cache" 字样的
// .Get / .GetJSON 方法调用（如 s.cache.Get / f.cache.GetJSON / l.cache.Get）。
func isCacheReadCall103(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Get" && sel.Sel.Name != "GetJSON" {
		return false
	}
	x, ok := sel.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return x.Sel != nil && x.Sel.Name == "cache"
}

// cacheAsideResidue103 手写 cache-aside 标志性序列检测（D-103-18）：
// 在同一个函数体内，存在位置严格递增的三元组 cache.Get/GetJSON < json.Unmarshal
// < cache.Set/SetJSON——即「读缓存 → 反序列化 → 回源后写缓存」的完整手写形态。
// 用函数域内位置序而非行数窗口（实证校准：selector_learner 原形态 Get@170 与
// Set@227 相隔 57 行，中间夹整个 DB 查询主体，行数窗口必然漏检）。
// 覆盖两种已知手写形态：
//   a) cache.Get → json.Unmarshal → cache.Set(json.Marshal(...))   (selector_learner 原形态)
//   b) cache.GetJSON → … → cache.SetJSON / cache.Set(json.Marshal)  (reconciliation 原形态)
// 返回 "文件名:行号" 列表（行号为 cache.Get 命中行）。
func cacheAsideResidue103(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	var hits []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		var getCalls, unmarshals, setCalls []token.Pos
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if isCacheReadCall103(call) {
				getCalls = append(getCalls, call.Pos())
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				switch sel.Sel.Name {
				case "Unmarshal":
					if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "json" {
						unmarshals = append(unmarshals, call.Pos())
					}
				case "Set", "SetJSON":
					if x, ok := sel.X.(*ast.SelectorExpr); ok && x.Sel != nil && x.Sel.Name == "cache" {
						setCalls = append(setCalls, call.Pos())
					}
				}
			}
			return true
		})
		for _, gp := range getCalls {
			hasUnmarshal := false
			for _, up := range unmarshals {
				if up > gp {
					hasUnmarshal = true
					break
				}
			}
			if !hasUnmarshal {
				continue
			}
			hasSet := false
			for _, sp := range setCalls {
				if sp > gp {
					hasSet = true
					break
				}
			}
			if hasSet {
				hits = append(hits, fmt.Sprintf("%s:%d", filepath.Base(path), fset.Position(gp).Line))
			}
		}
	}
	return hits
}

// TestNoInterfaceGetOrSetResidue103 D-103-17：services root + asset + rpa 全部
// 业务 .go 文件零 interface{} 闭包式 GetOrSet（收敛面硬档，其余文件同硬档——
// 该域从未有登记豁免，出现即倒退）。
func TestNoInterfaceGetOrSetResidue103(t *testing.T) {
	files := scanFiles103(t)
	if len(files) < 10 {
		t.Fatalf("扫描文件数 %d 异常偏小——扫描器口径漂移，检查 scanDirs103", len(files))
	}
	perFile := map[string][]string{}
	for base, path := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s 失败: %v", path, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && isClosureGetOrSet103(call) {
				pos := fset.Position(call.Pos())
				perFile[base] = append(perFile[base], fmt.Sprintf("%s:%d", base, pos.Line))
			}
			return true
		})
	}
	assertResidueBudget103(t, perFile, "interface{} 闭包式 GetOrSet")
}

// TestNoHandwrittenCacheAside D-103-18：手写 cache-aside 标志性序列检测。
// 收敛面 4 文件必须 == 0（硬失败档）；同域其余文件 warning 计数
//（api_endpoint_service / dashboard_service / widget_data_fetcher 等既有站点
// 是 v1.31+ 迁移候选，不在 Phase 103 范围）。
func TestNoHandwrittenCacheAside(t *testing.T) {
	files := scanFiles103(t)
	perFile := map[string][]string{}
	for base, path := range files {
		hits := cacheAsideResidue103(t, path)
		if len(hits) > 0 {
			perFile[base] = append(perFile[base], hits...)
		}
	}

	// 硬失败档：收敛面 4 文件
	for _, migrated := range migratedFiles103 {
		if hits, ok := perFile[migrated]; ok {
			for _, h := range hits {
				t.Errorf("手写 cache-aside 残留：%s（D-103-18 硬档——Phase 103 收敛面不可倒退；确需豁免请显式登记 allowedResidues103）", h)
			}
		}
	}

	// warning 档：其余文件仅计数
	names := make([]string, 0, len(perFile))
	for name := range perFile {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if isMigrated103(name) {
			continue
		}
		t.Logf("[WARNING] %s: %d 处手写 cache-aside（v1.31+ 迁移候选，Phase 103 范围外）", name, len(perFile[name]))
	}
}

// isMigrated103 判断文件是否在 Phase 103 收敛面。
func isMigrated103(base string) bool {
	for _, m := range migratedFiles103 {
		if m == base {
			return true
		}
	}
	return false
}

// assertResidueBudget103 通用预算断言：超出白名单额度即 fail，warning 档仅日志。
func assertResidueBudget103(t *testing.T, perFile map[string][]string, kind string) {
	t.Helper()
	names := make([]string, 0, len(perFile))
	for name := range perFile {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		hits := perFile[name]
		allowed := allowedResidues103[name]
		if len(hits) > allowed {
			for _, h := range hits[allowed:] {
				if isMigrated103(name) {
					t.Errorf("%s 残留：%s（D-103-17 硬档——Phase 103 收敛面已清零不可倒退；确需豁免请显式登记 allowedResidues103）", kind, h)
				} else {
					t.Errorf("%s 残留：%s（D-103-17——services root/asset/rpa 域内新增残留即倒退；确需豁免请显式登记 allowedResidues103）", kind, h)
				}
			}
		} else if len(hits) > 0 {
			for _, h := range hits {
				t.Logf("[ALLOWED] %s 处于白名单豁免额度内", h)
			}
		}
	}
}
