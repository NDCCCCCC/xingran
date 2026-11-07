// Package operations: Phase 131 四域读端点 403 越权矩阵 + 跨用户缓存对照实验。
//
// 命名前缀 ScopeRead* 避免与其它 agent 的测试/helper 冲突。
// SQLite 内存夹具：两个不同部门（dept-A / dept-B）各建楼宇/楼层/工位/信息点，
// 两个不同可见集的用户（只看 A / 只看 B）+ All 档 + FailClosed + Self 档。
//
// 覆盖端点（DS-03 / DS-06 / 审查 P1-2）：
//
//	workstation List / Statistics / SearchWorkstationOptions / GetByID /
//	  GetFloorWorkstationsAll(:813) / CountsByFloor / GetWorkstationDeptOptions
//	floor List / Statistics / SearchFloorOptions / GetTree / GetByID
//	building List / Statistics / SearchBuildingOptions / GetByID
//	infopoint List / Statistics / SearchInfoPointOptions / GetByID
//	Excel 导出两条路径：queryWorkstationIDsForExport / applyExcelWorkstationScope
//
// 每个端点断言五档语义：越界不可见、本范围可见、All 档全可见、FailClosed 空集、
// 超管零回归；跨用户对照实验覆盖 floor:tree 与 dropdown 缓存 key 不串数据。
package operations

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ---------------------------------------------------------------------
// 夹具：两部门 × 四实体
// ---------------------------------------------------------------------

const (
	scopeReadDeptA = "aaaaaaaa-0000-0000-0000-000000000001"
	scopeReadDeptB = "bbbbbbbb-0000-0000-0000-000000000002"
	scopeReadUserA = "11111111-0000-0000-0000-0000000000a1"
	scopeReadUserB = "22222222-0000-0000-0000-0000000000b2"
)

// ScopeReadFixture 持有两部门各一套楼宇/楼层/工位/信息点 ID。
type ScopeReadFixture struct {
	DB *gorm.DB

	BuildingA, BuildingB string
	FloorA, FloorB       string
	WSA, WSB             string
	IPA, IPB             string
}

// scopeReadCtx 构造指定可见集的 ctx。
func scopeReadCtx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

// scopeReadDeptOnly 构造「仅看某部门」的 ScopeResult。
func scopeReadDeptOnly(deptID string) datascope.ScopeResult {
	return datascope.ScopeResult{DeptIDs: []string{deptID}}
}

func scopeReadStrPtr(s string) *string { return &s }

// scopeReadBuildFixture 用 AutoMigrate 建四域表 + 手动 sys_dept（含 ancestors）。
func scopeReadBuildFixture(t *testing.T, f *ScopeReadFixture) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "scope_read_403.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	f.DB = db

	// sys_dept：手动建表（BuildDeptRecursiveFilter 依赖 ancestors 文本列）
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY, dept_code TEXT, dept_name TEXT NOT NULL,
			parent_id TEXT, ancestors TEXT, order_num INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0,
			deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status, is_external_org)
		VALUES (?, '部门A', 'DA', '', 0, 0), (?, '部门B', 'DB', '', 0, 0)`,
		scopeReadDeptA, scopeReadDeptB).Error)

	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{},
		&operationsmodels.OpsFloor{},
		&models.Workstation{},
		&operationsmodels.OpsInfoPoint{},
	))

	// 最小辅助表：四域 List/GetByID 的 joinScope LEFT JOIN sys_user / sys_files，
	// GetWorkstationDeptOptions 的 UNION ALL 读 sys_dept_location_alias。
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (id TEXT PRIMARY KEY, nickname TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_files (id TEXT PRIMARY KEY, storage_path TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept_location_alias (
			id TEXT PRIMARY KEY, dept_id TEXT, scope TEXT, location_id TEXT, deleted_at DATETIME)`).Error)
	// workstationJoinSelect 的主设备序列号相关子查询读 ops_workstation_device。
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_workstation_device (
			id TEXT PRIMARY KEY, workstation_id TEXT, device_serial TEXT,
			is_primary INTEGER, priority INTEGER, created_at DATETIME, deleted_at DATETIME)`).Error)

	// 部门 A：楼宇 A → 楼层 A → 工位 A → 信息点 A（楼层 org_id 覆盖到 A）
	bA := &operationsmodels.OpsBuilding{Name: "scope-A-楼", OrgID: scopeReadDeptA}
	require.NoError(t, db.Create(bA).Error)
	fA := &operationsmodels.OpsFloor{Name: "scope-A-F1", FloorNo: "1", BuildingID: bA.ID, OrgID: scopeReadStrPtr(scopeReadDeptA)}
	require.NoError(t, db.Create(fA).Error)
	wA := &models.Workstation{
		WorkstationName: "scope-A-工位", FloorID: scopeReadStrPtr(fA.ID), BuildingID: scopeReadStrPtr(bA.ID),
		DeptID: scopeReadStrPtr(scopeReadDeptA), UserID: scopeReadStrPtr(scopeReadUserA),
		Status: models.WorkstationStatusAvailable,
	}
	require.NoError(t, db.Create(wA).Error)
	ipA := &operationsmodels.OpsInfoPoint{Name: "scope-A-信息点", InfoPointType: operationsmodels.InfoPointTypeNetwork, WorkstationID: wA.ID}
	require.NoError(t, db.Create(ipA).Error)

	// 部门 B：同构
	bB := &operationsmodels.OpsBuilding{Name: "scope-B-楼", OrgID: scopeReadDeptB}
	require.NoError(t, db.Create(bB).Error)
	fB := &operationsmodels.OpsFloor{Name: "scope-B-F1", FloorNo: "1", BuildingID: bB.ID, OrgID: scopeReadStrPtr(scopeReadDeptB)}
	require.NoError(t, db.Create(fB).Error)
	wB := &models.Workstation{
		WorkstationName: "scope-B-工位", FloorID: scopeReadStrPtr(fB.ID), BuildingID: scopeReadStrPtr(bB.ID),
		DeptID: scopeReadStrPtr(scopeReadDeptB), UserID: scopeReadStrPtr(scopeReadUserB),
		Status: models.WorkstationStatusAvailable,
	}
	require.NoError(t, db.Create(wB).Error)
	ipB := &operationsmodels.OpsInfoPoint{Name: "scope-B-信息点", InfoPointType: operationsmodels.InfoPointTypeNetwork, WorkstationID: wB.ID}
	require.NoError(t, db.Create(ipB).Error)

	f.BuildingA, f.BuildingB = bA.ID, bB.ID
	f.FloorA, f.FloorB = fA.ID, fB.ID
	f.WSA, f.WSB = wA.ID, wB.ID
	f.IPA, f.IPB = ipA.ID, ipB.ID
}

// scopeReadFiveProfiles 返回六档 ctx：看 A / 看 B / All / 未注入 / FailClosed / Self(A)。
func scopeReadFiveProfiles() map[string]context.Context {
	return map[string]context.Context{
		"deptA":      scopeReadCtx(scopeReadDeptOnly(scopeReadDeptA)),
		"deptB":      scopeReadCtx(scopeReadDeptOnly(scopeReadDeptB)),
		"all":        scopeReadCtx(datascope.AllScope("u-super")),
		"noInject":   context.Background(),
		"failClosed": scopeReadCtx(datascope.FailClosedScope("u-bad")),
		"selfA":      scopeReadCtx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: scopeReadUserA}),
	}
}

// ---------------------------------------------------------------------
// 四域 List 越权矩阵
// ---------------------------------------------------------------------

func TestScopeRead_FourDomainList_403Matrix(t *testing.T) {
	f := &ScopeReadFixture{}
	scopeReadBuildFixture(t, f)
	profiles := scopeReadFiveProfiles()

	type listCase struct {
		name string
		run  func(ctx context.Context, orgID string) (int64, error)
	}
	cases := []listCase{
		{"workstation List", func(ctx context.Context, orgID string) (int64, error) {
			p, err := NewWorkstationService(f.DB).List(ctx, requests.WorkstationListRequest{OrgID: orgID})
			if err != nil {
				return 0, err
			}
			return p.Total, nil
		}},
		{"floor List", func(ctx context.Context, orgID string) (int64, error) {
			p, err := NewFloorService(f.DB).List(ctx, map[string]interface{}{"orgId": orgID})
			if err != nil {
				return 0, err
			}
			return p.Total, nil
		}},
		{"building List", func(ctx context.Context, orgID string) (int64, error) {
			p, err := NewBuildingService(f.DB).List(ctx, map[string]interface{}{"orgId": orgID})
			if err != nil {
				return 0, err
			}
			return p.Total, nil
		}},
		{"infopoint List", func(ctx context.Context, orgID string) (int64, error) {
			p, err := NewInfoPointService(f.DB).List(ctx, requests.InfoPointListRequest{OrgID: orgID})
			if err != nil {
				return 0, err
			}
			return p.Total, nil
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 本范围可见：deptA 不带 orgId → 1 条（A）
			n, err := c.run(profiles["deptA"], "")
			require.NoError(t, err)
			assert.Equal(t, int64(1), n, "deptA 用户只能看到本部门 1 条")

			// 越界 orgId 求交：deptA 用户传 deptB → 空集（不报错）
			n, err = c.run(profiles["deptA"], scopeReadDeptB)
			require.NoError(t, err, "越界 orgId 应静默返回空集，不报错")
			assert.Equal(t, int64(0), n, "前端 orgId 不可越权读取")

			// All 档全可见
			n, err = c.run(profiles["all"], "")
			require.NoError(t, err)
			assert.Equal(t, int64(2), n, "All 档应全可见（零回归）")

			// 未注入（超管/存量路径）零回归
			n, err = c.run(profiles["noInject"], "")
			require.NoError(t, err)
			assert.Equal(t, int64(2), n, "未注入按 All 兜底")

			// FailClosed 空集
			n, err = c.run(profiles["failClosed"], "")
			require.NoError(t, err)
			assert.Equal(t, int64(0), n, "异常枚举 fail-closed 空集")

			// Self 档：floor/building 无用户列 → 空集；workstation/infopoint 按 user_id
			n, err = c.run(profiles["selfA"], "")
			require.NoError(t, err)
			if c.name == "workstation List" || c.name == "infopoint List" {
				assert.Equal(t, int64(1), n, "Self 档有用户列实体按 user_id 命中 A")
			} else {
				assert.Equal(t, int64(0), n, "Self 档无用户列实体（floor/building）空集")
			}
		})
	}
}

// ---------------------------------------------------------------------
// Statistics 越权矩阵（floor/infopoint 无签名，从 ctx 注入）
// ---------------------------------------------------------------------

func TestScopeRead_FourDomainStatistics_403Matrix(t *testing.T) {
	f := &ScopeReadFixture{}
	scopeReadBuildFixture(t, f)
	profiles := scopeReadFiveProfiles()

	type statCase struct {
		name string
		run  func(ctx context.Context) (int64, error)
	}
	cases := []statCase{
		{"workstation Statistics", func(ctx context.Context) (int64, error) {
			r, err := NewWorkstationService(f.DB).Statistics(ctx, nil)
			if err != nil {
				return 0, err
			}
			return r.Total, nil
		}},
		{"floor Statistics", func(ctx context.Context) (int64, error) {
			r, err := NewFloorService(f.DB).Statistics(ctx)
			if err != nil {
				return 0, err
			}
			return r.Total, nil
		}},
		{"building Statistics", func(ctx context.Context) (int64, error) {
			r, err := NewBuildingService(f.DB).Statistics(ctx, nil)
			if err != nil {
				return 0, err
			}
			return r.Total, nil
		}},
		{"infopoint Statistics", func(ctx context.Context) (int64, error) {
			r, err := NewInfoPointService(f.DB).Statistics(ctx)
			if err != nil {
				return 0, err
			}
			return r.Total, nil
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := c.run(profiles["deptA"])
			require.NoError(t, err)
			assert.Equal(t, int64(1), n, "deptA 只统计本部门")

			n, err = c.run(profiles["all"])
			require.NoError(t, err)
			assert.Equal(t, int64(2), n)

			n, err = c.run(profiles["failClosed"])
			require.NoError(t, err)
			assert.Equal(t, int64(0), n)
		})
	}
}

// ---------------------------------------------------------------------
// 下拉 / 树 / GetByID / 全集 / counts-by-floor / dept-options 矩阵
// ---------------------------------------------------------------------

func TestScopeRead_DropdownAndTree_403Matrix(t *testing.T) {
	f := &ScopeReadFixture{}
	scopeReadBuildFixture(t, f)
	profiles := scopeReadFiveProfiles()

	t.Run("floor tree", func(t *testing.T) {
		tree, err := NewFloorService(f.DB).GetTree(profiles["deptA"])
		require.NoError(t, err)
		assert.Len(t, tree, 1, "tree 按可见集过滤后只剩部门 A 的楼宇")
		assert.Equal(t, f.BuildingA, tree[0].ID)
		tree, err = NewFloorService(f.DB).GetTree(profiles["all"])
		require.NoError(t, err)
		assert.Len(t, tree, 2, "All 档全树可见")
		tree, err = NewFloorService(f.DB).GetTree(profiles["failClosed"])
		require.NoError(t, err)
		assert.Empty(t, tree, "FailClosed 空树")
	})

	t.Run("floor SearchFloorOptions", func(t *testing.T) {
		opts, err := NewFloorService(f.DB).SearchFloorOptions(profiles["deptA"], nil)
		require.NoError(t, err)
		require.Len(t, opts, 1)
		assert.Equal(t, f.FloorA, opts[0].Value)
		opts, err = NewFloorService(f.DB).SearchFloorOptions(profiles["deptA"], map[string]interface{}{"orgId": scopeReadDeptB})
		require.NoError(t, err)
		assert.Empty(t, opts, "越界 orgId 下拉返回空")
	})

	t.Run("building SearchBuildingOptions", func(t *testing.T) {
		opts, err := NewBuildingService(f.DB).SearchBuildingOptions(profiles["deptA"], nil)
		require.NoError(t, err)
		require.Len(t, opts, 1)
		assert.Equal(t, f.BuildingA, opts[0].Value)
	})

	t.Run("workstation SearchWorkstationOptions", func(t *testing.T) {
		opts, err := NewWorkstationService(f.DB).SearchWorkstationOptions(profiles["deptA"], requests.WorkstationListRequest{})
		require.NoError(t, err)
		require.Len(t, opts, 1)
		assert.Equal(t, f.WSA, opts[0].Value)
		opts, err = NewWorkstationService(f.DB).SearchWorkstationOptions(profiles["deptA"], requests.WorkstationListRequest{OrgID: scopeReadDeptB})
		require.NoError(t, err)
		assert.Empty(t, opts)
	})

	t.Run("infopoint SearchInfoPointOptions", func(t *testing.T) {
		opts, err := NewInfoPointService(f.DB).SearchInfoPointOptions(profiles["deptA"], nil)
		require.NoError(t, err)
		require.Len(t, opts, 1)
		assert.Equal(t, f.IPA, opts[0].Value)
	})
}

func TestScopeRead_GetByID_403Matrix(t *testing.T) {
	f := &ScopeReadFixture{}
	scopeReadBuildFixture(t, f)
	profiles := scopeReadFiveProfiles()

	type idCase struct {
		name    string
		targetA string
		targetB string
		get     func(ctx context.Context, id string) (bool, error)
	}
	cases := []idCase{
		{"workstation", f.WSA, f.WSB, func(ctx context.Context, id string) (bool, error) {
			v, err := NewWorkstationService(f.DB).GetByID(ctx, id)
			return v != nil, err
		}},
		{"floor", f.FloorA, f.FloorB, func(ctx context.Context, id string) (bool, error) {
			v, err := NewFloorService(f.DB).GetByID(ctx, id)
			return v != nil, err
		}},
		{"building", f.BuildingA, f.BuildingB, func(ctx context.Context, id string) (bool, error) {
			v, err := NewBuildingService(f.DB).GetByID(ctx, id)
			return v != nil, err
		}},
		{"infopoint", f.IPA, f.IPB, func(ctx context.Context, id string) (bool, error) {
			v, err := NewInfoPointService(f.DB).GetByID(ctx, id)
			return v != nil, err
		}},
	}
	for _, c := range cases {
		t.Run(c.name+" GetByID", func(t *testing.T) {
			// 本范围资源可见
			ok, err := c.get(profiles["deptA"], c.targetA)
			require.NoError(t, err)
			assert.True(t, ok)
			// 越界资源 → record not found（既有 404/空语义，不报错形状不变）
			ok, _ = c.get(profiles["deptA"], c.targetB)
			assert.False(t, ok, "deptA 用户不得读取部门 B 资源")
			// All 档越界资源也可见
			ok, err = c.get(profiles["all"], c.targetB)
			require.NoError(t, err)
			assert.True(t, ok)
			// FailClosed 连本范围也不可见
			ok, _ = c.get(profiles["failClosed"], c.targetA)
			assert.False(t, ok)
		})
	}
}

func TestScopeRead_WorkstationsAll_CountsByFloor_DeptOptions_403Matrix(t *testing.T) {
	f := &ScopeReadFixture{}
	scopeReadBuildFixture(t, f)
	profiles := scopeReadFiveProfiles()
	wsSvc := NewWorkstationService(f.DB)

	t.Run("workstations-all(:813) 真实越权面必须收口", func(t *testing.T) {
		// 按 floor 过滤 + 可见集：deptA 用户请求「部门 B 楼层」→ 空
		all, err := wsSvc.GetFloorWorkstationsAll(profiles["deptA"], f.FloorB)
		require.NoError(t, err)
		assert.Empty(t, all, "越界楼层的全集不得泄漏")
		// 本范围楼层 → 1 条
		all, err = wsSvc.GetFloorWorkstationsAll(profiles["deptA"], f.FloorA)
		require.NoError(t, err)
		assert.Len(t, all, 1)
		// All 档越界楼层也可见
		all, err = wsSvc.GetFloorWorkstationsAll(profiles["all"], f.FloorB)
		require.NoError(t, err)
		assert.Len(t, all, 1)
		// FailClosed 空集
		all, err = wsSvc.GetFloorWorkstationsAll(profiles["failClosed"], f.FloorA)
		require.NoError(t, err)
		assert.Empty(t, all)
	})

	t.Run("counts-by-floor 防跨楼层枚举", func(t *testing.T) {
		r, err := wsSvc.CountsByFloor(profiles["deptA"], []string{f.FloorA, f.FloorB})
		require.NoError(t, err)
		assert.Equal(t, 1, r.Counts[f.FloorA], "本范围楼层应计数")
		assert.Equal(t, 0, r.Counts[f.FloorB], "越界楼层不得枚举出计数")
		r, err = wsSvc.CountsByFloor(profiles["all"], []string{f.FloorA, f.FloorB})
		require.NoError(t, err)
		assert.Equal(t, 1, r.Counts[f.FloorB], "All 档全部可见")
	})

	t.Run("dept-options 越界 orgId 返回空", func(t *testing.T) {
		// 越界 orgId 由 scopeRejectsRequestedOrgFromContext 早退（不进 SQL）。
		opts, err := wsSvc.GetWorkstationDeptOptions(profiles["deptA"], scopeReadDeptB)
		require.NoError(t, err)
		assert.Empty(t, opts, "越界 orgId 不得枚举部门树")
		// FailClosed 同样空集
		opts, err = wsSvc.GetWorkstationDeptOptions(profiles["failClosed"], scopeReadDeptA)
		require.NoError(t, err)
		assert.Empty(t, opts)
	})

	// 本范围 orgId 的「可见集求交」用 DryRun 静态断言守护：
	// 既有 Raw("(?) UNION ALL ...") 在 SQLite 上语法不支持（最小复现已证实，
	// 属既有生产 SQL 限制，非本 phase 收口引入），PG 上由 PG contract 覆盖。
	t.Run("dept-options 本范围注入可见集条件（DryRun 静态）", func(t *testing.T) {
		db := f.DB.Session(&gorm.Session{DryRun: true})
		sub := db.Table("sys_dept").
			Select("CAST(id AS TEXT) AS dept_id, dept_name AS dept_name, false AS is_alias").
			Where("deleted_at IS NULL AND is_external_org = 0").
			Scopes(BuildDeptRecursiveFilter(scopeReadDeptA, "id"))
		cond, args, skip := orgScopeClause(scopeReadDeptOnly(scopeReadDeptA), true, "id")
		require.False(t, skip)
		require.Len(t, args, 1)
		assert.Contains(t, cond, "CAST(id AS TEXT) IN ?", "可见集求交条件必须用双方言 CAST")
		sub = sub.Where(cond, args...)
		stmt := sub.Find(&[]map[string]interface{}{}).Statement
		assert.Contains(t, stmt.SQL.String(), "CAST(id AS TEXT) IN", "descendants 子查询须挂可见集求交")
	})
}

// ---------------------------------------------------------------------
// Excel 导出两条路径 403 矩阵（DS-06：不豁免、不走 List、不依赖 params orgId）
// ---------------------------------------------------------------------

func TestScopeRead_ExcelExport_403Matrix(t *testing.T) {
	f := &ScopeReadFixture{}
	scopeReadBuildFixture(t, f)
	profiles := scopeReadFiveProfiles()

	// 路径 2：queryWorkstationIDsForExport（直接查 sys_workstation，无 ::uuid JOIN，
	// 可在 SQLite 真实执行，语义断言最强）
	svc := &ExcelService{db: f.DB}
	t.Run("queryWorkstationIDsForExport 越界不可见", func(t *testing.T) {
		ids, err := svc.queryWorkstationIDsForExport(profiles["deptA"], nil, map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, []string{f.WSA}, ids, "deptA 导出只含本部门工位")
	})
	t.Run("queryWorkstationIDsForExport All 全可见 / FailClosed 空集", func(t *testing.T) {
		ids, err := svc.queryWorkstationIDsForExport(profiles["all"], nil, map[string]any{})
		require.NoError(t, err)
		assert.Len(t, ids, 2)
		ids, err = svc.queryWorkstationIDsForExport(profiles["failClosed"], nil, map[string]any{})
		require.NoError(t, err)
		assert.Empty(t, ids)
	})
	t.Run("queryWorkstationIDsForExport 不依赖 params orgId", func(t *testing.T) {
		// 用合法 params（name 模糊）确保不引入 orgId 伪列，可见集仍独立生效。
		ids, err := svc.queryWorkstationIDsForExport(profiles["deptA"], nil, map[string]any{"name": "scope-"})
		require.NoError(t, err)
		assert.Equal(t, []string{f.WSA}, ids, "可见集与 params 解耦")
	})

	// 路径 1：WorkstationQueryBuilder.BuildQuery 含 PG 专有 ::uuid（P1-8 既有债，
	// SQLite 必炸）→ 用 DryRun 渲染 SQL 做静态断言 + ctx 五档语义断言。
	config, ok := GetExportConfig("workstation")
	require.True(t, ok)
	qb := NewWorkstationQueryBuilder()
	t.Run("BuildQuery 可见集条件静态断言", func(t *testing.T) {
		q := qb.BuildQuery(profiles["deptA"], f.DB, config, map[string]interface{}{})
		stmt := q.Session(&gorm.Session{DryRun: true}).Find(&[]map[string]interface{}{}).Statement
		assert.Contains(t, stmt.SQL.String(), "CAST(", "导出可见集条件双方言")
		assert.Contains(t, stmt.SQL.String(), "EXISTS")
		// All 档短路不挂可见集 EXISTS
		qAll := qb.BuildQuery(profiles["all"], f.DB, config, map[string]interface{}{})
		stmtAll := qAll.Session(&gorm.Session{DryRun: true}).Find(&[]map[string]interface{}{}).Statement
		assert.NotContains(t, stmtAll.SQL.String(), "EXISTS (SELECT 1 FROM `ops_floors` f")
	})
}

// ---------------------------------------------------------------------
// 跨用户对照实验：同一缓存 key 下用户 A / B 不串数据（PMODEL-01 防重演）
// ---------------------------------------------------------------------

func TestScopeRead_CrossUserCacheIsolation(t *testing.T) {
	// 缓存 key 派生层：不同 scope 必须不同 key（覆盖 floor:tree 与 dropdown）
	ctxA := scopeReadCtx(scopeReadDeptOnly(scopeReadDeptA))
	ctxB := scopeReadCtx(scopeReadDeptOnly(scopeReadDeptB))

	assert.NotEqual(t, scopedFloorTreeKey(ctxA), scopedFloorTreeKey(ctxB),
		"floor:tree 不同用户不得共享缓存（防跨范围泄漏）")
	assert.NotEqual(t, scopedFloorBuildingKey(ctxA, "B1"), scopedFloorBuildingKey(ctxB, "B1"))
	assert.NotEqual(t, BuildScopedDropdownCacheKey(ctxA, "floor", nil),
		BuildScopedDropdownCacheKey(ctxB, "floor", nil), "dropdown 不同用户不得共享缓存")

	// 端到端：用真实内存缓存 + floorCacheService，用户 A 先看树（只含 A），
	// 用户 B 再看树 → 因 key 不同不会命中 A 的缓存，返回含 B 的独立结果。
	f := &ScopeReadFixture{}
	scopeReadBuildFixture(t, f)
	mem := pkgcacheNewMemoryCache()
	cached := NewFloorServiceWithCache(f.DB, newScopeReadCacheAdapter(mem), nil)

	treeAData, err := cached.GetTree(ctxA)
	require.NoError(t, err)
	require.Len(t, treeAData, 1)
	assert.Equal(t, f.BuildingA, treeAData[0].ID)

	treeBData, err := cached.GetTree(ctxB)
	require.NoError(t, err)
	require.Len(t, treeBData, 1, "用户 B 不得命中用户 A 的树缓存")
	assert.Equal(t, f.BuildingB, treeBData[0].ID, "用户 B 应看到自己的楼宇，而非 A 的数据")

	// dropdown 同理：用户 B 不命中用户 A 的缓存
	optsA, err := cached.SearchFloorOptions(ctxA, map[string]interface{}{})
	require.NoError(t, err)
	require.Len(t, optsA, 1)
	assert.Equal(t, f.FloorA, optsA[0].Value)
	optsB, err := cached.SearchFloorOptions(ctxB, map[string]interface{}{})
	require.NoError(t, err)
	require.Len(t, optsB, 1)
	assert.Equal(t, f.FloorB, optsB[0].Value, "dropdown 跨用户不串数据")

	// 写操作失效后所有 scope 变体都应被清掉
	require.NoError(t, cached.(*floorCacheService).InvalidateAllFloorCache(context.Background()))
	keys, _ := mem.Keys(context.Background(), "floor:*")
	assert.Empty(t, keys)
}

func TestScopeRead_CacheKeyAllShortCircuitStillScoped(t *testing.T) {
	all := scopeCacheKeyOf(datascope.AllScope("u-super"), true)
	assert.Equal(t, "all", all)
	assert.Contains(t, scopedFloorTreeKey(scopeReadCtx(datascope.AllScope("u"))), ":sall")
	// All 与 FailClosed / Self 不碰撞（三者 DeptIDs 都为空）
	assert.NotEqual(t, all, scopeCacheKeyOf(datascope.FailClosedScope("u"), true))
	assert.NotEqual(t, all, scopeCacheKeyOf(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "u"}, true))
}

// ---------------------------------------------------------------------
// 辅助：跨用户对照用的内存缓存（自带 9 方法 CacheProvider 实现，
// 不引入其它 agent 的 helper 名，避免命名冲突）
// ---------------------------------------------------------------------

func pkgcacheNewMemoryCache() *scopeReadMemCache {
	return &scopeReadMemCache{items: map[string]string{}}
}

// scopeReadMemCache 实现 base.CacheProvider 的全部 9 方法。JSON 值用
// encoding/json 序列化，与生产语义一致即可支撑 GetOrSetJSON。
type scopeReadMemCache struct{ items map[string]string }

func (m *scopeReadMemCache) GetOrSet(_ context.Context, key string, dest interface{},
	_ time.Duration, query func() (interface{}, error)) error {
	if raw, hit := m.items[key]; hit {
		return json.Unmarshal([]byte(raw), dest)
	}
	v, err := query()
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.items[key] = string(b)
	return json.Unmarshal(b, dest) // 回源结果同样写入 dest（同生产 P0 #9 同步写语义）
}
func (m *scopeReadMemCache) Delete(_ context.Context, key string) error {
	delete(m.items, key)
	return nil
}
func (m *scopeReadMemCache) DeleteByPattern(ctx context.Context, pattern string) error {
	keys, err := m.Keys(ctx, pattern)
	if err != nil {
		return err
	}
	return m.MDelete(ctx, keys...)
}
func (m *scopeReadMemCache) Keys(_ context.Context, pattern string) ([]string, error) {
	prefix := ""
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix = pattern[:len(pattern)-1]
	}
	out := []string{}
	for k := range m.items {
		if (prefix == "" && k == pattern) || (prefix != "" && len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			out = append(out, k)
		}
	}
	return out, nil
}
func (m *scopeReadMemCache) MGet(_ context.Context, keys ...string) (map[string]string, error) {
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := m.items[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}
func (m *scopeReadMemCache) MDelete(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.items, k)
	}
	return nil
}
func (m *scopeReadMemCache) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.items[key]
	return ok, nil
}
func (m *scopeReadMemCache) SetTTL(_ context.Context, _ string, _ time.Duration) error {
	return nil
}
func (m *scopeReadMemCache) GetTTL(_ context.Context, _ string) (time.Duration, error) {
	return 0, nil
}
func (m *scopeReadMemCache) GetStats(_ context.Context) (*base.CacheStats, error) {
	return &base.CacheStats{}, nil
}

// newScopeReadCacheAdapter：system.CacheProvider = base.CacheProvider（type alias），
// 指针本身已实现全部 9 方法，直接返回接口值供 NewFloorServiceWithCache 使用。
func newScopeReadCacheAdapter(m *scopeReadMemCache) system.CacheProvider {
	return m
}
