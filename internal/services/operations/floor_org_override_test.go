package operations

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 一栋楼跨机构办公回归测试（quick-261106-wso）。
//
// 场景：浙商大厦 org_id = 分公司本部；4/5/6/47 楼继承本部；
// 1/3 楼 org_id = 武汉中支（楼层级覆盖）。
// 期望：选武汉中支时，工位列表/统计/楼层下拉/楼宇下拉都能正确命中 1/3 楼
// 与浙商大厦；选分公司本部时仍能拿到其余楼层（继承语义不被破坏）。

const (
	wsoOrgHQ       = "11111111-1111-1111-1111-111111111111" // 分公司本部
	wsoOrgWuhan    = "22222222-2222-2222-2222-222222222222" // 武汉中支
	wsoBuildingID  = "b-zheshang"
	wsoFloorHQ     = "f-4f"  // 继承本部
	wsoFloorWuhan  = "f-1f"  // 覆盖到武汉中支
	wsoFloorWuhan2 = "f-3f"  // 覆盖到武汉中支
)

// setupFloorOrgOverrideDB 建最小 schema + 浙商大厦跨机构数据。
func setupFloorOrgOverrideDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT,
			ancestors TEXT, is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0,
			deleted_at DATETIME
		)`).Error)
	// List 的 joinScope 会 LEFT JOIN sys_user / sys_files，并在 select 里
	// 子查询 ops_workstation_device，SQLite 下需补最小表。
	require.NoError(t, db.Exec(`CREATE TABLE sys_user (id TEXT PRIMARY KEY, nickname TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_files (id TEXT PRIMARY KEY, storage_path TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE ops_workstation_device (id TEXT PRIMARY KEY, workstation_id TEXT, device_serial TEXT, is_primary INTEGER, priority INTEGER, created_at DATETIME, deleted_at DATETIME)`).Error)

	// AutoMigrate 保证 OpsFloor 带本次新增的 org_id 列。
	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{},
		&operationsmodels.OpsFloor{},
		&models.Workstation{},
	))

	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status) VALUES
			(?, '分公司本部', 'HQ', '', 0),
			(?, '武汉中支', 'WH', '', 0)`,
		wsoOrgHQ, wsoOrgWuhan).Error)

	// 浙商大厦归属分公司本部。
	require.NoError(t, db.Exec(
		`INSERT INTO ops_buildings (id, name, org_id, level) VALUES (?, '浙商大厦', ?, 2)`,
		wsoBuildingID, wsoOrgHQ).Error)

	// 4 楼继承楼宇（org_id NULL）；1/3 楼覆盖到武汉中支。
	require.NoError(t, db.Exec(
		`INSERT INTO ops_floors (id, name, floor_no, building_id, org_id) VALUES
			(?, '4楼', '4', ?, NULL)`,
		wsoFloorHQ, wsoBuildingID).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO ops_floors (id, name, floor_no, building_id, org_id) VALUES
			(?, '1楼', '1', ?, ?),
			(?, '3楼', '3', ?, ?)`,
		wsoFloorWuhan, wsoBuildingID, wsoOrgWuhan,
		wsoFloorWuhan2, wsoBuildingID, wsoOrgWuhan).Error)

	// 每层一个工位。
	require.NoError(t, db.Exec(
		`INSERT INTO sys_workstation (id, workstation_name, floor_id, status) VALUES
			('ws-4f', 'W-4F-001', ?, 0),
			('ws-1f', 'W-1F-001', ?, 0),
			('ws-3f', 'W-3F-001', ?, 0)`,
		wsoFloorHQ, wsoFloorWuhan, wsoFloorWuhan2).Error)
	return db
}

// readDeptInferWorkstation 复用别名：读取单个工位的 dept_id / status。
func readFloorOrgWs(t *testing.T, db *gorm.DB, id string) (deptID string, status int) {
	t.Helper()
	var rec struct {
		DeptID *string
		Status int
	}
	require.NoError(t, db.Table("sys_workstation").Where("id = ?", id).First(&rec).Error, id)
	if rec.DeptID != nil {
		deptID = *rec.DeptID
	}
	return deptID, rec.Status
}

// TestFloorOrgOverride_WorkstationList 工位列表按覆盖机构筛选。
func TestFloorOrgOverride_WorkstationList(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	svc := NewWorkstationService(db)
	ctx := context.Background()

	// 选武汉中支 → 只出 1/3 楼的两个工位。
	res, err := svc.List(ctx, requests.WorkstationListRequest{OrgID: wsoOrgWuhan})
	require.NoError(t, err)
	list, ok := res.List.([]models.Workstation)
	require.True(t, ok, "List 应返回 []models.Workstation, got %T", res.List)
	assert.Len(t, list, 2, "武汉中支应只筛出 1/3 楼工位")
	names := make([]string, 0, len(list))
	for _, w := range list {
		names = append(names, w.WorkstationName)
	}
	assert.ElementsMatch(t, []string{"W-1F-001", "W-3F-001"}, names)

	// 选分公司本部 → 只出继承 4 楼的工位（覆盖楼层不再算本部）。
	resHQ, err := svc.List(ctx, requests.WorkstationListRequest{OrgID: wsoOrgHQ})
	require.NoError(t, err)
	listHQ, ok := resHQ.List.([]models.Workstation)
	require.True(t, ok)
	assert.Len(t, listHQ, 1, "分公司本部应只筛出继承楼层的工位")
	assert.Equal(t, "W-4F-001", listHQ[0].WorkstationName)
}

// TestFloorOrgOverride_WorkstationStatistics 统计口径与 List 一致。
func TestFloorOrgOverride_WorkstationStatistics(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	svc := NewWorkstationService(db)
	ctx := context.Background()

	st, err := svc.Statistics(ctx, map[string]interface{}{"orgId": wsoOrgWuhan})
	require.NoError(t, err)
	assert.Equal(t, int64(2), st.Total, "武汉中支统计应为 1/3 楼两个工位")

	stHQ, err := svc.Statistics(ctx, map[string]interface{}{"orgId": wsoOrgHQ})
	require.NoError(t, err)
	assert.Equal(t, int64(1), stHQ.Total, "分公司本部统计应为继承楼层一个工位")
}

// TestFloorOrgOverride_WorkstationSearchOptions 工位下拉同口径。
func TestFloorOrgOverride_WorkstationSearchOptions(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	svc := NewWorkstationService(db)
	ctx := context.Background()

	opts, err := svc.SearchWorkstationOptions(ctx, requests.WorkstationListRequest{OrgID: wsoOrgWuhan})
	require.NoError(t, err)
	assert.Len(t, opts, 2, "工位下拉按武汉中支应只出 1/3 楼工位")
}

// TestFloorOrgOverride_FloorListAndSearchOptions 楼层列表/下拉按覆盖机构筛选。
// 子树口径（审计 P1-3）：orgId 先展开「部门+子孙」再 IN，与楼宇/工位一致。
func TestFloorOrgOverride_FloorListAndSearchOptions(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	res, err := svc.List(ctx, map[string]interface{}{"orgId": wsoOrgWuhan, "current": 1, "pageSize": 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "楼层列表选武汉中支应只出 1/3 楼")

	opts, err := svc.SearchFloorOptions(ctx, map[string]interface{}{"orgId": wsoOrgWuhan})
	require.NoError(t, err)
	assert.Len(t, opts, 2, "楼层下拉选武汉中支应只出 1/3 楼")

	// 继承语义：选分公司本部仍只出 4 楼。
	resHQ, err := svc.List(ctx, map[string]interface{}{"orgId": wsoOrgHQ, "current": 1, "pageSize": 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resHQ.Total, "分公司本部应只出继承的 4 楼")

	optsHQ, err := svc.SearchFloorOptions(ctx, map[string]interface{}{"orgId": wsoOrgHQ})
	require.NoError(t, err)
	assert.Len(t, optsHQ, 1, "楼层下拉选分公司本部应只出 4 楼")
}

// TestFloorOrgOverride_BuildingOptions 楼宇下拉：因含覆盖楼层，武汉中支也能拿到浙商大厦。
func TestFloorOrgOverride_BuildingOptions(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	opts, err := svc.SearchBuildingOptions(ctx, map[string]interface{}{"orgId": wsoOrgWuhan})
	require.NoError(t, err)
	require.Len(t, opts, 1, "浙商大厦含 1/3 楼覆盖到武汉中支，应出现在该机构楼宇下拉")
	assert.Equal(t, "浙商大厦", opts[0].Label)

	// 本部自身命中，且不应因 EXISTS 子查询产生重复行。
	optsHQ, err := svc.SearchBuildingOptions(ctx, map[string]interface{}{"orgId": wsoOrgHQ})
	require.NoError(t, err)
	assert.Len(t, optsHQ, 1, "EXISTS 不应让楼宇重复")

	// List 路径同口径。
	res, err := svc.List(ctx, map[string]interface{}{"orgId": wsoOrgWuhan, "current": 1, "pageSize": 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total, "楼宇 List 选武汉中支也应拿到浙商大厦")
}

// TestFloorOrgOverride_EmptyStringInheritsBuilding 空串覆盖值等同继承（表单清空写入空串）。
func TestFloorOrgOverride_EmptyStringInheritsBuilding(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	// 把 1 楼的覆盖值改成空串，模拟表单清空。
	require.NoError(t, db.Exec(`UPDATE ops_floors SET org_id = '' WHERE id = ?`, wsoFloorWuhan).Error)

	svc := NewWorkstationService(db)
	ctx := context.Background()

	// 空串应回退到楼宇 org_id(分公司本部)，而不是被武汉中支筛走。
	res, err := svc.List(ctx, requests.WorkstationListRequest{OrgID: wsoOrgHQ})
	require.NoError(t, err)
	list, ok := res.List.([]models.Workstation)
	require.True(t, ok)
	assert.Len(t, list, 2, "空串覆盖值应继承楼宇(本部)：4 楼 + 1 楼")

	resWH, err := svc.List(ctx, requests.WorkstationListRequest{OrgID: wsoOrgWuhan})
	require.NoError(t, err)
	listWH, ok := resWH.List.([]models.Workstation)
	require.True(t, ok)
	assert.Len(t, listWH, 1, "空串楼层不应再算武汉中支")
	assert.Equal(t, "W-3F-001", listWH[0].WorkstationName)
}
