package executor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ============================================================================
// Phase 154 / ENGINE-02 验收：types.go 每个 ActionType 常量都必须在
// engine.go 的 executeAction switch 里有对应 case。
//
// 做法：AST 扫描（不依赖反射，也不依赖「常量名 == 字符串」的巧合）。
//  1. 扫 internal/types/types.go，收集所有声明为 types.ActionType 的常量；
//  2. 扫 internal/executor/engine.go，定位 executeAction 内 `switch action.Type`
//     语句，收集所有 `case types.ActionXxx:` 的标识符；
//  3. 差集非空即红 —— 新增动作但忘了落 case 会在提交时被拦截。
//
// 反向也断言：switch 里不得引用 types 包里不存在的动作常量（防拼写错误）；
// 并断言 switch 有 default 分支（未支持动作必须走可读错误而非静默）。
// ============================================================================

// actionConstsIn 从 types.go 解析出所有 ActionType 常量名。
func actionConstsIn(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(dir, "types.go"), nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("解析 types.go 失败: %v", err)
	}

	var names []string
	ast.Inspect(f, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// 只在显式标注 ActionType 的常量上收集（避免误把
			// StatusRunning 一类的 other 类型常量算进来）。
			if ident, ok := vs.Type.(*ast.Ident); !ok || ident.Name != "ActionType" {
				continue
			}
			for _, name := range vs.Names {
				names = append(names, name.Name)
			}
		}
		return true
	})
	sort.Strings(names)
	return names
}

// switchCasesIn 从 engine.go 的 executeAction 提取 `case types.Xxx:` 名单。
func switchCasesIn(t *testing.T, dir string) (cases []string, hasDefault bool) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(dir, "engine.go"), nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("解析 engine.go 失败: %v", err)
	}

	var found bool
	ast.Inspect(f, func(n ast.Node) bool {
		decl, ok := n.(*ast.FuncDecl)
		if !ok || decl.Name.Name != "executeAction" {
			return true
		}
		ast.Inspect(decl.Body, func(inner ast.Node) bool {
			stmt, ok := inner.(*ast.SwitchStmt)
			if !ok {
				return true
			}
			sel, ok := stmt.Tag.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Type" {
				return true
			}
			found = true
			for _, cs := range stmt.Body.List {
				cl, ok := cs.(*ast.CaseClause)
				if !ok {
					continue
				}
				if cl.List == nil {
					hasDefault = true
					continue
				}
				for _, expr := range cl.List {
					if se, ok := expr.(*ast.SelectorExpr); ok {
						cases = append(cases, se.Sel.Name)
					}
				}
			}
			return false
		})
		return !found
	})
	if !found {
		t.Fatal("engine.go 中找不到 executeAction 的 `switch action.Type`")
	}
	sort.Strings(cases)
	return cases, hasDefault
}

// TestActionTypeSwitchCoverage ENGINE-02 主断言：枚举与 switch 一一对应。
func TestActionTypeSwitchCoverage(t *testing.T) {
	typesDir := filepath.Join("..", "types")
	execDir := "."

	// 与测试文件自身同包目录（go test 的 cwd 是包目录）。
	if _, err := os.Stat(filepath.Join(typesDir, "types.go")); err != nil {
		wd, _ := os.Getwd()
		t.Fatalf("找不到 types.go（cwd=%s）: %v", wd, err)
	}

	consts := actionConstsIn(t, typesDir)
	if len(consts) == 0 {
		t.Fatal("types.go 未解析出任何 ActionType 常量（AST 匹配规则可能失效）")
	}

	cases, hasDefault := switchCasesIn(t, execDir)
	if len(cases) == 0 {
		t.Fatal("engine.go 的 executeAction switch 未解析出任何 case")
	}
	if !hasDefault {
		t.Fatal("executeAction switch 必须有 default 分支（未支持动作要给可读错误）")
	}

	have := make(map[string]bool, len(cases))
	for _, c := range cases {
		have[c] = true
	}
	want := make(map[string]bool, len(consts))
	for _, c := range consts {
		want[c] = true
	}

	var missing []string
	for _, c := range consts {
		if !have[c] {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("types.go 声明但 engine.go switch 无 case 的动作 %d 个: %s\n"+
			"（ENGINE-02 要求动作枚举与执行分支同一次 commit 落地）",
			len(missing), strings.Join(missing, ", "))
	}

	var unknown []string
	for _, c := range cases {
		if !want[c] {
			unknown = append(unknown, c)
		}
	}
	if len(unknown) > 0 {
		t.Fatalf("engine.go switch 引用了 types.go 未声明的动作常量: %s",
			strings.Join(unknown, ", "))
	}

	t.Logf("动作覆盖一致：%d 个 ActionType 常量全部有 switch case", len(consts))
}

// TestActionTypeConstantsAreUnique ENGINE-02 附加断言：常量字符串值不重复
// （两个常量映射同一字符串会让 switch 出现不可达 case）。
func TestActionTypeConstantsAreUnique(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join("..", "types", "types.go"), nil, 0)
	if err != nil {
		t.Fatalf("解析 types.go 失败: %v", err)
	}

	type entry struct{ name, value string }
	var entries []entry
	ast.Inspect(f, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
				continue
			}
			ident, ok := vs.Type.(*ast.Ident)
			if !ok || ident.Name != "ActionType" {
				continue
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			entries = append(entries, entry{name: vs.Names[0].Name, value: strings.Trim(lit.Value, `"`)})
		}
		return true
	})

	seen := map[string]string{}
	for _, e := range entries {
		if prev, dup := seen[e.value]; dup {
			t.Fatalf("ActionType 字符串值重复: %s 与 %s 都是 %q", prev, e.name, e.value)
		}
		seen[e.value] = e.name
	}
	t.Logf("动作字符串值唯一性通过：%d 个", len(entries))
}
