package system

// =====================================================================
// Phase 92-04 Task 2: cache_impl 无 interface{} 闭包式 GetOrSet 残留锁（D-10②）。
//
// 与 pkg/constants/pagination_test.go（Phase 89 AST 锁值先例）同模式：AST 扫描 +
// 白名单豁免机制。Phase 92 已把 system/operations 的 *_cache_impl.go 全部收敛
// base.GetOrSetJSON[T] 泛型函数族（CACHE-UNIFY-02/03），本测试使该成果不可无声
// 倒退——新豁免必须显式登记 allowedResidues（diff 可见，T-92-01 缓解）。
//
// 断言双档（D-10② 与 Phase 89/90 硬锁惯例的调和）：
//   - 硬档：system + operations 残留必须 == allowedResidues（初始 0），超出即 fail
//   - warning 档：duty/knowledge/network/workorder 的同构残留仅计数日志不 fail
//     （A5 范围外站点，v1.30+ 迁移候选）
//
// 纪律：文件定位用 runtime.Caller 相对推导，禁本地绝对路径断言（v1.27 测试纪律）。
// =====================================================================

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// allowedResidues 白名单：文件名 → 允许的 interface{} 闭包式 GetOrSet 残留数。
// 初始全空 = 期望 0；未来确需豁免必须在此显式登记并附说明（diff 可见）。
var allowedResidues = map[string]int{}

// cacheImplResidue92 解析单个 Go 文件，返回其中 interface{} 闭包式 GetOrSet
// 调用的 "文件名:行号" 列表。
//
// 检测口径（AST 精确匹配）：CallExpr 的 selector 为 .GetOrSet 且任一实参为
// FuncLit、其签名为二元返回 (interface{}, error)——即 CacheProvider/DataCacheService
// 读穿透样板的闭包实参（Phase 92 D-03 消灭对象）。`any` 为 interface{} 的别名，
// 同样计入。实现端不误伤：adapter/provider 自身实现 GetOrSet 方法时不含闭包
// 实参（system/adapter.go:32 先例，92-02 校准记录 2）。
func cacheImplResidue92(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	var hits []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "GetOrSet" {
			return true
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
				pos := fset.Position(call.Pos())
				hits = append(hits, fmt.Sprintf("%s:%d", filepath.Base(pos.Filename), pos.Line))
			}
		}
		return true
	})
	return hits
}

// isEmptyInterface 判断类型表达式是否为空接口（interface{} / any）。
func isEmptyInterface(expr ast.Expr) bool {
	switch v := expr.(type) {
	case *ast.InterfaceType:
		// interface{} 在 go/parser 下 Methods 可能为非 nil 空 FieldList，双口径判定
		return v.Methods == nil || len(v.Methods.List) == 0
	case *ast.Ident:
		return v.Name == "any"
	}
	return false
}

// cacheImplFiles92 返回目录下 *_cache_impl.go 与 cache_impl.go 两个口径的并集
//（network 的文件名为 cache_impl.go，不匹配 *_cache_impl.go 前缀通配）。
func cacheImplFiles92(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	for _, pattern := range []string{"*_cache_impl.go", "cache_impl.go"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			t.Fatalf("glob %s 失败: %v", pattern, err)
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	// 去重（cache_impl.go 同时命中 *_cache_impl.go 的 * 为空情形不存在，
	// 但防御性去重保持扫描口径稳定）
	deduped := files[:0]
	var prev string
	for _, f := range files {
		if f != prev {
			deduped = append(deduped, f)
		}
		prev = f
	}
	return deduped
}

// servicesRoot92 以本测试文件位置推导 internal/services 根（相对路径，禁绝对路径断言）。
func servicesRoot92(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败——无法定位测试文件位置")
	}
	// thisFile = .../internal/services/system/cache_invariants_92_test.go
	return filepath.Dir(filepath.Dir(thisFile))
}

// TestNoInterfaceGetOrSetResidue 锁定 Phase 92 定性底线：system/operations 的
// cache_impl 文件零 interface{} 闭包式 GetOrSet 残留（硬档 == 白名单），外围
// duty/knowledge/network/workorder 同构残留仅 warning 计数（A5 范围外）。
func TestNoInterfaceGetOrSetResidue(t *testing.T) {
	servicesRoot := servicesRoot92(t)

	// 硬档目录：Phase 92 已迁移面（CACHE-UNIFY-02 system 9 文件 + CACHE-UNIFY-03 floor）
	hardDirs := []string{
		filepath.Join(servicesRoot, "system"),
		filepath.Join(servicesRoot, "operations"),
	}
	// warning 档目录：A5 范围外站点（v1.30+ 迁移候选，D-10② 不 fail）
	warnDirs := []string{
		filepath.Join(servicesRoot, "duty"),
		filepath.Join(servicesRoot, "knowledge"),
		filepath.Join(servicesRoot, "network"),
		filepath.Join(servicesRoot, "workorder"),
	}

	// ---- 硬档：残留按文件计数，超出白名单部分即 fail ----
	perFile := map[string][]string{}
	for _, dir := range hardDirs {
		files := cacheImplFiles92(t, dir)
		if len(files) == 0 {
			t.Errorf("目录 %s 未匹配到任何 cache_impl 文件——扫描器空转，检查路径口径", dir)
			continue
		}
		for _, path := range files {
			hits := cacheImplResidue92(t, path)
			if len(hits) > 0 {
				perFile[filepath.Base(path)] = append(perFile[filepath.Base(path)], hits...)
			}
		}
	}

	names := make([]string, 0, len(perFile))
	for name := range perFile {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		hits := perFile[name]
		allowed := allowedResidues[name]
		if len(hits) > allowed {
			for _, h := range hits[allowed:] {
				t.Errorf("interface{} 闭包式 GetOrSet 残留：%s（D-10② 硬档——Phase 92 已清零面不可倒退；确需豁免请显式登记 allowedResidues）", h)
			}
		} else if len(hits) > 0 {
			for _, h := range hits {
				t.Logf("[ALLOWED] %s 处于白名单豁免额度内", h)
			}
		}
	}

	// 白名单卫生：登记了额度却零残留的条目提示清理（不 fail，保持映射即诚实口径）
	for name, allowed := range allowedResidues {
		if _, ok := perFile[name]; !ok && allowed > 0 {
			t.Logf("[ALLOWED] 白名单条目 %s=%d 当前无对应残留（可清理）", name, allowed)
		}
	}

	// ---- warning 档：A5 范围外站点，仅计数日志不 fail ----
	for _, dir := range warnDirs {
		files := cacheImplFiles92(t, dir)
		total := 0
		for _, path := range files {
			total += len(cacheImplResidue92(t, path))
		}
		t.Logf("[WARNING] %s: %d 处 interface{} 闭包式 GetOrSet（v1.30+ 候选，A5 范围外）",
			strings.ToUpper(filepath.Base(dir)), total)
	}
}
