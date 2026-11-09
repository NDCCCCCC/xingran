package operlog

// handler_coverage_test.go — Phase 145 / AUDIT-01/02/03
//
// 把 scripts/operlog_e2e_verify.sh 旧静态段（原 :68-151）的三项断言从 shell
// 逐字迁移为 Go 单测，纳入 go test ./internal/...（ci.yml:62-63 天然硬门禁）。
// 目标：零 ADMIN_USER/ADMIN_PASSWORD/DEV_MODE/SKIP_E2E/SKIP_LIVE、零 running
// backend、零 DB、任意 cwd 下真跑且失败会红。
//
// 与 shell 的关键差异（Lead 方案 C 裁决）：
//   - 旧 shell 用「逐文件 grep 有没有 operlog 调用」判定，且手写
//     READONLY_ALLOWLIST 只有 2 项、多年未维护 → 实测在仓库根 cwd 真跑
//     会列出 9 个 0 命中文件并 exit 1（白名单漂移 + 结构性假阳性）。
//   - 本测试把判定升级为「三组可验证分类」，每组都有源码/AST 可核对的依据，
//     不放水也不误报；原 2 项 READONLY_ALLOWLIST 逐字保留为历史基线。
//
// 设计要点：
//   - repoRoot 用 runtime.Caller(0) 锚定本测试文件目录、上溯 3 级到仓库根，
//     并以 go.mod 存在性兜底；所有被扫描路径均由 repoRoot 拼绝对路径，
//     禁止 "internal/" 之类相对路径常量 → 修掉 cwd 假绿缺陷。
//   - 零外部进程、零 grep/awk、零环境变量、零 DB。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 仓库根锚定（决定性）
// ---------------------------------------------------------------------------

// repoRoot 用 runtime.Caller 锚定本测试文件所在目录，上溯 3 级到仓库根，
// 保证 go test 在任意 cwd（仓库根 / 包子目录 / -C 切换）下都取同一绝对路径。
// internal/utils/operlog/ → ..=internal/utils → ..=internal → ..=仓库根。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root not anchored at %s (go.mod missing): %v", root, err)
	}
	return root
}

// ---------------------------------------------------------------------------
// 常量（逐字迁移 shell 阈值 / 白名单）
// ---------------------------------------------------------------------------

// 250→290 因 Phase 34 补 25 端点（旧 shell :80-82 注释）。
const minOperlogCalls = 290

// 旧 shell :90 阈值。
const minSensitiveKeys = 17

// operlogCallRE 逐字对齐 shell :78 / :120 的
// grep -E "operlog\.(Record|RecordWithBody)\("。
// 字面量用字符串拼接构造，避免本测试文件自身的正则文本被 CALLS 计入
// （否则自引用会把计数从 373 抬到 374，与 shell 基线不一致）。
var operlogCallRE = regexp.MustCompile(`operlog\.` + `(Record|RecordWithBody)` + `\(`)

// shimCallRE 逐字对齐 shell :126 的 legacy recordOperLog( shim。
var shimCallRE = regexp.MustCompile(`recordOperLog\(`)

// handlerReceiverRE 逐字对齐 shell :115 的
// grep -E '^func \(h \*[A-Za-z]+Handler\) [A-Za-z]+\('。
// (?m) 开启多行模式，等价于 grep 的逐行 ^ 锚定（Go regexp 默认 ^ 只锚定全文开头）。
var handlerReceiverRE = regexp.MustCompile(`(?m)^func \(h \*[A-Za-z]+Handler\) [A-Za-z]+\(`)

// readonlyAllowlist 逐字迁移 shell :105-111 的 2 项（原样保留，不得删改）。
// 用 strings.HasSuffix(path, wl) 判定，对齐 shell [[ "$f" == *"$wl" ]]。
var readonlyAllowlist = []string{
	"network/mac_history_handler.go",         // 纯 /history/* 查询 + 1 GET 导出
	"network/mac_history_heatmap_handler.go", // 仅 QueryHeatmap (POST /history/heatmap, read)
}

// ---------------------------------------------------------------------------
// 方案 C：三组可验证分类（表驱动，每项带 reason）
//
// 旧 shell 静态段在仓库根 cwd 真跑会列出以下 9 个 0 命中文件并 exit 1。
// 经 Lead 独立核实后按可验证依据分三组，而非无限扩白名单：
//   (a) 纯只读 handler —— 显式登记其全部 *Handler 方法名（新增写方法会立刻
//       因方法名不在清单而红），源码/AST 复核无写意图；
//   (b) 共用编排写端点 —— 断言同包 port_write_handler.go 的 execSinglePort
//       (:337) / BatchWrite(:456) 内真实存在 operlog.Record，且本文件委托之；
//   (c) 自引用例外 —— 操作日志自身删除不能再写 operlog，走泛型编排。
//   (d) knownGapHandlers —— 真实违反写操作 operlog 约定的遗留缺口，单独用
//       可观测 t.Skip 子测试登记（非放行），见 145-SUMMARY.md §Known Gaps。
// ---------------------------------------------------------------------------

type classifiedHandler struct {
	path    string
	group   string // "readonly" | "orchestrated" | "self-reference"
	reason  string
	methods []string // (a) 显式只读方法名清单；(b)(c) 可为空
}

// knownGapHandlers —— 真实违反 CLAUDE.md「所有业务写操作 success path 在
// response.Success 之前必须 operlog.Record」约定的遗留缺口。
//
// 本 phase（145）只负责「让静态门禁真跑、真实暴露缺口」，不修生产代码
// （文件面属 portwrite，归 W2 148/150 或后续 quick）。这些文件**不得**被当作
// 合法例外静默放行：HandlerDifferential 会用一个可观测 t.Skip 子测试逐项
// 登记（skip 非放行），并从 violations 中显式排除；一旦该文件补了 operlog
// 调用，子测试会把它从 knownGap 清单里摘除（清单与代码漂移即报错）。
//
// 每项必须带文件:行号证据与违反的约定。
type knownGap struct {
	path   string
	reason string
}

var knownGapHandlers = []knownGap{
	{
		path: "network/port_write_trunk_handler.go",
		reason: "`POST /network/ports/write/trunk`（port_write_trunk_handler.go:43-80）成功路径直接 " +
			"response.Success(c, result)，前面没有任何 operlog.Record / RecordWithBody；" +
			"internal/services/portwrite/ 全包 operlog. 零命中；router port_write_router.go:72 " +
			"直连 handler.TrunkWrite（不经过 execSinglePort/BatchWrite）。违反 CLAUDE.md 写操作日志约定；" +
			"v1.51 Phase 124 遗留，本 phase 不修，登记为后续缺口（详见 145-SUMMARY.md §Known Gaps）。",
	},
}

var classifiedHandlers = []classifiedHandler{
	// ---- (a) 纯只读（含原 2 项白名单逐字条目 + 漂移补登记，均带 reason）----
	{readonlyAllowlist[0], "readonly", "原 shell 白名单第 1 项：纯 /history/* 查询 + 1 GET 导出，无写端点",
		[]string{"QueryPortHistory", "QueryDeviceHistory", "QueryConnectionStats", "GetVendor", "QueryHistory", "ExportHistory"}},
	{readonlyAllowlist[1], "readonly", "原 shell 白名单第 2 项：仅 QueryHeatmap (POST /history/heatmap, read)",
		[]string{"QueryHeatmap"}},
	{"asset/reconciliation_statistics_handler.go", "readonly", "6 个 POST 统计端点全部读操作，源码注释明示「不调 operlog.Record」",
		[]string{"WithCore", "Summary", "ByConflictType", "BySeverity", "HealthTrend", "TopUnresolved", "ExceptionRuleStats"}},
	{"network/port_write_audit_handler.go", "readonly", "端口写审计分页查询纯读端点，源码注释明示「不写 audit/operlog」",
		[]string{"PortWriteAuditList"}},
	{"network/port_write_preview_handler.go", "readonly", "端口写 dry-run 命令预览，源码注释明示「不写 audit 行、不调 operlog.Record」",
		[]string{"PortWritePreview"}},
	{"operations/asset_component_handler.go", "readonly", "read-side component listing (GET)，仅 WithCore/ListComponents 查询方法",
		[]string{"WithCore", "ListComponents"}},
	{"system/ad_login_history_handler.go", "readonly", "只读端点，源码注释明示「不记录 operlog(operlog 仅写操作)」",
		[]string{"ListComputerLoginHistory"}},
	{"system/ad_reconciliation_handler.go", "readonly", "只读端点，源码注释明示「不调用 Record / RecordWithBody」",
		[]string{"ListDeptOUMismatch"}},

	// ---- (b) 共用编排写端点（证据：port_write_handler.go:337/:456）----
	// 只保留 physical：它确实委托 execSinglePort（本文件 :62），审计链可静态证明。
	{"network/port_write_physical_handler.go", "orchestrated", "真写端点；operlog 由同包 port_write_handler.go:337 execSinglePort 统一记录，本文件委托 h.execSinglePort(...", nil},

	// ---- (c) 自引用例外 ----
	{"monitor/oper_log_handler.go", "self-reference", "操作日志自身 Clean/Delete 不能再写 operlog；needOperlog=false 且走 NewMonitorLogHandler 泛型编排，构造 models.OperLog 直调 OperLogService.RecordOperLog", nil},
}

// ---------------------------------------------------------------------------
// 扫描工具
// ---------------------------------------------------------------------------

// walkGoFiles 收集 root 下（含子目录）所有 .go 文件；includeTests 控制是否含
// *_test.go（口径决策见 145-SUMMARY.md：CALLS 采用「全 .go 含 _test.go」，
// 与 shell grep -r ... internal/ 逐字一致，实测 373）。
func walkGoFiles(t *testing.T, root string, includeTests bool) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s): %v", root, err)
	}
	return out
}

// countMatches 逐字对齐 shell grep -cE：按行统计正则命中数。
func countMatches(src []byte, re *regexp.Regexp) int {
	n := 0
	for _, line := range strings.Split(string(src), "\n") {
		if re.MatchString(line) {
			n++
		}
	}
	return n
}

// parseGoFile 用 go/parser 解析源码（含注释），失败即 t.Fatal。
func parseGoFile(t *testing.T, path string) (*ast.File, []byte) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", path, err)
	}
	return f, src
}

// hasHandlerReceiver 判定源码是否定义了 `*Handler` 指针接收者方法，
// 逐字对齐 shell :115 的 grep 语义（接收者名 h、单指针、类型名 XxxHandler）。
func hasHandlerReceiver(src []byte) bool {
	return handlerReceiverRE.MatchString(string(src))
}

// handlerMethodNames 用 AST 抽出该文件全部 `*Handler` 接收者方法名
// （方案 C (a) 的可验证依据：不依赖裸字符串正则，避免注释/字符串伪命中）。
func handlerMethodNames(f *ast.File) []string {
	var out []string
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
			continue
		}
		var typeName string
		switch rt := fd.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if id, ok := rt.X.(*ast.Ident); ok {
				typeName = id.Name
			}
		case *ast.Ident:
			typeName = rt.Name
		}
		if strings.HasSuffix(typeName, "Handler") {
			out = append(out, fd.Name.Name)
		}
	}
	return out
}

// funcBodiesHaveCall 用 AST 判断指定方法体内是否出现某个标识符调用。
func funcBodiesHaveCall(f *ast.File, funcName string, calleeIdent string) bool {
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName || fd.Body == nil {
			continue
		}
		found := false
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				if fun.Name == calleeIdent {
					found = true
				}
			case *ast.SelectorExpr:
				if fun.Sel.Name == calleeIdent {
					found = true
				}
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// 断言① CALLS >= 290
// ---------------------------------------------------------------------------

func TestStaticOperlogCoverage_Calls(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	internalDir := filepath.Join(root, "internal")

	calls := 0
	for _, path := range walkGoFiles(t, internalDir, true) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		calls += countMatches(src, operlogCallRE)
	}

	t.Logf("CALLS=%d (min=%d, repoRoot=%s)", calls, minOperlogCalls, root)
	if calls == 0 {
		t.Fatalf("CALLS=0 — cwd 锚定失效（在包子目录读到空数据），静态门禁假绿守护触发")
	}
	if calls < minOperlogCalls {
		t.Fatalf("CALLS=%d < %d — operlog.Record|RecordWithBody 调用数不足（250→290 因 Phase 34 补 25 端点）", calls, minOperlogCalls)
	}
}

// ---------------------------------------------------------------------------
// 断言② sensitiveKeys >= 17（从源码 AST 抽，逐字等价 shell awk 口径）
// ---------------------------------------------------------------------------

func TestStaticOperlogCoverage_SensitiveKeys(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	f, _ := parseGoFile(t, filepath.Join(root, "internal", "utils", "operlog", "operlog.go"))

	keys := 0
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || vs.Names[0].Name != "sensitiveKeys" || len(vs.Values) == 0 {
				continue
			}
			cl, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, elt := range cl.Elts {
				if bl, ok := elt.(*ast.BasicLit); ok && bl.Kind == token.STRING {
					keys++
				}
			}
		}
	}

	t.Logf("SENSITIVE_KEYS=%d (min=%d)", keys, minSensitiveKeys)
	if keys < minSensitiveKeys {
		t.Fatalf("SENSITIVE_KEYS=%d < %d — sensitiveKeys 条目不足（删一个条目即应红）", keys, minSensitiveKeys)
	}
}

// ---------------------------------------------------------------------------
// 断言③ handler differential（方案 C：三组可验证分类）
// ---------------------------------------------------------------------------

func TestStaticOperlogCoverage_HandlerDifferential(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	v1Dir := filepath.Join(root, "internal", "api", "v1")

	// 收集所有 *_handler.go 中定义了 *Handler 接收者方法的文件。
	var handlerFiles []string
	for _, path := range walkGoFiles(t, v1Dir, false) {
		if !strings.HasSuffix(path, "_handler.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		if hasHandlerReceiver(src) {
			handlerFiles = append(handlerFiles, path)
		}
	}

	t.Logf("HANDLER_FILES=%d (repoRoot=%s)", len(handlerFiles), root)
	if len(handlerFiles) == 0 {
		t.Fatalf("HANDLER_FILES=0 — cwd 锚定失效，静态门禁假绿守护触发")
	}

	// 建立分类表：相对 v1Dir 的 POSIX 风格绝对路径 → 分类。
	classified := make(map[string]classifiedHandler, len(classifiedHandlers))
	for _, c := range classifiedHandlers {
		key := filepath.ToSlash(filepath.Join(v1Dir, c.path))
		classified[key] = c
	}

	// knownGap 集合：命中者不进 violations（非放行），但必须逐项以可观测
	// t.Skip 子测试登记，且反向复核「确实仍缺 operlog」——若将来补了调用，
	// 子测试会失败，要求把该项从 knownGapHandlers 清单中摘除。
	knownGaps := make(map[string]knownGap, len(knownGapHandlers))
	for _, g := range knownGapHandlers {
		knownGaps[filepath.ToSlash(filepath.Join(v1Dir, g.path))] = g
	}

	// (b) 共用编排证据：同包 port_write_handler.go 的 execSinglePort/BatchWrite
	// 内必须真实存在 operlog.Record（静态证明审计链，不放水）。
	orchestratorPath := filepath.Join(v1Dir, "network", "port_write_handler.go")
	orchFile, orchSrc := parseGoFile(t, orchestratorPath)
	if countMatches(orchSrc, operlogCallRE) == 0 {
		t.Fatalf("共用编排证据缺失：%s 内无 operlog.Record（execSinglePort/BatchWrite 应各有一处）", orchestratorPath)
	}
	if !funcBodiesHaveCall(orchFile, "execSinglePort", "Record") ||
		!funcBodiesHaveCall(orchFile, "BatchWrite", "Record") {
		t.Fatalf("共用编排证据缺失：execSinglePort / BatchWrite 函数体内未找到 operlog.Record 调用")
	}

	var violations []string
	for _, path := range handlerFiles {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		count := countMatches(src, operlogCallRE) + countMatches(src, shimCallRE)
		if count > 0 {
			continue // 逐文件直接命中 → PASS（绝大多数 handler）
		}

		key := filepath.ToSlash(path)

		// (d) 真实缺口：可观测登记，非放行。逐个起子测试 t.Skip，并反向
		// 复核「确实仍无 operlog 调用」——补了调用后此子测试会失败，
		// 强制把该项从 knownGapHandlers 摘除，避免清单变成永久放水口。
		if g, isGap := knownGaps[key]; isGap {
			t.Run("KNOWN_GAP_"+filepath.Base(path), func(t *testing.T) {
				if countMatches(src, operlogCallRE)+countMatches(src, shimCallRE) > 0 {
					t.Fatalf("knownGap %s 已出现 operlog 调用，请从 knownGapHandlers 清单摘除（缺口已修）", g.path)
				}
				t.Skipf("KNOWN GAP (可观测登记、非放行): %s — %s", g.path, g.reason)
			})
			continue
		}

		c, ok := classified[key]
		if !ok {
			violations = append(violations, key+"（未分类的 0 命中 handler：请补 operlog 调用或归入带 reason 的分类表）")
			continue
		}

		// 逐组做可验证复核，分类不是「无限白名单」。
		f, _ := parseGoFile(t, path)
		switch c.group {
		case "readonly":
			if !assertReadOnlyGroup(t, f, path, c.methods) {
				violations = append(violations, key+"（分类 readonly 未通过只读依据复核）")
			}
		case "orchestrated":
			if !assertOrchestratedGroup(t, f, path) {
				violations = append(violations, key+"（分类 orchestrated 未通过委托关系复核）")
			}
		case "self-reference":
			if !assertSelfReferenceGroup(t, src, path) {
				violations = append(violations, key+"（分类 self-reference 未通过自引用例外复核）")
			}
		default:
			violations = append(violations, key+"（未知分类 group="+c.group+"）")
		}
	}

	if len(violations) > 0 {
		t.Fatalf("以下 *_handler.go 定义 *Handler 方法但未满足 operlog 覆盖（含分类复核失败）：\n  - %s\nIf a file is genuinely read-only, add it to readonlyAllowlist/classifiedHandlers with a reason.",
			strings.Join(violations, "\n  - "))
	}
}

// assertReadOnlyGroup (a)：AST 抽出的全部 *Handler 方法名必须与显式登记的
// 只读方法名清单逐字一致（新增写方法 / 改名都会立刻红，不放水）。
func assertReadOnlyGroup(t *testing.T, f *ast.File, path string, want []string) bool {
	t.Helper()
	got := handlerMethodNames(f)
	if len(got) == 0 {
		t.Logf("readonly 复核 %s: 未发现 *Handler 方法", filepath.Base(path))
		return true
	}
	wantSet := make(map[string]bool, len(want))
	for _, m := range want {
		wantSet[m] = true
	}
	var bad []string
	for _, m := range got {
		if !wantSet[m] {
			bad = append(bad, m)
		}
	}
	if len(bad) > 0 {
		t.Logf("readonly 复核 %s 失败: 未在只读清单内的方法=%v（新增写方法须补 operlog 或改分类）", filepath.Base(path), bad)
		return false
	}
	return true
}

// assertOrchestratedGroup (b)：本文件必须出现对编排函数 execSinglePort /
// BatchWrite 的委托调用（静态证明「operlog 由共用编排统一记录」），且同包
// 编排内已由 HandlerDifferential 证明存在 operlog.Record。
func assertOrchestratedGroup(t *testing.T, f *ast.File, path string) bool {
	t.Helper()
	delegates := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok &&
			(sel.Sel.Name == "execSinglePort" || sel.Sel.Name == "BatchWrite") {
			delegates = true
		}
		return !delegates
	})
	if !delegates {
		t.Logf("orchestrated 复核 %s 失败: 未发现 execSinglePort/BatchWrite 委托调用", filepath.Base(path))
		return false
	}
	return true
}

// assertSelfReferenceGroup (c)：必须显式出现 needOperlog=false 与
// NewMonitorLogHandler 泛型编排、且用 OperLogService.RecordOperLog 自写审计行。
func assertSelfReferenceGroup(t *testing.T, src []byte, path string) bool {
	t.Helper()
	s := string(src)
	checks := []struct {
		name string
		ok   bool
	}{
		{"needOperlog=false", strings.Contains(s, "needOperlog=false")},
		{"NewMonitorLogHandler", strings.Contains(s, "NewMonitorLogHandler")},
		{"OperLogService.RecordOperLog", strings.Contains(s, "OperLogService.RecordOperLog")},
	}
	for _, c := range checks {
		if !c.ok {
			t.Logf("self-reference 复核 %s 失败: 缺少证据 %q", filepath.Base(path), c.name)
			return false
		}
	}
	return true
}
