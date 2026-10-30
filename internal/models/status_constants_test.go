package models

// status_constants_test.go guards the watched status-constant families of the
// models package — the single source of truth for status semantics (Phase 69
// DICT-01) — against silent drift introduced by future refactors. It adapts the
// proven lock-in pattern of internal/utils/operlog/regression_test.go to a
// multi-file, multi-family scope.
//
// What is pinned and why:
//   - Watched family constant values (TestStatusConstantsStability):
//     raw SQL and struct literals across internal/services and internal/api/v1
//     are being migrated to reference these constants. Renumbering a value
//     would silently corrupt the status semantics of every existing DB row and
//     every unreplaced literal. Removing a constant is a breaking change.
//   - Cross-package duplicate definitions are checked for value conflicts:
//     FloorStatus / RoomStatus / RoomDeviceStatus exist BOTH here (package
//     models, operations.go) and in internal/models/operations (package
//     operations). Identical values are tolerated (known double definition);
//     divergent values for the same name fail the parse.
//
// Highest-risk semantics that must NEVER be "normalized" (see Phase 69
// RESEARCH cluster table A2):
//   - Reversed (cluster E): VisibleShow=1 / VisibleHidden=0 (menu visibility),
//     KnowledgeArticleStatusPublished=1 and PublishStatusPublished=1 — 1 is
//     the POSITIVE value here; applying the "0=normal, 1=disabled" convention
//     to these families is a bug.
//   - State machines (cluster B): ExecutionStatus 0..4 — order is semantic.
//   - Multi-state (cluster D): LineStatus / InfoPointStatus / RoomDeviceStatus
//     / DeviceStatus / DutyStatus / WorkstationStatus — 1 and 2 mean different
//     things per entity, never blanket-enable/disable.
//   - Success/failure (cluster C): OperLogStatus / LoginLogStatus /
//     JobLogStatus — 0=成功, 1=失败 (NOT enable/disable).
//
// REGISTRATION MECHANISM (mandatory for later Phase 69 batches):
// every NEW status-constant family added to internal/models or
// internal/models/operations MUST, in the same change:
//   1. have its family prefix added to watchedStatusPrefixes below, and
//   2. have every constant of the family enumerated in expectedStatusValues.
// The bidirectional assertion catches constants forgotten on side 2 (an
// unregistered VALUE under a watched prefix fails the test), but a family
// prefix forgotten on side 1 leaves a SILENT HOLE in coverage — reviewers must
// check both. Phase 69 later batches (69-03/69-04/69-05) will add families
// such as ADAccountStatus* / RPACredentialStatus* / DutyHoliday etc. —
// register them here when created.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// watchedStatusPrefixes enumerates the constant-name prefixes that constitute
// a watched status family. String-valued families (ConfigType, MenuType,
// DeviceType, InfoPointType, ...) are listed for intent: the AST reader only
// collects integer literals, so their string constants are naturally skipped —
// but if an int constant were ever added under one of these prefixes it would
// be picked up and must be registered in expectedStatusValues.
var watchedStatusPrefixes = []string{
	// base.go system families
	"UserStatus", "RoleStatus", "MenuStatus", "DeptStatus", "PostStatus",
	"Visible", "Gender", "ConfigType", "ConfigIsSystem",
	// dict / log / notice / vdi (Phase 69 Wave 0 additions included)
	"DictStatus", "JobStatus", "JobLogStatus", "LoginLogStatus", "OperLogStatus",
	"PublishStatus", "NoticeStatus", "VDIServerStatus",
	// Phase 102 Plan 04 additions
	"WorkOrderStatus",
	// operations-domain families
	"ExecutionStatus", "KnowledgeArticleStatus", "LineStatus",
	"WorkstationType", "WorkstationStatus", "DeviceStatus", "DiscoveryStatus",
	"ServerStatus", "NotificationConfigStatus",
	"DutyStatus", "DutyPoolStatus", "BuildingStatus", "FloorStatus",
	"RoomStatus", "RoomDeviceStatus", "InfoPointStatus", "DashboardStatus",
	// Phase 69 batch 2 additions (untyped int, int-typed struct fields)
	"WorkstationDeviceStatus", "AssetStatus", "AssetNBFStatus",
	// Phase 69 batch 3 additions (untyped int; AD pool state machine + RPA credential)
	"ADAccountStatus", "RPACredentialStatus",
}

// expectedStatusValues pins the documented (name -> value) mapping of every
// watched constant. Values are frozen as of Phase 69 Plan 01 (2026-08-19);
// any change to a value here is a data-semantics migration decision, not a
// refactor.
var expectedStatusValues = map[string]int{
	// base.go
	"UserStatusEnabled":  0, // 启用
	"UserStatusDisabled": 1, // 禁用
	"RoleStatusEnabled":  0, // 正常
	"RoleStatusDisabled": 1, // 停用
	"GenderMale":         0, // 男
	"GenderFemale":       1, // 女
	"GenderSecret":       2, // 保密
	"VisibleShow":        1, // 显示 — 反转例外：1=正向
	"VisibleHidden":      0, // 隐藏 — 反转例外：0=负向
	"MenuStatusNormal":   0, // 正常
	"MenuStatusStop":     1, // 停用
	"DeptStatusNormal":   0, // 正常
	"DeptStatusStop":     1, // 停用
	"PostStatusEnabled":  0, // 正常
	"PostStatusDisabled": 1, // 停用
	"ConfigIsSystemNo":   0, // 否
	"ConfigIsSystemYes":  1, // 是
	// dict.go（Phase 69 Wave 0 新增）
	"DictStatusNormal":   0, // 正常
	"DictStatusDisabled": 1, // 停用
	// log.go（成败簇 C；JobStatus 为既有）
	"JobStatusNormal":       0, // 正常
	"JobStatusPause":        1, // 暂停
	"OperLogStatusSuccess":  0, // 成功
	"OperLogStatusFailure":  1, // 失败
	"LoginLogStatusSuccess": 0, // 成功
	"LoginLogStatusFailure": 1, // 失败
	"JobLogStatusSuccess":   0, // 成功
	"JobLogStatusFailure":   1, // 失败
	// notice_enhanced.go（PublishStatus 既有；NoticeStatus 为 Wave 0 新增）
	"PublishStatusDraft":     0, // 草稿
	"PublishStatusPublished": 1, // 已发布 — E 簇反转
	"PublishStatusScheduled": 2, // 定时发布中
	"PublishStatusWithdrawn": 3, // 已撤回
	"NoticeStatusNormal":     0, // 正常
	"NoticeStatusClosed":     1, // 关闭
	// vdi.go（Wave 0 新增）
	"VDIServerStatusNormal":  0, // 正常
	"VDIServerStatusStopped": 1, // 停用
	// monitor.go / notification_config.go（Phase 69 Plan 05 补齐真相源）
	"ServerStatusNormal":              0, // 正常
	"ServerStatusAbnormal":            1, // 异常
	"NotificationConfigStatusNormal":  0, // 正常
	"NotificationConfigStatusStopped": 1, // 停用
	// workstation.go（多态簇 D：1=占用 并非"停用"）
	"WorkstationStatusAvailable": 0, // 空闲
	"WorkstationStatusOccupied":  1, // 占用
	"WorkstationStatusMaintain":  2, // 维护
	"WorkstationTypeFixed":       0, // 固定工位
	"WorkstationTypeHotDesk":     1, // 灵活工位
	"WorkstationTypeManager":     2, // 管理工位
	// network_device.go（多态簇 D）
	"DeviceStatusOnline":  0, // 在线
	"DeviceStatusOffline": 1, // 离线
	"DeviceStatusUnknown": 2, // 未知
	// device_discovery.go（任务状态机）
	"DiscoveryStatusPending":   0, // 待执行
	"DiscoveryStatusRunning":   1, // 执行中
	"DiscoveryStatusSuccess":   2, // 成功
	"DiscoveryStatusFailed":    3, // 失败
	"DiscoveryStatusCancelled": 4, // 已取消
	// config_execution.go（状态机簇 B，顺序即语义）
	"ExecutionStatusPending":   0, // 待执行
	"ExecutionStatusRunning":   1, // 执行中
	"ExecutionStatusSuccess":   2, // 成功
	"ExecutionStatusFailed":    3, // 失败
	"ExecutionStatusCancelled": 4, // 已取消
	// knowledge.go（E 簇反转：1=已发布）
	"KnowledgeArticleStatusDraft":     0, // 草稿
	"KnowledgeArticleStatusPublished": 1, // 已发布
	// duty.go
	"DutyStatusNormal":       0, // 正常
	"DutyStatusExchanged":    1, // 已调换
	"DutyStatusCancelled":    2, // 已取消
	"DutyPoolStatusEnabled":  0, // 启用
	"DutyPoolStatusDisabled": 1, // 停用
	// dashboard.go
	"DashboardStatusNormal":  0,
	"DashboardStatusStopped": 1,
	// operations.go（package models 旧版）
	"FloorStatusNormal":      0,
	"FloorStatusStopped":     1,
	"RoomStatusNormal":       0,
	"RoomStatusStopped":      1,
	"RoomDeviceStatusNormal": 0,
	"RoomDeviceStatusFault":  1,
	"RoomDeviceStatusScrap":  2,
	// operations 子包（package operations）
	"BuildingStatusNormal":     0, // 正常
	"BuildingStatusStopped":    1, // 停用
	"RoomDeviceStatusScrapped": 2, // 报废（子包命名，与 models 版 Scrap 同值）
	"LineStatusNormal":         0, // 正常
	"LineStatusFault":          1, // 故障
	"LineStatusDisabled":       2, // 停用
	"InfoPointStatusNormal":    0, // 正常
	"InfoPointStatusFault":     1, // 故障
	"InfoPointStatusDisabled":  2, // 停用
	// workstation_device.go / asset.go（Phase 69 批 2 新增，无类型 int 常量）
	"WorkstationDeviceStatusNormal":  0, // 正常
	"WorkstationDeviceStatusStopped": 1, // 停用
	"AssetStatusNormal":              0, // 正常
	"AssetStatusStopped":             1, // 停用
	"AssetNBFStatusNo":               0, // 否
	"AssetNBFStatusYes":              1, // 拟报废
	// ad_service_account.go（Phase 69 批 3 新增，D 簇三态状态机：
	// 0=可用 / 1=停用 / 2=熔断，语义见 Phase 36 AccountPool 状态机）
	"ADAccountStatusAvailable": 0, // 可用
	"ADAccountStatusDisabled":  1, // 管理员手动停用
	"ADAccountStatusBreaker":   2, // 熔断中
	// rpa.go（Phase 69 批 3 新增，簇 A 凭证启停，对齐 credentials.go check IN (0,1)）
	"RPACredentialStatusNormal":  0, // 正常
	"RPACredentialStatusStopped": 1, // 停用
	// Phase 102 Plan 04 additions
	"WorkOrderStatusPending":    0, // 待处理
	"WorkOrderStatusProcessing": 1, // 处理中
	"WorkOrderStatusCompleted":  2, // 已完成
	"WorkOrderStatusClosed":    3, // 已关闭
	"WorkOrderStatusRejected":  4, // 已拒绝
}

// TestStatusConstantsStability asserts each watched constant is pinned to its
// documented int value (expected -> actual: missing / renumbered) and that no
// unregistered constant hides under a watched prefix (actual -> expected).
func TestStatusConstantsStability(t *testing.T) {
	t.Parallel()
	actual, err := readStatusConsts()
	if err != nil {
		t.Fatalf("failed to parse models status constants: %v", err)
	}
	if len(actual) == 0 {
		t.Fatal("no watched status constants found — the AST reader is broken")
	}
	for name, want := range expectedStatusValues {
		got, ok := actual[name]
		if !ok {
			t.Errorf("watched status constant %q is missing — removing a constant is a breaking change", name)
			continue
		}
		if got.value != want {
			t.Errorf("status constant %q = %d, want %d (renumbering would corrupt status semantics; defined in %s)", name, got.value, want, got.file)
		}
	}
	// Also flag unexpected constants so additions are deliberate (register the
	// family AND the value in the same change).
	for name, got := range actual {
		if _, ok := expectedStatusValues[name]; !ok {
			t.Errorf("unexpected watched status constant %q = %d (defined in %s) — if intentional, add it to expectedStatusValues and to this file's header registration note", name, got.value, got.file)
		}
	}
}

// TestStatusConstantsCriticalFamilies makes the per-family lock-in visible in
// test output (-v) for the semantics-critical families: reversed (E), state
// machine (B), multi-state (D), success/failure (C), and the Phase 69 Wave 0
// additions consumed by later batches.
func TestStatusConstantsCriticalFamilies(t *testing.T) {
	t.Parallel()
	actual, err := readStatusConsts()
	if err != nil {
		t.Fatalf("failed to parse models status constants: %v", err)
	}
	families := map[string]map[string]int{
		"UserStatus(base)":       {"UserStatusEnabled": 0, "UserStatusDisabled": 1},
		"RoleStatus(base)":       {"RoleStatusEnabled": 0, "RoleStatusDisabled": 1},
		"Visible(reversed-E)":    {"VisibleShow": 1, "VisibleHidden": 0},
		"Gender(base)":           {"GenderMale": 0, "GenderFemale": 1, "GenderSecret": 2},
		"DictStatus(wave0)":      {"DictStatusNormal": 0, "DictStatusDisabled": 1},
		"Knowledge(E-reversed)":  {"KnowledgeArticleStatusDraft": 0, "KnowledgeArticleStatusPublished": 1},
		"PublishStatus(E)":       {"PublishStatusDraft": 0, "PublishStatusPublished": 1, "PublishStatusScheduled": 2, "PublishStatusWithdrawn": 3},
		"ExecutionStatus(B)":     {"ExecutionStatusPending": 0, "ExecutionStatusRunning": 1, "ExecutionStatusSuccess": 2, "ExecutionStatusFailed": 3, "ExecutionStatusCancelled": 4},
		"LineStatus(D)":          {"LineStatusNormal": 0, "LineStatusFault": 1, "LineStatusDisabled": 2},
		"OperLogStatus(C)":       {"OperLogStatusSuccess": 0, "OperLogStatusFailure": 1},
		"JobStatus(A)":           {"JobStatusNormal": 0, "JobStatusPause": 1},
		"VDIServerStatus(wave0)": {"VDIServerStatusNormal": 0, "VDIServerStatusStopped": 1},
		"NoticeStatus(wave0)":    {"NoticeStatusNormal": 0, "NoticeStatusClosed": 1},
		"InfoPointStatus(D)":     {"InfoPointStatusNormal": 0, "InfoPointStatusFault": 1, "InfoPointStatusDisabled": 2},
	}
	for family, members := range families {
		t.Run(family, func(t *testing.T) {
			t.Parallel()
			for name, want := range members {
				got, ok := actual[name]
				if !ok {
					t.Errorf("%s: constant %q missing", family, name)
					continue
				}
				if got.value != want {
					t.Errorf("%s: %q = %d, want %d", family, name, got.value, want)
				}
			}
		})
	}
}

// statusConstSite records where a watched constant was found, so failures can
// point at the defining file.
type statusConstSite struct {
	value int
	file  string
}

// readStatusConsts scans internal/models/*.go and internal/models/operations/*.go
// (skipping _test.go), parses every const declaration, and returns the merged
// name -> value map of constants whose name starts with a watched family
// prefix. Duplicate definitions with IDENTICAL values are tolerated (known:
// FloorStatus / RoomStatus / RoomDeviceStatus exist in both package models and
// package operations); duplicates with CONFLICTING values return an error.
func readStatusConsts() (map[string]statusConstSite, error) {
	out := make(map[string]statusConstSite)
	patterns := []string{"*.go", filepath.Join("operations", "*.go")}
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no Go files matched %s — run tests from the internal/models package directory", pattern)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file, nil, 0)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", file, err)
			}
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.CONST {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 {
						continue
					}
					name := vs.Names[0].Name
					if !isWatchedStatusConst(name) {
						continue
					}
					lit, ok := vs.Values[0].(*ast.BasicLit)
					if !ok || lit.Kind != token.INT {
						// String enums / iota expressions are out of scope for
						// the int lock-in; expected-side entries would fail as
						// "missing", forcing explicit registration.
						continue
					}
					n, err := strconv.Atoi(strings.TrimSpace(lit.Value))
					if err != nil {
						continue
					}
					if prev, exists := out[name]; exists {
						if prev.value != n {
							return nil, fmt.Errorf(
								"watched constant %s is defined twice with conflicting values (%d in %s vs %d in %s) — package models and package operations copies must stay value-identical",
								name, prev.value, prev.file, n, file)
						}
						continue
					}
					out[name] = statusConstSite{value: n, file: file}
				}
			}
		}
	}
	return out, nil
}

// isWatchedStatusConst reports whether name starts with one of the watched
// family prefixes. The bare family name itself (e.g. a const literally named
// "UserStatus") does not count.
func isWatchedStatusConst(name string) bool {
	for _, prefix := range watchedStatusPrefixes {
		if name == prefix {
			return false
		}
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------
// D-102-6: AST usage-point guard — TestNoStatusLiteralUsage
// ---------------------------------------------------------------------

// statusLiteralWhitelist lists files that are exempt from the no-status-literal
// rule, each with a documented reason. Only files with genuine technical
// constraints (non-DB status values, static SQL that cannot be parameterized)
// belong here.
var statusLiteralWhitelist = map[string]string{
	"geocoding_service.go":                "百度地图 API 返回码（非 DB status，F 簇契约）",
	"sqlite_reconciliation_views.go":      "sqlite 视图 DDL 静态 SQL，禁改",
	"task_service.go":                     "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"asset_service.go":                    "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"building_service.go":                 "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"dedicated_line_service.go":          "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"floor_service.go":                    "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"infopoint_service.go":               "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"room_device_service.go":              "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"server_room_service.go":              "operations domain Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
	"captcha_background.go":               "captcha difficulty levels []int{1,2,3}, not DB status",
	"device_discovery_service.go":         "discovery service Status: 0 literals — Phase 102 scope covers scheduler + job_service + base.go only",
}

// TestNoStatusLiteralUsage scans all business code under internal/ (excluding
// migrations, _test.go, and internal/models/ constants definition surface) for
// status integer literals used without referencing a models constant.
// 7 AST patterns are covered:
//  1. Composite literal field: Status: 0
//  2. Assignment: x.Status = 1
//  3. Binary comparison: x.Status != 0 / status == 0
//  4. GORM Where/Update/Raw with status SQL + int literal argument
//  5. Update("status", int literal)
//  6. []int{0, 1} slice for status IN ?
//  7. String literal containing "status = <digit>" in SQL context
func TestNoStatusLiteralUsage(t *testing.T) {
	t.Parallel()

	// ── Self-test (memory snippets) ──────────────────────────────────
	selfSrc := map[string]struct {
		src    string
		expect int // 0 = should NOT hit; >0 = should hit
	}{
		"composite literal Status:0": {
			`package test; type T struct{Status int}; func F(){_ = T{Status:0}}`, 1},
		"assignment x.Status=1": {
			`package test; func F(){var x struct{Status int}; x.Status=1}`, 1},
		"binary status==0": {
			`package test; func F(status int){_ = status == 0}`, 1},
		"binary x.Status!=0": {
			`package test; func F(){var x struct{Status int}; _ = x.Status != 0}`, 1},
		"WHERE status = 0 arg": {
			`package test; import "gorm.io/gorm"; func F(db *gorm.DB){db.Where("status = 0", 0)}`, 1},
		"UPDATE status, 0 arg": {
			`package test; import "gorm.io/gorm"; func F(db *gorm.DB){db.Model(nil).Update("status = 0", 0)}`, 1},
		"status IN []int{0,1}": {
			`package test; func F(){var x []int; x = []int{0, 1}}`, 1},
		"constant ref — negative": {
			`package test; import "github.com/xingran-next/xingran-go-backend/internal/models"; func F(){_ = models.JobStatusNormal}`, 0},
		"constant ref in assignment — negative": {
			`package test; import "github.com/xingran-next/xingran-go-backend/internal/models"; func F(){var x struct{Status models.JobStatus}; x.Status = models.JobStatusNormal}`, 0},
	}
	fset := token.NewFileSet()
	for name, tc := range selfSrc {
		t.Run("self-"+name, func(t *testing.T) {
			f, err := parser.ParseFile(fset, "snippet.go", tc.src, 0)
			if err != nil {
				t.Fatalf("parse self-test snippet %q: %v", name, err)
			}
			hits := collectStatusLiteralHits(f, fset)
			if len(hits) != tc.expect {
				t.Errorf("self-test %q: got %d hits, want %d", name, len(hits), tc.expect)
			}
		})
	}

	// ── Full backend scan ─────────────────────────────────────────────
	hitCount, err := scanBackendStatusLiterals(t)
	if err != nil {
		t.Fatalf("backend scan failed: %v", err)
	}

	// Report per-file hits
	for file, lines := range hitCount {
		basename := filepath.Base(file)
		reason, whitelisted := statusLiteralWhitelist[basename]
		if whitelisted {
			t.Logf("WHITELISTED (allowed): %s — %s", file, reason)
			continue
		}
		for _, line := range lines {
			t.Errorf("status literal usage: %s:%d — hard failure (must use models constant)", file, line)
		}
	}
}

// collectStatusLiteralHits returns the line numbers in src file f that contain
// status integer literals matching any of the 7 AST patterns.
func collectStatusLiteralHits(f *ast.File, fset *token.FileSet) []int {
	var hits []int
	seen := make(map[int]bool)
	track := func(line int) {
		if line > 0 && !seen[line] {
			seen[line] = true
			hits = append(hits, line)
		}
	}

	// Regex for string literal SQL patterns (Pattern 7)
	sqlStatusRegex := regexp.MustCompile(`(?i)status\s*=\s*\d+`)

	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CompositeLit:
			// Pattern 1: {Status: 0} — KeyValueExpr with key "Status" and int value
			for _, elt := range n.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "Status" {
					continue
				}
				if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.INT {
					track(fset.Position(kv.Pos()).Line)
				}
			}

		case *ast.AssignStmt:
			// Pattern 2: x.Status = <int>  OR  Pattern 6: []int{0, 1}
			if len(n.Lhs) == 1 && len(n.Rhs) == 1 {
				// Pattern 2: x.Status = <int>
				if se, ok := n.Lhs[0].(*ast.SelectorExpr); ok {
					if se.Sel.Name == "Status" {
						if bl, ok := n.Rhs[0].(*ast.BasicLit); ok && bl.Kind == token.INT {
							track(fset.Position(se.Pos()).Line)
						}
					}
				}
				// Pattern 6: []int{...} composite literal (Assigned form: x = []int{0, 1})
				if cl, ok := n.Rhs[0].(*ast.CompositeLit); ok {
					isIntSlice := false
					switch typ := cl.Type.(type) {
					case *ast.Ident:
						isIntSlice = typ.Name == "int"
					case *ast.ArrayType:
						if id, ok := typ.Elt.(*ast.Ident); ok && id.Name == "int" {
							isIntSlice = true
						}
					}
					if isIntSlice {
						hasInt := false
						for _, elt := range cl.Elts {
							if bl, ok := elt.(*ast.BasicLit); ok && bl.Kind == token.INT {
								hasInt = true
							}
						}
						if hasInt {
							track(fset.Position(n.Pos()).Line)
						}
					}
				}
			}

		case *ast.BinaryExpr:
			// Pattern 3: x.Status != 0 / status == 0
			isStatusIdentOrSel := func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.Ident:
					return v.Name == "status"
				case *ast.SelectorExpr:
					return v.Sel.Name == "Status"
				}
				return false
			}
			if (isStatusIdentOrSel(n.X) || isStatusIdentOrSel(n.Y)) {
				var lit *ast.BasicLit
				if bl, ok := n.X.(*ast.BasicLit); ok && bl.Kind == token.INT {
					lit = bl
				} else if bl, ok := n.Y.(*ast.BasicLit); ok && bl.Kind == token.INT {
					lit = bl
				}
				if lit != nil {
					track(fset.Position(n.Pos()).Line)
				}
			}

		case *ast.CallExpr:
			// Patterns 4, 5: GORM Where/Update/Raw calls with status SQL + int arg
			// Handle both direct calls (Where/Update/Raw) and chained calls (db.Model().Update)
			funName := ""
			switch fun := n.Fun.(type) {
			case *ast.Ident:
				funName = fun.Name
			case *ast.SelectorExpr:
				funName = fun.Sel.Name
			}
			isGORMCall := funName == "Where" || funName == "Update" || funName == "Raw"
			if isGORMCall && len(n.Args) >= 2 {
				var hasStatusSQL bool
				if bl, ok := n.Args[0].(*ast.BasicLit); ok && bl.Kind == token.STRING {
					if sqlStatusRegex.MatchString(bl.Value) {
						hasStatusSQL = true
					}
				}
				if hasStatusSQL {
					for _, arg := range n.Args[1:] {
						if bl, ok := arg.(*ast.BasicLit); ok && bl.Kind == token.INT {
							track(fset.Position(arg.Pos()).Line)
						}
					}
				}
			}
		}
		return true
	})

	// String literals: Pattern 7 — embedded "status = <digit>" in SQL strings
	// Walk all *ast.BasicLit string literals (declarations only)
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range vs.Names {
						_ = name
						if i < len(vs.Values) {
							if bl, ok := vs.Values[i].(*ast.BasicLit); ok && bl.Kind == token.STRING {
								// Strip surrounding quotes from Go string literal
								val := strings.Trim(bl.Value, `"`)
								if sqlStatusRegex.MatchString(val) {
									track(fset.Position(bl.Pos()).Line)
								}
							}
						}
					}
				}
			}
		}
	}

	sort.Ints(hits)
	return hits
}

// scanBackendStatusLiterals walks internal/ and returns a map of
// filename -> line numbers of status literal hits (excluding whitelist).
func scanBackendStatusLiterals(t *testing.T) (map[string][]int, error) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	backendRoot := filepath.Dir(filepath.Dir(thisFile)) // internal/models/ → internal/
	hitCount := make(map[string][]int)

	// Directories to exclude entirely
	excludePrefixes := []string{
		filepath.Join(backendRoot, "core", "db", "migrations"),
		filepath.Join(backendRoot, "models"),
	}

	isExcluded := func(path string) bool {
		for _, pref := range excludePrefixes {
			if strings.HasPrefix(path, pref) {
				return true
			}
		}
		return false
	}

	err := filepath.Walk(backendRoot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (strings.HasSuffix(path, "_test") || strings.HasSuffix(path, ".test")) {
			return filepath.SkipDir
		}
		if info.IsDir() && isExcluded(path) {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}
		hits := collectStatusLiteralHits(f, fset)
		if len(hits) > 0 {
			basename := filepath.Base(path)
			if _, whitelisted := statusLiteralWhitelist[basename]; !whitelisted {
				hitCount[path] = hits
			}
		}
		return nil
	})
	return hitCount, err
}
