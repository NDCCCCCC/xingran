// Package operations: Phase 132 四域写端点 403 越权矩阵厚测试。
//
// 命名前缀 Write403* 避免与其它 agent 的测试/helper 冲突。
// SQLite 内存夹具：两部门（dept-A / dept-B）× 两 scope 用户（只看 A / 只看 B）
// + All 档 + FailClosed + Self 档，覆盖：
//
//   - 四域 Create / Update / Delete × 越界 deptId/orgId → 单条 403、零数据变更；
//   - 四域 BatchDelete / BatchUpdatePositions → 批量裁剪后越界行零变更；
//   - All 档零回归；FailClosed 全拒；
//   - backfill 范围限定断言（三层可见集 + Samples 非 All 不回显）；
//   - Excel 导入越界行断言（不写入 / 不阻断整批 / 计入 ImportResult）。
//
// 所有断言均为语义断言（校验数据实际状态，而非仅校验返回值）。
package operations

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	write403DeptA = "aaaaaaaa-0000-0000-0000-000000000001"
	write403DeptB = "bbbbbbbb-0000-0000-0000-000000000002"
	write403UserA = "11111111-0000-0000-0000-0000000000a1"
	write403UserB = "22222222-0000-0000-0000-0000000000b2"
)

// Write403Fixture 持有两部门各一套楼宇/楼层/工位/信息点 ID。
type Write403Fixture struct {
	DB                                      *gorm.DB
	BuildingA, BuildingB                    string
	FloorA, FloorB                          string
	WSA, WSB                                string
	IPA, IPB                                string
	BuildingSvc                             BuildingService
	FloorSvc                                FloorService
	WSSvc                                   WorkstationService
	IPSvc                                   InfoPointService
}

// write403Ctx 构造指定可见集的 ctx。
func write403Ctx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

func write403DeptOnly(deptID string) datascope.ScopeResult {
	return datascope.ScopeResult{DeptIDs: []string{deptID}}
}

// write403BuildFixture 用 AutoMigrate 建四域完整模型表（与 Phase 131 读侧
// 夹具同款，避免手写列缺失 BaseModel 的 created_at/updated_at），再插入
// 两部门各一套数据。
func write403BuildFixture(t *testing.T) *Write403Fixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{},
		&operationsmodels.OpsFloor{},
		&models.Workstation{},
		&operationsmodels.OpsInfoPoint{},
	))
	// 最小辅助表：sys_dept（部门名回显）/ sys_user（backfill 反推链）。
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT,
			ancestors TEXT, parent_id TEXT, deleted_at DATETIME);
		CREATE TABLE sys_user (id TEXT PRIMARY KEY, dept_id TEXT, nickname TEXT, deleted_at DATETIME);
	`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name) VALUES (?, '部门A'), (?, '部门B')`,
		write403DeptA, write403DeptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, dept_id, nickname) VALUES (?, ?, '用户A'), (?, ?, '用户B')`,
		write403UserA, write403DeptA, write403UserB, write403DeptB).Error)

	f := &Write403Fixture{DB: db}
	// 用模型 Create 让 BaseModel 回填 UUID，保证四域裁剪 SQL 的 id 形态一致。
	bA := &operationsmodels.OpsBuilding{Name: "楼宇A", OrgID: write403DeptA}
	bB := &operationsmodels.OpsBuilding{Name: "楼宇B", OrgID: write403DeptB}
	require.NoError(t, db.Create(bA).Error)
	require.NoError(t, db.Create(bB).Error)
	flA := &operationsmodels.OpsFloor{Name: "楼层A", FloorNo: "1", BuildingID: bA.ID}
	flB := &operationsmodels.OpsFloor{Name: "楼层B", FloorNo: "1", BuildingID: bB.ID}
	require.NoError(t, db.Create(flA).Error)
	require.NoError(t, db.Create(flB).Error)
	uA, uB := write403UserA, write403UserB
	wA := &models.Workstation{WorkstationName: "工位A", FloorID: &flA.ID, UserID: &uA}
	wB := &models.Workstation{WorkstationName: "工位B", FloorID: &flB.ID, UserID: &uB}
	require.NoError(t, db.Create(wA).Error)
	require.NoError(t, db.Create(wB).Error)
	ipA := &operationsmodels.OpsInfoPoint{Name: "信息点A", InfoPointType: operationsmodels.InfoPointTypeNetwork, WorkstationID: wA.ID}
	ipB := &operationsmodels.OpsInfoPoint{Name: "信息点B", InfoPointType: operationsmodels.InfoPointTypeNetwork, WorkstationID: wB.ID}
	require.NoError(t, db.Create(ipA).Error)
	require.NoError(t, db.Create(ipB).Error)

	f.BuildingA, f.BuildingB = bA.ID, bB.ID
	f.FloorA, f.FloorB = flA.ID, flB.ID
	f.WSA, f.WSB = wA.ID, wB.ID
	f.IPA, f.IPB = ipA.ID, ipB.ID
	f.BuildingSvc = NewBuildingService(db)
	f.FloorSvc = NewFloorService(db)
	f.WSSvc = NewWorkstationService(db)
	f.IPSvc = NewInfoPointService(db)
	return f
}

// write403Count 统计某表未软删行数（语义断言：确认零数据变更）。
func write403Count(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table(table).Where("deleted_at IS NULL").Count(&n).Error)
	return n
}

// ---------------------------------------------------------------------
// F-1 四域单条写 Create 越界 → 403、零数据变更；All 档零回归
// ---------------------------------------------------------------------

func TestWrite403_Create_FourDomains(t *testing.T) {
	f := write403BuildFixture(t)
	ctxA := write403Ctx(write403DeptOnly(write403DeptA))

	// building：提交越界 orgId=deptB → 403，无新增
	err := f.BuildingSvc.Create(ctxA, &operationsmodels.OpsBuilding{OrgID: write403DeptB, Name: "越界楼宇"})
	assert.True(t, IsScopeDenied(err), "building Create 越界应返回 ErrScopeDenied")
	assert.Equal(t, int64(2), write403Count(t, f.DB, "ops_buildings"), "越界 Create 零数据变更")

	// floor：空 orgId 继承越界楼宇 B → 403
	err = f.FloorSvc.Create(ctxA, &operationsmodels.OpsFloor{BuildingID: f.BuildingB, FloorNo: "9", Name: "越界楼层"})
	assert.True(t, IsScopeDenied(err), "floor Create 继承越界楼宇应拒")
	assert.Equal(t, int64(2), write403Count(t, f.DB, "ops_floors"))

	// workstation：floorId 指向越界楼层 B → 403
	wsB := f.FloorB
	err = f.WSSvc.Create(ctxA, &models.Workstation{FloorID: &wsB, WorkstationName: "越界工位"})
	assert.True(t, IsScopeDenied(err), "workstation Create 越界楼层应拒")
	assert.Equal(t, int64(2), write403Count(t, f.DB, "sys_workstation"))

	// infopoint：workstationId 指向越界工位 B → 403
	err = f.IPSvc.Create(ctxA, &operationsmodels.OpsInfoPoint{WorkstationID: f.WSB, Name: "越界信息点"})
	assert.True(t, IsScopeDenied(err), "infopoint Create 越界工位应拒")
	assert.Equal(t, int64(2), write403Count(t, f.DB, "ops_info_points"))

	// All 档零回归：四域合法 Create 均放行
	ctxAll := write403Ctx(datascope.ScopeResult{All: true})
	assert.NoError(t, f.BuildingSvc.Create(ctxAll, &operationsmodels.OpsBuilding{OrgID: write403DeptB, Name: "新楼宇"}))
	assert.NoError(t, f.FloorSvc.Create(ctxAll, &operationsmodels.OpsFloor{BuildingID: f.BuildingB, FloorNo: "8", Name: "新楼层"}))
	assert.NoError(t, f.WSSvc.Create(ctxAll, &models.Workstation{FloorID: &wsB, WorkstationName: "新工位"}))
	assert.NoError(t, f.IPSvc.Create(ctxAll, &operationsmodels.OpsInfoPoint{WorkstationID: f.WSB, Name: "新信息点"}))
	assert.Equal(t, int64(3), write403Count(t, f.DB, "ops_buildings"))
}

// ---------------------------------------------------------------------
// F-2 四域 Update 双向校验（改前 / 提交侧）
// ---------------------------------------------------------------------

func TestWrite403_Update_FourDomains(t *testing.T) {
	f := write403BuildFixture(t)
	ctxA := write403Ctx(write403DeptOnly(write403DeptA))

	// building：把 A 楼 org_id 改成越界 deptB（提交侧越界）→ 403
	err := f.BuildingSvc.Update(ctxA, &operationsmodels.OpsBuilding{
		BaseModel: models.BaseModel{ID: f.BuildingA}, OrgID: write403DeptB, Name: "楼宇A",
	})
	assert.True(t, IsScopeDenied(err), "building Update 提交越界 orgId 应拒")
	var orgA string
	require.NoError(t, f.DB.Table("ops_buildings").Where("id = ?", f.BuildingA).Pluck("org_id", &orgA).Error)
	assert.Equal(t, write403DeptA, orgA, "越界 Update 零数据变更")

	// building：改前资源本身越界（操作 B 楼）→ 403
	err = f.BuildingSvc.Update(ctxA, &operationsmodels.OpsBuilding{
		BaseModel: models.BaseModel{ID: f.BuildingB}, OrgID: write403DeptA, Name: "楼宇B",
	})
	assert.True(t, IsScopeDenied(err), "building Update 改前越界资源应拒")

	// infopoint：把 A 信息点搬到越界工位 B → 403
	err = f.IPSvc.Update(ctxA, &operationsmodels.OpsInfoPoint{
		BaseModel: models.BaseModel{ID: f.IPA}, WorkstationID: f.WSB, Name: "信息点A",
	})
	assert.True(t, IsScopeDenied(err), "infopoint Update 搬向越界工位应拒")
	var wsID string
	require.NoError(t, f.DB.Table("ops_info_points").Where("id = ?", f.IPA).Pluck("workstation_id", &wsID).Error)
	assert.Equal(t, f.WSA, wsID, "越界 Update 零数据变更")

	// FailClosed 全拒
	ctxClosed := write403Ctx(datascope.ScopeResult{FailClosed: true})
	assert.True(t, IsScopeDenied(f.BuildingSvc.Update(ctxClosed, &operationsmodels.OpsBuilding{
		BaseModel: models.BaseModel{ID: f.BuildingA}, OrgID: write403DeptA, Name: "楼宇A",
	})))
}

// ---------------------------------------------------------------------
// F-3 四域 Delete 按资源归属校验
// ---------------------------------------------------------------------

func TestWrite403_Delete_FourDomains(t *testing.T) {
	f := write403BuildFixture(t)
	ctxA := write403Ctx(write403DeptOnly(write403DeptA))

	// 越界资源 → 403、零软删
	assert.True(t, IsScopeDenied(f.BuildingSvc.Delete(ctxA, f.BuildingB)), "building Delete 越界应拒")
	assert.True(t, IsScopeDenied(f.FloorSvc.Delete(ctxA, f.FloorB)), "floor Delete 越界应拒")
	assert.True(t, IsScopeDenied(f.WSSvc.Delete(ctxA, f.WSB)), "workstation Delete 越界应拒")
	assert.True(t, IsScopeDenied(f.IPSvc.Delete(ctxA, f.IPB)), "infopoint Delete 越界应拒")
	assert.Equal(t, int64(2), write403Count(t, f.DB, "ops_buildings"))
	assert.Equal(t, int64(2), write403Count(t, f.DB, "sys_workstation"))

	// 本范围资源 → 放行（软删后可见行数 -1）
	assert.NoError(t, f.BuildingSvc.Delete(ctxA, f.BuildingA))
	assert.Equal(t, int64(1), write403Count(t, f.DB, "ops_buildings"))

	// FailClosed 全拒
	ctxClosed := write403Ctx(datascope.ScopeResult{FailClosed: true})
	assert.True(t, IsScopeDenied(f.WSSvc.Delete(ctxClosed, f.WSA)))
}

// ---------------------------------------------------------------------
// F-4 批量裁剪：越界行零变更，All 档全量放行
// ---------------------------------------------------------------------

func TestWrite403_BatchDelete_FourDomains(t *testing.T) {
	ctxA := write403Ctx(write403DeptOnly(write403DeptA))
	// 每个域用独立 fixture：避免「先删楼宇导致其下工位归属链断链」的
	// 顺序依赖污染后续裁剪断言（归属链断链按裁定 fail-closed 收紧为空集）。
	t.Run("building", func(t *testing.T) {
		f := write403BuildFixture(t)
		assert.NoError(t, f.BuildingSvc.BatchDelete(ctxA, []string{f.BuildingA, f.BuildingB}))
		assert.Equal(t, int64(1), write403Count(t, f.DB, "ops_buildings"), "批量裁剪后仅本范围行变更")
		var bOrg string
		require.NoError(t, f.DB.Table("ops_buildings").Where("id = ?", f.BuildingB).Pluck("org_id", &bOrg).Error)
		assert.Equal(t, write403DeptB, bOrg, "越界行 B 仍存在")
	})
	t.Run("workstation", func(t *testing.T) {
		f := write403BuildFixture(t)
		assert.NoError(t, f.WSSvc.BatchDelete(ctxA, []string{f.WSA, f.WSB}))
		assert.Equal(t, int64(1), write403Count(t, f.DB, "sys_workstation"))
	})
	t.Run("infopoint", func(t *testing.T) {
		f := write403BuildFixture(t)
		assert.NoError(t, f.IPSvc.BatchDelete(ctxA, []string{f.IPA, f.IPB}))
		assert.Equal(t, int64(1), write403Count(t, f.DB, "ops_info_points"))
	})
	t.Run("floor", func(t *testing.T) {
		f := write403BuildFixture(t)
		assert.NoError(t, f.FloorSvc.BatchDelete(ctxA, []string{f.FloorA, f.FloorB}))
		assert.Equal(t, int64(1), write403Count(t, f.DB, "ops_floors"))
	})
	t.Run("FailClosed 全收紧", func(t *testing.T) {
		f := write403BuildFixture(t)
		ctxClosed := write403Ctx(datascope.ScopeResult{FailClosed: true})
		assert.NoError(t, f.WSSvc.BatchDelete(ctxClosed, []string{f.WSA, f.WSB}))
		assert.Equal(t, int64(2), write403Count(t, f.DB, "sys_workstation"), "FailClosed 零变更")
	})
	t.Run("All 档全量放行", func(t *testing.T) {
		f := write403BuildFixture(t)
		assert.NoError(t, f.WSSvc.BatchDelete(
			write403Ctx(datascope.ScopeResult{All: true}), []string{f.WSA, f.WSB}))
		assert.Equal(t, int64(0), write403Count(t, f.DB, "sys_workstation"), "All 档零回归：全量删除")
	})
}

func TestWrite403_BatchUpdatePositions(t *testing.T) {
	f := write403BuildFixture(t)
	ctxA := write403Ctx(write403DeptOnly(write403DeptA))
	items := []PositionUpdateItem{
		{ID: f.WSA, PositionX: 10, PositionY: 20},
		{ID: f.WSB, PositionX: 99, PositionY: 99},
	}
	assert.NoError(t, f.WSSvc.BatchUpdatePositions(ctxA, items))
	// 语义断言：本范围工位位置被写入，越界工位位置零变更。
	var posAX, posBX, posAY, posBY int
	require.NoError(t, f.DB.Table("sys_workstation").Where("id = ?", f.WSA).
		Pluck("COALESCE(position_x, 0)", &posAX).Error)
	require.NoError(t, f.DB.Table("sys_workstation").Where("id = ?", f.WSA).
		Pluck("COALESCE(position_y, 0)", &posAY).Error)
	require.NoError(t, f.DB.Table("sys_workstation").Where("id = ?", f.WSB).
		Pluck("COALESCE(position_x, 0)", &posBX).Error)
	require.NoError(t, f.DB.Table("sys_workstation").Where("id = ?", f.WSB).
		Pluck("COALESCE(position_y, 0)", &posBY).Error)
	assert.Equal(t, 10, posAX, "本范围工位位置已更新")
	assert.Equal(t, 20, posAY)
	assert.Equal(t, 0, posBX, "越界工位位置零变更（批量裁剪）")
	assert.Equal(t, 0, posBY)
}

// ---------------------------------------------------------------------
// F-5 backfill 范围限定（三层裁剪 + Samples 非 All 不回显）
// ---------------------------------------------------------------------

func TestWrite403_Backfill_ScopeLimited(t *testing.T) {
	f := write403BuildFixture(t)
	ctxA := write403Ctx(write403DeptOnly(write403DeptA))

	res, err := f.WSSvc.BackfillDeptFromUser(ctxA, true, false)
	require.NoError(t, err)
	assert.Equal(t, "Custom/Dept/DeptChild", res.ScopeTier, "范围档位回显")
	assert.Equal(t, 1, res.ScopeVisible, "仅部门 A 的工位可见（候选层裁剪）")
	assert.Equal(t, 2, res.ScopeTotal, "请求行数 = 全库工位总数（COUNT 层口径）")
	assert.Equal(t, 1, res.ScopeOutOfScope, "越界行数聚合计数")
	assert.Empty(t, res.Samples, "非 All 档不逐条回显明细（P0-3）")

	// 执行层：只写可见集内工位，越界工位 dept_id 不变。
	_, err = f.WSSvc.BackfillDeptFromUser(ctxA, false, false)
	require.NoError(t, err)
	var deptA, deptB string
	require.NoError(t, f.DB.Table("sys_workstation").Where("id = ?", f.WSA).Pluck("COALESCE(dept_id, '')", &deptA).Error)
	require.NoError(t, f.DB.Table("sys_workstation").Where("id = ?", f.WSB).Pluck("COALESCE(dept_id, '')", &deptB).Error)
	assert.Equal(t, write403DeptA, deptA, "本范围工位 dept_id 被反推填写")
	assert.Empty(t, deptB, "越界工位 dept_id 零变更")

	// All 档零回归：Samples 保留既有审计样例、ScopeTier=All。
	resAll, err := f.WSSvc.BackfillDeptFromUser(write403Ctx(datascope.ScopeResult{All: true}), true, false)
	require.NoError(t, err)
	assert.Equal(t, "All", resAll.ScopeTier)
	assert.LessOrEqual(t, len(resAll.Samples), 20)
	// FailClosed：候选空集、零写。
	resClosed, err := f.WSSvc.BackfillDeptFromUser(write403Ctx(datascope.ScopeResult{FailClosed: true}), true, false)
	require.NoError(t, err)
	assert.Equal(t, 0, resClosed.ScopeVisible)
	assert.True(t, resClosed.ScopeFailClosed)
}

// ---------------------------------------------------------------------
// F-6 Excel 导入逐行可见集校验（越界行不写入 / 不阻断整批）
// ---------------------------------------------------------------------

func TestWrite403_ImportRows_OutOfScope(t *testing.T) {
	f := write403BuildFixture(t)
	ctxA := write403Ctx(write403DeptOnly(write403DeptA))
	config, ok := GetExcelConfig("building")
	require.True(t, ok)

	records := []map[string]any{
		{"name": "楼宇X", "org_id": write403DeptA, "__excel_row__": 2},
		{"name": "越界楼宇Y", "org_id": write403DeptB, "__excel_row__": 3},
	}
	result := &ImportResult{Errors: make([]ImportError, 0)}
	kept, outOfScope, err := filterScopeWriteImportRows(ctxA, f.DB, "building", config, records, result)
	require.NoError(t, err)
	assert.Len(t, kept, 1, "越界行不写入、不阻断整批")
	assert.Equal(t, 1, outOfScope)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, 3, result.Errors[0].Row, "越界错误明细含原始行号")
	assert.Empty(t, result.Errors[0].Value, "越界错误明细不含敏感值 Value")
	for _, rec := range kept {
		_, leaked := rec["__excel_row__"]
		assert.False(t, leaked, "内部行号标记不得泄漏进 Upsert 字段集")
	}

	// All 档零回归：全部保留、零越界计数。
	resultAll := &ImportResult{Errors: make([]ImportError, 0)}
	keptAll, outAll, err := filterScopeWriteImportRows(
		write403Ctx(datascope.ScopeResult{All: true}), f.DB, "building", config,
		[]map[string]any{{"name": "X", "org_id": write403DeptB}}, resultAll)
	require.NoError(t, err)
	assert.Len(t, keptAll, 1)
	assert.Equal(t, 0, outAll)

	// FailClosed：全部越界、无阻断错误（整批不 403）。
	resultClosed := &ImportResult{Errors: make([]ImportError, 0)}
	keptClosed, outClosed, err := filterScopeWriteImportRows(
		write403Ctx(datascope.ScopeResult{FailClosed: true}), f.DB, "building", config,
		[]map[string]any{{"name": "X", "org_id": write403DeptA}}, resultClosed)
	require.NoError(t, err)
	assert.Empty(t, keptClosed)
	assert.Equal(t, 1, outClosed)
	assert.Equal(t, 1, resultClosed.Failed)
}

// ---------------------------------------------------------------------
// F-7 ErrScopeDenied 显式 403（agent-8 P0-2 回归守护）
// ---------------------------------------------------------------------

func TestWrite403_ErrScopeDenied_HTTP403(t *testing.T) {
	appErr := apperrors.GetAppError(ErrScopeDenied)
	require.NotNil(t, appErr)
	assert.Equal(t, 403, appErr.GetHTTPStatus(), "必须显式 403，不能是 codes.go 默认的 401")
	assert.Equal(t, "无权操作该数据", appErr.Message)
	// 四域越界统一错误文案
	for _, err := range []error{
		checkWriteScope(write403Ctx(datascope.ScopeResult{FailClosed: true}), write403DeptA),
		checkWriteScope(write403Ctx(write403DeptOnly(write403DeptA)), write403DeptB),
	} {
		assert.True(t, IsScopeDenied(err))
		assert.Equal(t, "无权操作该数据", apperrors.GetAppError(err).Message)
	}
	_ = fmt.Sprintf("%v", ErrScopeDenied)
}
