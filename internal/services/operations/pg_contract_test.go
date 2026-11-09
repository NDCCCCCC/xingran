// Package operations: PG-gated 类型契约集成测试。
//
// 根因(2026-10-15,ops 三列表 500,详见 .planning/debug/resolved/ops-list-join-uuid-500.md):
// ops 表 FK 列在 PG 里是混合类型(多数 varchar(64),但 sys_workstation.building_id 与
// ops_floors.plan_image_id 是 uuid)。手写 JOIN SQL 一旦按"FK 全 varchar"统一写成
// CAST(pk AS TEXT) = fk,对 uuid 列即产生 text=uuid / varchar=uuid → SQLSTATE 42883。
// SQLite 动态类型对两种写法都放行 → "SQLite 单测全绿 + PG 42883 炸"类回归 SQLite 测不出,
// 本测试是唯一能真实守护这类类型契约的回归防线。
//
// 启用方式(默认跳过,无 TEST_PG_DSN 时 t.Skip,CI 不红):
//
//	TEST_PG_DSN="postgres://user:pass@host/db?sslmode=disable" \
//	  go test -v -run TestPGContract ./internal/services/operations/
//
// 只读保证:本测试对 DB 仅执行三服务 List/Statistics 内部的 SELECT(JOIN 无条件生效,
// 正是被守护路径);禁止任何 Create/Update/Delete/Raw 写 SQL —— TEST_PG_DSN 可能指向生产库。
//
// Phase 150：差分攻击测试（sidechannel_diff_attack_test.go）为静态断言 + SQLite 真跑；
// PG 双跑仍为 DIALECT-04③ 前置依赖，本期不建 services: postgres、不伪造双跑。
package operations

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openPGForContractTest 按 TEST_PG_DSN 打开真实 PG 连接(共享 helper)。
// 空 DSN → skip;连不通 → skip。返回 *gorm.DB,测试结束后自动关闭底层连接池。
func openPGForContractTest(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set; PG contract test skipped")
	}
	// pgx v5 禁写 keepalives 类 DSN 参数(进 RuntimeParams → FATAL 42704),
	// connect_timeout 是安全参数;防坏主机挂死。
	// URL 形式(postgres://...?k=v)用 & 追加,keyword 形式(host=... dbname=...)用空格;
	// 对 URL 用空格会把 sslmode 的值污染成 "disable connect_timeout=5" → TLS 配置解析失败。
	if !strings.Contains(dsn, "connect_timeout") {
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			sep := "?"
			if strings.Contains(dsn, "?") {
				sep = "&"
			}
			dsn += sep + "connect_timeout=5"
		} else {
			dsn += " connect_timeout=5"
		}
	}

	// DisableAutomaticPing:Open 阶段不探活(等价 lazy connect;GORM v1.30.5 无 OpenLazy 选项),
	// 由下方 PingContext 统一探活并控制超时
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Skipf("TEST_PG_DSN unreachable (gorm open): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("TEST_PG_DSN unreachable (sql.DB): %v", err)
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		t.Skipf("TEST_PG_DSN unreachable: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// requireNoPGErr 断言 err==nil;42883(operator does not exist)给出显式类型契约失败信息。
func requireNoPGErr(t *testing.T, stage string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "42883") {
		t.Fatalf("%s: PG type contract broken — FK comparison direction mismatch (uuid vs text/varchar): %v", stage, err)
	}
	t.Fatalf("%s: %v", stage, err)
}

// TestPGContract_OpsListJoins 三服务 List+Statistics 真实打 PG:
// 任一 JOIN 比较方向写反(uuid 列被 CAST 成 text 比较)都会在真实 PG 上报 42883 并在此失败。
func TestPGContract_OpsListJoins(t *testing.T) {
	db := openPGForContractTest(t)
	ctx := context.Background()

	t.Run("floor", func(t *testing.T) {
		floorSvc := NewFloorService(db)
		page, err := floorSvc.List(ctx, nil)
		requireNoPGErr(t, "floor List", err)
		if page == nil {
			t.Fatal("floor List: result is nil")
		}
		requireNoPGErr(t, "floor Statistics", func() error {
			_, err := floorSvc.Statistics(ctx)
			return err
		}())
	})

	t.Run("workstation", func(t *testing.T) {
		wsSvc := NewWorkstationService(db)
		page, err := wsSvc.List(ctx, requests.WorkstationListRequest{})
		requireNoPGErr(t, "workstation List", err)
		if page == nil {
			t.Fatal("workstation List: result is nil")
		}
		requireNoPGErr(t, "workstation Statistics", func() error {
			_, err := wsSvc.Statistics(ctx, nil)
			return err
		}())
	})

	t.Run("infopoint", func(t *testing.T) {
		ipSvc := NewInfoPointService(db)
		page, err := ipSvc.List(ctx, requests.InfoPointListRequest{})
		requireNoPGErr(t, "infopoint List", err)
		if page == nil {
			t.Fatal("infopoint List: result is nil")
		}
		requireNoPGErr(t, "infopoint Statistics", func() error {
			_, err := ipSvc.Statistics(ctx)
			return err
		}())
	})
}

// TestPGContract_WorkstationOrgAndBackfill 快速-261106-wsb/-wso 的类型契约守护
// （2026-11-07 生产 500 回归：sys_user.dept_id 是原生 uuid 列，候选查询里裸写
// `su.dept_id <> ”` 让 PG 尝试 ” → uuid 隐式转换报 invalid input syntax，
// SQLite 动态类型全绿测不出 —— 与本文件头部记载的 42883 同族缺陷）。
//
// 只读保证：BackfillDeptFromUser 以 dryRun=true 调用，service 内部仅 SELECT/Count，
// 不写库不清缓存（与 header 的「禁止写 SQL」约束一致）；Statistics/List 的 orgId
// 筛选是纯读 EXISTS 子查询。
func TestPGContract_WorkstationOrgAndBackfill(t *testing.T) {
	db := openPGForContractTest(t)
	ctx := context.Background()

	wsSvc := NewWorkstationService(db)

	// 1) 按人员反推部门预览（覆盖候选 JOIN 里的 su.dept_id/user_id 空串判断）。
	requireNoPGErr(t, "backfill-dept dryRun preview", func() error {
		_, err := wsSvc.BackfillDeptFromUser(ctx, true, false)
		return err
	}())

	// 1c) 回填写路径契约（审计 P2-3：dryRun 只覆盖读路径，UPDATE 的 CASE
	// WHEN CAST 形态与 WHERE id IN (?) 参数化推断此前无 PG 守护）。在显式
	// 事务里执行 fixMismatch=true 的完整写路径后**回滚**——真实走 UPDATE
	// SQL 但零净变更，符合本文件只读保证的精神。
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		t.Fatalf("begin tx: %v", tx.Error)
	}
	requireNoPGErr(t, "backfill-dept write path (rolled back)", func() error {
		_, err := NewWorkstationService(tx).BackfillDeptFromUser(ctx, false, true)
		return err
	}())
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// 1b) 导入兜底 infer 查询同款 SQL 直跑（审计 P0-1 回归：inferDeptFromUser
	// 内部 Warnf 吞错误、返回 0，无法靠调用它断言 —— 必须直接执行同款 JOIN
	// 才能拦住 uuid 列空串比较这类类型契约回归）。只 SELECT。
	requireNoPGErr(t, "excel infer query shape", func() error {
		return db.WithContext(ctx).
			Table("sys_user AS su").
			Select("su.id AS user_id, su.dept_id").
			Joins("INNER JOIN sys_dept d ON CAST(d.id AS TEXT) = CAST(su.dept_id AS TEXT)").
			Where("su.id IN ? AND su.deleted_at IS NULL AND su.dept_id IS NOT NULL AND CAST(su.dept_id AS TEXT) <> ''",
				[]string{"00000000-0000-0000-0000-000000000000"}).
			Find(&[]map[string]any{}).Error
	}())

	// 2) 工位 List/Statistics/SearchOptions 的有效机构 COALESCE 筛选
	//    （f.org_id/b.org_id 双侧 CAST 后的 COALESCE 类型契约）。
	const dummyOrg = "00000000-0000-0000-0000-000000000000"
	requireNoPGErr(t, "workstation List orgId COALESCE", func() error {
		_, err := wsSvc.List(ctx, requests.WorkstationListRequest{OrgID: dummyOrg})
		return err
	}())
	requireNoPGErr(t, "workstation Statistics orgId COALESCE", func() error {
		_, err := wsSvc.Statistics(ctx, map[string]interface{}{"orgId": dummyOrg})
		return err
	}())
	requireNoPGErr(t, "workstation SearchOptions orgId COALESCE", func() error {
		_, err := wsSvc.SearchWorkstationOptions(ctx, requests.WorkstationListRequest{OrgID: dummyOrg})
		return err
	}())

	// 3) 楼层 List/SearchFloorOptions 的同款 COALESCE 筛选。
	floorSvc := NewFloorService(db)
	requireNoPGErr(t, "floor List orgId COALESCE", func() error {
		_, err := floorSvc.List(ctx, map[string]interface{}{"orgId": dummyOrg})
		return err
	}())
	requireNoPGErr(t, "floor SearchFloorOptions orgId COALESCE", func() error {
		_, err := floorSvc.SearchFloorOptions(ctx, map[string]interface{}{"orgId": dummyOrg})
		return err
	}())
}

// TestPGContract_ExcelExportScope 守护 Phase 131 两条 Excel 导出独立路径注入的
// 可见集条件在真实 PG 上的类型契约（P1-8 / DS-06）。
//
// 门控：与全文件统一使用 TEST_PG_DSN；未设置/连不通 → t.Skip 带原因（CI 可观测）。
// 只读保证：仅执行 SELECT，不写库；BuildQuery 的 ::uuid JOIN 是既有技术债
// （P1-8，Phase 133 或后续清偿），本测试不修改它，只验证新增 CAST 条件的 PG 兼容性。
//
// 注意：BuildQuery 含 PG 专有 ::uuid/::text，SQLite 上必炸（既有测试
// TestExp77_ExportData_WorkstationMainSheet_FixtureBoundary 已锁定，Phase 148 更名），
// 故本用例必须在真实 PG 上跑；静态断言（不依赖 DSN）见 TestScopeExport_StaticContract。
func TestPGContract_ExcelExportScope(t *testing.T) {
	db := openPGForContractTest(t)
	ctx := context.Background()

	// 构造一个非 All 档的可见集（DeptIDs 非空），让 applyExcelWorkstationScope
	// 真正走到 EXISTS + CAST(... AS TEXT) IN ? 分支。
	scopeCtx := datascope.WithContext(ctx, datascope.ScopeResult{
		DeptIDs: []string{"00000000-0000-0000-0000-000000000000"},
	})

	config, ok := GetExportConfig("workstation")
	if !ok {
		t.Fatal("工位导出配置缺失")
	}

	t.Run("BuildQuery 可见集条件 PG 契约", func(t *testing.T) {
		qb := NewWorkstationQueryBuilder()
		// 真实 PG 执行整条 BuildQuery（只读），守护新增 CAST 条件无 42883。
		requireNoPGErr(t, "Excel BuildQuery scope filter", func() error {
			return qb.BuildQuery(scopeCtx, db, config, map[string]interface{}{}).
				Find(&[]map[string]interface{}{}).Error
		}())
	})

	t.Run("queryWorkstationIDsForExport 可见集条件 PG 契约", func(t *testing.T) {
		svc := &ExcelService{db: db}
		requireNoPGErr(t, "Excel queryWorkstationIDsForExport scope filter", func() error {
			_, err := svc.queryWorkstationIDsForExport(scopeCtx, nil, map[string]any{})
			return err
		}())
	})
}

// TestScopeWriteImport_StaticContract 静态断言守护「Excel 导入路径含可见集
// 校验」+「Phase 150 响应体去规模化不回退」（裁定 §4 / 验收硬约束 / D-3/D-6）。
// 不依赖 TEST_PG_DSN，CI 恒跑：防止后续重构把 filterScopeWriteImportRows 的
// 调用从 ImportData 里静默摘掉、或把全库规模字段 / 越界错误明细改回响应体。
func TestScopeWriteImport_StaticContract(t *testing.T) {
	readSrc := func(name string) string {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", name, err)
		}
		return string(src)
	}
	assertImportHook := func(token, desc string) {
		if !strings.Contains(readSrc("excel_service.go"), token) {
			t.Fatalf("导入路径可见集校验缺失：%s（应含 %q）", desc, token)
		}
	}
	assertImportHook("filterScopeWriteImportRows(", "ImportData 必须调用逐行可见集过滤")
	assertImportHook("result.ScopeOutOfScope = outOfScope", "越界行数必须回写 ImportResult（内部字段供 operlog）")
	assertImportHook("__excel_row__", "导入行号标记保留供内部定位，但越界行不写入响应体")

	// Phase 150 防复活静态断言（D-3/D-6）：源码层锁死「响应体去规模化」。
	forbidden := []string{
		"ScopeTotal", "ScopeVisible", "ScopeOutOfScope", "ScopeDeptTotal",
		"SkippedNoUser", "SkippedBadUserDept",
	}
	wsSrc := readSrc("workstation_service.go")
	for _, field := range forbidden {
		if strings.Contains(wsSrc, field+" ") || strings.Contains(wsSrc, field+"\t") ||
			strings.Contains(wsSrc, field+"\n") || strings.Contains(wsSrc, field+" int") {
			t.Fatalf("BackfillDeptResult 不得含已删规模字段 %q（响应体去规模化不得回退）", field)
		}
	}
	// D-3 防复活：BackfillDeptFromUser 内被整段删除的全库 countQ 专有 token
	// 不得出现（不误伤 :165 的 statistics 合法 COUNT —— 8 个 /statistics 已同源）。
	for _, dead := range []string{
		`countQ := s.db.WithContext(ctx).Table("sys_workstation")`,
		"countQ.Scan", "统计工位总数失败",
	} {
		if strings.Contains(wsSrc, dead) {
			t.Fatalf("workstation_service.go 不得残留不带 visibleCond 的全库 countQ 死代码 %q（D-3）", dead)
		}
	}
	excelSrc := readSrc("excel_service.go")
	if !strings.Contains(excelSrc, "ScopeOutOfScope int") || !strings.Contains(excelSrc, `json:"-"`) ||
		strings.Contains(excelSrc, `json:"scopeOutOfScope`) {
		t.Fatalf("ImportResult.ScopeOutOfScope 必须为 json:\"-\"（仅供 operlog，永不进 HTTP 响应）")
	}
	if strings.Contains(readSrc("scope_write.go"), "result.Errors = append") {
		t.Fatalf("filterScopeWriteImportRows 越界分支不得再 append Errors（D-6 静默跳过）")
	}
}

// TestPGContract_ScopeWriteImportResolve 把 Phase 132 导入逐行校验的新增
// 查询（scopeWriteResolveEffOrg 四种实体归属链）纳入 TEST_PG_DSN 门控覆盖：
// 真实 PG 上执行 SELECT，守护 effectiveOrgExpr 的 COALESCE / 双侧 CAST
// 与 JOIN 骨架无 42883（uuid/varchar 漂移）。
//
// 统一门控 openPGForContractTest：未设置/连不通 → t.Skip（CI 可观测）。
// 只读保证：仅 SELECT，不写库（与全文件一致）。
func TestPGContract_ScopeWriteImportResolve(t *testing.T) {
	db := openPGForContractTest(t)
	ctx := context.Background()
	const dummy = "00000000-0000-0000-0000-000000000000"

	cases := []struct {
		entity string
		record map[string]any
	}{
		{"building", map[string]any{"org_id": dummy}},
		{"floor", map[string]any{"building_id": dummy}},
		{"workstation", map[string]any{"floor_id": dummy}},
		{"infoPoint", map[string]any{"workstation_id": dummy}},
	}
	for _, tc := range cases {
		t.Run(tc.entity, func(t *testing.T) {
			requireNoPGErr(t, "导入归属链解析 "+tc.entity, func() error {
				_, err := scopeWriteResolveEffOrg(ctx, db, tc.entity, []map[string]any{tc.record})
				return err
			}())
		})
	}
}
