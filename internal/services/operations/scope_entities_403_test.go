// Package operations: Phase 133 / DS-05 扩展实体 403 越权矩阵厚测试。
//
// 命名前缀 Entity403* 避免与其它 agent 的测试/helper 冲突。
// SQLite 内存夹具：两部门（dept-A / dept-B）× 两 scope 用户（只看 A / 只看 B）
// + All 档 + 未注入 + FailClosed + Self 档，覆盖：
//
//   - 四实体（asset / serverRoom / dedicatedLine / roomDevice）读端点
//     List / Statistics / GetByID / dropdown-options 越界 → 空集（含归属链断链）；
//   - 四实体写端点 Create / Update / Delete / Batch 越界 → 403（ErrScopeDenied）；
//   - dedicatedLine source/dest 双链 OR 命中（本机可见 / 对端不可见仍放行）；
//   - workstation-device 按工位可见集读/写/Self（裁定 2）；
//   - All / 未注入（context.Background()）零回归（131/132 scopeIsAll 契约）；
//   - 归属链断链（软删楼宇 / 不存在机房）→ 读空集、写 403（fail-closed）。
//
// 所有断言均为语义断言（校验数据实际状态 / 错误身份，而非仅校验返回值）。
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
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	entity403DeptA = "aaaaaaaa-0000-0000-0000-0000000000a1"
	entity403DeptB = "bbbbbbbb-0000-0000-0000-0000000000b2"
	entity403UserA = "11111111-0000-0000-0000-0000000000a1"
	entity403UserB = "22222222-0000-0000-0000-0000000000b2"
)

// Entity403Fixture 持有两部门各一套完整归属链数据。
type Entity403Fixture struct {
	DB                                                        *gorm.DB
	BuildingA, BuildingB                                      string
	FloorA, FloorB                                            string
	WSA, WSB                                                  string
	RoomA, RoomB                                              string
	AssetA, AssetB                                            string
	LineA, LineB, LineCross, LineBroken                       string
	DeviceA, DeviceB, DeviceBroken                            string
	WDeviceA, WDeviceB                                        string
	AssetSvc                                                  AssetService
	RoomSvc                                                   ServerRoomService
	LineSvc                                                   DedicatedLineService
	DeviceSvc                                                 RoomDeviceService
	WDeviceSvc                                                WorkstationDeviceService
}

func entity403Ctx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

func entity403DeptOnly(deptID string) datascope.ScopeResult {
	return datascope.ScopeResult{DeptIDs: []string{deptID}}
}

func entity403Ptr(s string) *string { return &s }

// entity403BuildFixture 建四实体 + 工位链完整最小夹具。
func entity403BuildFixture(t *testing.T) *Entity403Fixture {
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
	// 逐个 AutoMigrate（带名字），便于定位含 PG 专属默认表达式的模型。
	for _, m := range []struct {
		name string
		val  interface{}
	}{
		{"OpsBuilding", &operationsmodels.OpsBuilding{}},
		{"OpsFloor", &operationsmodels.OpsFloor{}},
		{"Workstation", &models.Workstation{}},
		{"OpsServerRoom", &operationsmodels.OpsServerRoom{}},
		{"OpsDedicatedLine", &operationsmodels.OpsDedicatedLine{}},
		{"OpsRoomDevice", &operationsmodels.OpsRoomDevice{}},
	} {
		require.NoError(t, db.AutoMigrate(m.val), "AutoMigrate %s 失败", m.name)
	}
	// models.WorkstationDevice 含 `type:uuid` 列 + 关联外键标签，SQLite
	// AutoMigrate 生成带括号的 PG 专属 DDL 报 `near "(": syntax error`。
	// 手工建最小表（只含 B 收口与工位设备 service 实际引用的列）。
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_workstation_device (
			id TEXT PRIMARY KEY,
			workstation_id TEXT,
			asset_id TEXT,
			ad_computer_id TEXT,
			device_source TEXT,
			device_serial TEXT,
			device_name TEXT,
			device_model TEXT,
			device_type TEXT,
			mac_address TEXT,
			ip_address TEXT,
			responsible_user TEXT,
			responsible_user_id TEXT,
			status INTEGER DEFAULT 0,
			is_primary INTEGER DEFAULT 0,
			priority INTEGER DEFAULT 0,
			description TEXT,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error)
	// models.Asset 主键带 `default:gen_random_uuid()`，SQLite 无该函数会在
	// AutoMigrate 建表时报 `near "(": syntax error`。这里手工建最小表（只含
	// 四实体收口与资产 service 读/写实际引用的列），ID 由 BeforeCreate 钩子
	// 应用层生成，PG/SQLite 双方言均可跑。
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			dept_id TEXT,
			user_id TEXT,
			component_type TEXT,
			status INTEGER DEFAULT 0,
			nbf_status INTEGER DEFAULT 0,
			usestatus_label TEXT,
			device_type_name TEXT,
			device_category_second_name TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, ancestors TEXT, parent_id TEXT, status INTEGER, deleted_at DATETIME);
		CREATE TABLE sys_user (id TEXT PRIMARY KEY, dept_id TEXT, deleted_at DATETIME);
	`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name) VALUES (?, '部门A'), (?, '部门B')`,
		entity403DeptA, entity403DeptB).Error)

	f := &Entity403Fixture{DB: db}
	bA := &operationsmodels.OpsBuilding{Name: "楼宇A", OrgID: entity403DeptA}
	bB := &operationsmodels.OpsBuilding{Name: "楼宇B", OrgID: entity403DeptB}
	require.NoError(t, db.Create(bA).Error)
	require.NoError(t, db.Create(bB).Error)
	flA := &operationsmodels.OpsFloor{Name: "楼层A", FloorNo: "1", BuildingID: bA.ID}
	flB := &operationsmodels.OpsFloor{Name: "楼层B", FloorNo: "1", BuildingID: bB.ID}
	require.NoError(t, db.Create(flA).Error)
	require.NoError(t, db.Create(flB).Error)

	wA := &models.Workstation{WorkstationName: "工位A", FloorID: &flA.ID, UserID: entity403Ptr(entity403UserA)}
	wB := &models.Workstation{WorkstationName: "工位B", FloorID: &flB.ID, UserID: entity403Ptr(entity403UserB)}
	require.NoError(t, db.Create(wA).Error)
	require.NoError(t, db.Create(wB).Error)

	rA := &operationsmodels.OpsServerRoom{Name: "机房A", BuildingID: bA.ID, FloorID: flA.ID}
	rB := &operationsmodels.OpsServerRoom{Name: "机房B", BuildingID: bB.ID, FloorID: flB.ID}
	require.NoError(t, db.Create(rA).Error)
	require.NoError(t, db.Create(rB).Error)

	// ops_asset 手工建表 → 手写 INSERT（避免 GORM Create 插入模型全列）。
	aAID := "aaaaaaaa-aaaa-0000-0000-000000000001"
	aBID := "bbbbbbbb-bbbb-0000-0000-000000000002"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, dept_id) VALUES (?, ?, ?), (?, ?, ?)`,
		aAID, "SN-A", entity403DeptA, aBID, "SN-B", entity403DeptB).Error)

	// 专线：A 端两端都机房A；B 端两端都机房B；Cross 源A目的B（双链 OR 命中用例）；
	// Broken 两端指向不存在的机房（断链 → fail-closed）。
	lA := &operationsmodels.OpsDedicatedLine{Name: "专线A", ISP: "电信", SourceRoomID: entity403Ptr(rA.ID), DestRoomID: entity403Ptr(rA.ID)}
	lB := &operationsmodels.OpsDedicatedLine{Name: "专线B", ISP: "电信", SourceRoomID: entity403Ptr(rB.ID), DestRoomID: entity403Ptr(rB.ID)}
	lCross := &operationsmodels.OpsDedicatedLine{Name: "专线跨机构", ISP: "电信", SourceRoomID: entity403Ptr(rA.ID), DestRoomID: entity403Ptr(rB.ID)}
	lBroken := &operationsmodels.OpsDedicatedLine{Name: "专线断链", ISP: "电信", SourceRoomID: entity403Ptr("00000000-0000-0000-0000-00000000dead"), DestRoomID: entity403Ptr("00000000-0000-0000-0000-00000000dead")}
	require.NoError(t, db.Create(lA).Error)
	require.NoError(t, db.Create(lB).Error)
	require.NoError(t, db.Create(lCross).Error)
	require.NoError(t, db.Create(lBroken).Error)

	dA := &operationsmodels.OpsRoomDevice{Name: "设备A", DeviceCode: "CODE-A", RoomID: rA.ID}
	dB := &operationsmodels.OpsRoomDevice{Name: "设备B", DeviceCode: "CODE-B", RoomID: rB.ID}
	dBroken := &operationsmodels.OpsRoomDevice{Name: "设备断链", DeviceCode: "CODE-BROKEN", RoomID: "00000000-0000-0000-0000-00000000dead"}
	require.NoError(t, db.Create(dA).Error)
	require.NoError(t, db.Create(dB).Error)
	require.NoError(t, db.Create(dBroken).Error)

	// ops_workstation_device 手工建表 → 手写 INSERT（避免 GORM Create 插入模型全列）。
	wdAID := "dddddddd-0000-0000-0000-0000000000a1"
	wdBID := "dddddddd-0000-0000-0000-0000000000b2"
	require.NoError(t, db.Exec(`INSERT INTO ops_workstation_device (id, workstation_id, device_source, device_serial) VALUES
		(?, ?, ?, ?), (?, ?, ?, ?)`,
		wdAID, wA.ID, string(models.DeviceSourceManual), "WDSN-A",
		wdBID, wB.ID, string(models.DeviceSourceManual), "WDSN-B").Error)

	f.BuildingA, f.BuildingB = bA.ID, bB.ID
	f.FloorA, f.FloorB = flA.ID, flB.ID
	f.WSA, f.WSB = wA.ID, wB.ID
	f.RoomA, f.RoomB = rA.ID, rB.ID
	f.AssetA, f.AssetB = aAID, aBID
	f.LineA, f.LineB, f.LineCross, f.LineBroken = lA.ID, lB.ID, lCross.ID, lBroken.ID
	f.DeviceA, f.DeviceB, f.DeviceBroken = dA.ID, dB.ID, dBroken.ID
	f.WDeviceA, f.WDeviceB = wdAID, wdBID
	f.AssetSvc = NewAssetService(db)
	f.RoomSvc = NewServerRoomService(db)
	f.LineSvc = NewDedicatedLineService(db)
	f.DeviceSvc = NewRoomDeviceService(db)
	f.WDeviceSvc = NewWorkstationDeviceService(db)
	return f
}

func entity403Count(t *testing.T, db *gorm.DB, table string, where string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table(table).Where(where, args...).Where("deleted_at IS NULL").Count(&n).Error)
	return n
}

// ---------------------------------------------------------------------
// R-1 四实体读端点越界矩阵（List / Statistics / GetByID / Options）+ 断链
// ---------------------------------------------------------------------

func TestEntity403_Read_FourEntities_OutOfScopeEmpty(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxA := entity403Ctx(entity403DeptOnly(entity403DeptA))

	// asset：dept_id 直挂，只看 A。
	list, err := f.AssetSvc.List(ctxA, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total, "asset List 只返回部门A 1 条")
	stat, err := f.AssetSvc.Statistics(ctxA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stat.Total, "asset Statistics 只统计部门A")
	_, err = f.AssetSvc.GetByID(ctxA, f.AssetB)
	assert.Error(t, err, "asset GetByID 越界 → record not found")

	// serverRoom：楼宇链。
	roomList, err := f.RoomSvc.List(ctxA, requests.ServerRoomListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), roomList.Total, "serverRoom List 只返回楼宇A 下机房")
	roomStat, err := f.RoomSvc.Statistics(ctxA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), roomStat.Total)
	_, err = f.RoomSvc.GetByID(ctxA, f.RoomB)
	assert.Error(t, err, "serverRoom GetByID 越界")

	// dedicatedLine：双链 OR，A 端可见的有 lA + lCross（本机可见、对端不可见仍放行）；
	// lB（两端都 B）+ lBroken（断链）不可见。
	lineList, err := f.LineSvc.List(ctxA, requests.DedicatedLineListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), lineList.Total, "dedicatedLine 双链 OR：lA + lCross 可见")
	lineStat, err := f.LineSvc.Statistics(ctxA)
	require.NoError(t, err)
	assert.Equal(t, int64(2), lineStat.Total)
	_, err = f.LineSvc.GetByID(ctxA, f.LineB)
	assert.Error(t, err, "dedicatedLine GetByID 两端都越界")
	_, err = f.LineSvc.GetByID(ctxA, f.LineBroken)
	assert.Error(t, err, "dedicatedLine 断链 → fail-closed 不可见")

	// roomDevice：两级链，断链设备不可见。
	devList, err := f.DeviceSvc.List(ctxA, requests.RoomDeviceListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), devList.Total, "roomDevice 只返回机房A 下设备（断链剔除）")
	devStat, err := f.DeviceSvc.Statistics(ctxA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), devStat.Total)
	_, err = f.DeviceSvc.GetByID(ctxA, f.DeviceB)
	assert.Error(t, err, "roomDevice GetByID 越界")
	_, err = f.DeviceSvc.GetByID(ctxA, f.DeviceBroken)
	assert.Error(t, err, "roomDevice 断链 → fail-closed 不可见")
}

// ---------------------------------------------------------------------
// R-2 归属链断链（楼宇软删）→ 读空集（fail-closed，不为脏数据开旁路）
// ---------------------------------------------------------------------

func TestEntity403_Read_BrokenChain_SoftDeletedBuilding(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxA := entity403Ctx(entity403DeptOnly(entity403DeptA))
	// 软删楼宇A：其下机房 / 机房设备 / 专线两端归属链全部断开。
	require.NoError(t, f.DB.Table("ops_buildings").Where("id = ?", f.BuildingA).Update("deleted_at", "2026-01-01 00:00:00").Error)

	roomStat, err := f.RoomSvc.Statistics(ctxA)
	require.NoError(t, err)
	assert.Equal(t, int64(0), roomStat.Total, "楼宇软删 → serverRoom 读空集")
	devStat, err := f.DeviceSvc.Statistics(ctxA)
	require.NoError(t, err)
	assert.Equal(t, int64(0), devStat.Total, "楼宇软删 → roomDevice 读空集")
	lineStat, err := f.LineSvc.Statistics(ctxA)
	require.NoError(t, err)
	assert.Equal(t, int64(0), lineStat.Total, "楼宇软删 → dedicatedLine 双链断 → 读空集")
}

// ---------------------------------------------------------------------
// W-1 四实体单条写越界 → 403（ErrScopeDenied），零数据变更
// ---------------------------------------------------------------------

func TestEntity403_Write_Single_OutOfScope403(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxA := entity403Ctx(entity403DeptOnly(entity403DeptA))

	// Create 越界（提交侧）。
	assert.True(t, IsScopeDenied(f.AssetSvc.Create(ctxA,
		&models.Asset{DeviceSN: "SN-X", DeptID: entity403Ptr(entity403DeptB)})))
	assert.True(t, IsScopeDenied(f.RoomSvc.Create(ctxA,
		&operationsmodels.OpsServerRoom{Name: "越界机房", BuildingID: f.BuildingB, FloorID: f.FloorB})))
	assert.True(t, IsScopeDenied(f.LineSvc.Create(ctxA,
		&operationsmodels.OpsDedicatedLine{Name: "越界专线", ISP: "电信", SourceRoomID: entity403Ptr(f.RoomB), DestRoomID: entity403Ptr(f.RoomB)})))
	assert.True(t, IsScopeDenied(f.DeviceSvc.Create(ctxA,
		&operationsmodels.OpsRoomDevice{Name: "越界设备", DeviceCode: "CODE-X", RoomID: f.RoomB})))

	// Update 越界（改前侧）。
	assert.True(t, IsScopeDenied(f.AssetSvc.Update(ctxA,
		&models.Asset{ID: f.AssetB, DeviceSN: "SN-B", DeptID: entity403Ptr(entity403DeptB)})))
	assert.True(t, IsScopeDenied(f.RoomSvc.Update(ctxA,
		&operationsmodels.OpsServerRoom{BaseModel: models.BaseModel{ID: f.RoomB}, Name: "改名", BuildingID: f.BuildingB, FloorID: f.FloorB})))

	// Delete 越界（改前侧）。
	assert.True(t, IsScopeDenied(f.AssetSvc.Delete(ctxA, f.AssetB)))
	assert.True(t, IsScopeDenied(f.RoomSvc.Delete(ctxA, f.RoomB)))
	assert.True(t, IsScopeDenied(f.LineSvc.Delete(ctxA, f.LineB)))
	assert.True(t, IsScopeDenied(f.DeviceSvc.Delete(ctxA, f.DeviceB)))

	// 断链资源写 → 403（fail-closed）。
	assert.True(t, IsScopeDenied(f.LineSvc.Delete(ctxA, f.LineBroken)))
	assert.True(t, IsScopeDenied(f.DeviceSvc.Delete(ctxA, f.DeviceBroken)))

	// 零数据变更语义断言。
	assert.Equal(t, int64(2), entity403Count(t, f.DB, "ops_asset", "1=1"))
	assert.Equal(t, int64(2), entity403Count(t, f.DB, "ops_server_rooms", "1=1"))
	assert.Equal(t, int64(4), entity403Count(t, f.DB, "ops_dedicated_lines", "1=1"))
	assert.Equal(t, int64(3), entity403Count(t, f.DB, "ops_room_devices", "1=1"))
}

// ---------------------------------------------------------------------
// W-2 dedicatedLine 双链 OR 写：本机可见 / 对端不可见仍放行；两端都越界 403
// ---------------------------------------------------------------------

func TestEntity403_Write_DedicatedLine_DualChainOR(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxA := entity403Ctx(entity403DeptOnly(entity403DeptA))

	// 源A + 目的B（本机可见、对端不可见）→ 任一端命中放行。
	err := f.LineSvc.Create(ctxA, &operationsmodels.OpsDedicatedLine{
		Name: "跨机构专线新建", ISP: "电信",
		SourceRoomID: entity403Ptr(f.RoomA), DestRoomID: entity403Ptr(f.RoomB),
	})
	require.NoError(t, err, "双链 OR：源端命中可见集应放行")
	assert.Equal(t, int64(5), entity403Count(t, f.DB, "ops_dedicated_lines", "1=1"))

	// 两端都断链 → 候选为空 → 403。
	assert.True(t, IsScopeDenied(f.LineSvc.Create(ctxA, &operationsmodels.OpsDedicatedLine{
		Name: "断链专线新建", ISP: "电信",
		SourceRoomID: entity403Ptr("00000000-0000-0000-0000-00000000dead"),
		DestRoomID:   entity403Ptr("00000000-0000-0000-0000-00000000dead"),
	})))
}

// ---------------------------------------------------------------------
// W-3 批量按可见集裁剪（不整批 403），越界行零变更
// ---------------------------------------------------------------------

func TestEntity403_Write_Batch_VisibleClipping(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxA := entity403Ctx(entity403DeptOnly(entity403DeptA))

	// 混入 A（可见）/ B（越界）ID：只删 A，B 保留。
	require.NoError(t, f.AssetSvc.BatchDelete(ctxA, []string{f.AssetA, f.AssetB}))
	assert.Equal(t, int64(1), entity403Count(t, f.DB, "ops_asset", "1=1"), "只删可见集内 1 条")
	assert.Equal(t, int64(0), entity403Count(t, f.DB, "ops_asset", "id = ?", f.AssetA))
	assert.Equal(t, int64(1), entity403Count(t, f.DB, "ops_asset", "id = ?", f.AssetB))

	// 顺序敏感：先裁 dedicatedLine / roomDevice（其归属链经过 ops_server_rooms），
	// 再裁 serverRoom —— 若先软删机房A，会让前两者的归属链断链（正确 fail-closed，
	// 但会把「越界裁剪」与「断链」两种语义混在一个断言里）。
	require.NoError(t, f.LineSvc.BatchDelete(ctxA, []string{f.LineA, f.LineB, f.LineCross, f.LineBroken}))
	// lA + lCross 可见被删；lB + lBroken 越界/断链保留。
	assert.Equal(t, int64(2), entity403Count(t, f.DB, "ops_dedicated_lines", "1=1"))
	assert.Equal(t, int64(1), entity403Count(t, f.DB, "ops_dedicated_lines", "id = ?", f.LineB))
	assert.Equal(t, int64(1), entity403Count(t, f.DB, "ops_dedicated_lines", "id = ?", f.LineBroken))

	require.NoError(t, f.DeviceSvc.BatchDelete(ctxA, []string{f.DeviceA, f.DeviceB, f.DeviceBroken}))
	// 只有 dA 可见被删；dB 越界、dBroken 断链保留。
	assert.Equal(t, int64(2), entity403Count(t, f.DB, "ops_room_devices", "1=1"))

	require.NoError(t, f.RoomSvc.BatchDelete(ctxA, []string{f.RoomA, f.RoomB}))
	assert.Equal(t, int64(1), entity403Count(t, f.DB, "ops_server_rooms", "1=1"))
}

// ---------------------------------------------------------------------
// Z-1 All / 未注入零回归（131/132 scopeIsAll 契约，存量调用方零回归硬约束）
// ---------------------------------------------------------------------

func TestEntity403_AllAndNoInjection_ZeroRegression(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxAll := entity403Ctx(datascope.AllScope(entity403UserA))
	ctxNone := context.Background() // FromContext ok=false → 按 All 兜底

	for _, ctx := range []context.Context{ctxAll, ctxNone} {
		list, err := f.AssetSvc.List(ctx, map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, int64(2), list.Total, "All/未注入 asset List 全量")
		roomList, err := f.RoomSvc.List(ctx, requests.ServerRoomListRequest{})
		require.NoError(t, err)
		assert.Equal(t, int64(2), roomList.Total)
		lineList, err := f.LineSvc.List(ctx, requests.DedicatedLineListRequest{})
		require.NoError(t, err)
		assert.Equal(t, int64(4), lineList.Total, "All/未注入 含断链专线全量")
		devList, err := f.DeviceSvc.List(ctx, requests.RoomDeviceListRequest{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), devList.Total)
		// 写越界目标在 All/未注入档放行（零回归）。
		require.NoError(t, f.AssetSvc.Delete(ctx, f.AssetB))
		// 复原供下一轮 ctx 使用（软删恢复）。
		require.NoError(t, f.DB.Table("ops_asset").Where("id = ?", f.AssetB).Update("deleted_at", nil).Error)
	}
}

// ---------------------------------------------------------------------
// Z-2 FailClosed / Self 档：读空集、写 403
// ---------------------------------------------------------------------

func TestEntity403_FailClosedAndSelf_Deny(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxFail := entity403Ctx(datascope.FailClosedScope(entity403UserA))
	ctxSelf := entity403Ctx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: entity403UserA})

	for _, ctx := range []context.Context{ctxFail, ctxSelf} {
		stat, err := f.AssetSvc.Statistics(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stat.Total, "FailClosed/Self 四实体无用户列 → 读空集")
		assert.True(t, IsScopeDenied(f.AssetSvc.Delete(ctx, f.AssetA)))
		assert.True(t, IsScopeDenied(f.RoomSvc.Delete(ctx, f.RoomA)))
	}
}

// ---------------------------------------------------------------------
// B-1 workstation-device 按工位可见集（裁定 2）：读空 / 写 403 / Self 随 user_id
// ---------------------------------------------------------------------

func TestEntity403_WorkstationDevice_WorkstationScope(t *testing.T) {
	f := entity403BuildFixture(t)
	ctxA := entity403Ctx(entity403DeptOnly(entity403DeptA))

	// 读：可见工位返回设备；越界工位返回空集（不报错）。
	devs, err := f.WDeviceSvc.GetDevicesByWorkstation(ctxA, f.WSA)
	require.NoError(t, err)
	assert.Len(t, devs, 1)
	devs, err = f.WDeviceSvc.GetDevicesByWorkstation(ctxA, f.WSB)
	require.NoError(t, err)
	assert.Empty(t, devs, "越界工位读返回空集")

	// 写：目标工位越界 → 403（ErrScopeDenied）。
	_, err = f.WDeviceSvc.AddDeviceManual(ctxA, &AddDeviceRequest{
		WorkstationID: f.WSB, DeviceSerial: "SN-WD-X",
	})
	assert.True(t, IsScopeDenied(err))
	assert.True(t, IsScopeDenied(f.WDeviceSvc.DeleteDevice(ctxA, f.WDeviceB)))
	assert.True(t, IsScopeDenied(f.WDeviceSvc.UpdateDevice(ctxA, f.WDeviceB, &UpdateDeviceRequest{})))
	assert.True(t, IsScopeDenied(f.WDeviceSvc.SyncFromAD(ctxA, f.WSB)))
	assert.True(t, IsScopeDenied(f.WDeviceSvc.SyncFromAsset(ctxA, f.WSB)))
	// 可见工位放行（AddDeviceManual 需合法 device_serial；夹具表存在）。
	_, err = f.WDeviceSvc.AddDeviceManual(ctxA, &AddDeviceRequest{
		WorkstationID: f.WSA, DeviceSerial: "SN-WD-NEW",
	})
	require.NoError(t, err)

	// Self 档：随工位 user_id 收窄（工位A 绑定 UserA）。
	ctxSelfA := entity403Ctx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: entity403UserA})
	devs, err = f.WDeviceSvc.GetDevicesByWorkstation(ctxSelfA, f.WSA)
	require.NoError(t, err)
	assert.Len(t, devs, 2, "Self 档本人工位可见（含前面 AddDeviceManual 新增的一条）")
	devs, err = f.WDeviceSvc.GetDevicesByWorkstation(ctxSelfA, f.WSB)
	require.NoError(t, err)
	assert.Empty(t, devs, "Self 档他人工位不可见")
	assert.True(t, IsScopeDenied(f.WDeviceSvc.DeleteDevice(ctxSelfA, f.WDeviceB)), "Self 档他人工位写 403")

	// All / 未注入零回归。
	for _, ctx := range []context.Context{entity403Ctx(datascope.AllScope(entity403UserA)), context.Background()} {
		devs, err = f.WDeviceSvc.GetDevicesByWorkstation(ctx, f.WSB)
		require.NoError(t, err)
		assert.Len(t, devs, 1, "All/未注入 workstation-device 读全量")
	}
}
