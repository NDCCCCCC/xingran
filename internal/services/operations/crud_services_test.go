package operations

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// =====================================================================
// Phase 74-07: wall/door/server_room/dedicated_line/infopoint CRUD 测试。
// 共享 AutoMigrate sqlite 建表 helper。
// =====================================================================

// newCRUDTestDB 创建包含 operations 全家族表的内存 sqlite。
func newCRUDTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{},
		&operationsmodels.OpsFloor{},
		&operationsmodels.Wall{},
		&operationsmodels.Door{},
		&operationsmodels.OpsServerRoom{},
		&operationsmodels.OpsDedicatedLine{},
		&operationsmodels.OpsInfoPoint{},
		&operationsmodels.OpsRoomPhoto{},
		&models.Department{},
		&models.Workstation{}, // infopoint populateRedundantFields 查询 sys_workstation
	))
	return db
}

// seedBuildingFloor 建一楼一楼层，返回 buildingID/floorID。
func seedBuildingFloor(t *testing.T, db *gorm.DB, name string) (string, string) {
	t.Helper()
	b := &operationsmodels.OpsBuilding{Name: name, Address: "addr", Level: 2, OrgID: "d1"}
	require.NoError(t, db.Create(b).Error)
	f := &operationsmodels.OpsFloor{Name: name + "-F1", FloorNo: "1", BuildingID: b.ID}
	require.NoError(t, db.Create(f).Error)
	return b.ID, f.ID
}

func TestWallService_CRUD(t *testing.T) {
	db := newCRUDTestDB(t)
	_, floorID := seedBuildingFloor(t, db, "b1")
	svc := NewWallService(db)
	ctx := context.Background()

	// 楼层不存在 → 校验失败
	err := svc.Create(ctx, &operationsmodels.Wall{FloorID: "missing", Type: operationsmodels.WallTypeStraight, Points: "[]"})
	require.Error(t, err)

	// Create / GetByID
	wall := &operationsmodels.Wall{FloorID: floorID, Type: operationsmodels.WallTypeCurved, Points: `[{"x":1}]`, Thickness: 12}
	require.NoError(t, svc.Create(ctx, wall))
	got, err := svc.GetByID(ctx, wall.ID)
	require.NoError(t, err)
	assert.Equal(t, operationsmodels.WallTypeCurved, got.Type)
	assert.Equal(t, 12, got.Thickness)

	// Update
	got.Thickness = 20
	require.NoError(t, svc.Update(ctx, got))
	got, _ = svc.GetByID(ctx, wall.ID)
	assert.Equal(t, 20, got.Thickness)
	require.Error(t, svc.Update(ctx, &operationsmodels.Wall{FloorID: "missing", Type: "straight", Points: "[]"}))

	// List 过滤 + 分页
	require.NoError(t, svc.Create(ctx, &operationsmodels.Wall{FloorID: floorID, Type: operationsmodels.WallTypeStraight, Points: "[]"}))
	page, err := svc.List(ctx, requests.WallListRequest{FloorID: floorID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	page, err = svc.List(ctx, requests.WallListRequest{FloorID: floorID, WallType: "curved"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	asc := true
	page, err = svc.List(ctx, requests.WallListRequest{
		FloorID: floorID,
		PaginationParams: requests.PaginationParams{BaseListRequest: base.BaseListRequest{
			OrderByColumn: "createdAt", IsAsc: &asc,
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// BatchDelete（空列表 no-op）
	require.NoError(t, svc.BatchDelete(ctx, nil))
	require.NoError(t, svc.BatchDelete(ctx, []string{wall.ID}))
	page, err = svc.List(ctx, requests.WallListRequest{FloorID: floorID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// Delete / GetByID 缺失
	require.NoError(t, svc.Delete(ctx, got.ID))
	_, err = svc.GetByID(ctx, got.ID)
	require.Error(t, err)
}

func TestDoorService_CRUD(t *testing.T) {
	db := newCRUDTestDB(t)
	_, floorID := seedBuildingFloor(t, db, "b1")
	svc := NewDoorService(db)
	ctx := context.Background()

	// 楼层不存在 / 墙体不存在
	require.Error(t, svc.Create(ctx, &operationsmodels.Door{FloorID: "missing", Type: "single", Direction: "left", Position: "{}"}))
	wall := &operationsmodels.Wall{FloorID: floorID, Type: operationsmodels.WallTypeStraight, Points: "[]"}
	require.NoError(t, db.Create(wall).Error)
	badWall := "ghost-wall"
	require.Error(t, svc.Create(ctx, &operationsmodels.Door{FloorID: floorID, WallID: &badWall, Type: "single", Direction: "left", Position: "{}"}))

	// Create（含合法 WallID）
	door := &operationsmodels.Door{FloorID: floorID, WallID: &wall.ID, Type: "single", Direction: "left", Position: `{"x":1}`, Width: 90}
	require.NoError(t, svc.Create(ctx, door))
	got, err := svc.GetByID(ctx, door.ID)
	require.NoError(t, err)
	require.NotNil(t, got.WallID)
	assert.Equal(t, wall.ID, *got.WallID)

	// Update
	got.Width = 100
	require.NoError(t, svc.Update(ctx, got))

	// List 过滤
	require.NoError(t, svc.Create(ctx, &operationsmodels.Door{FloorID: floorID, Type: "double", Direction: "right", Position: "{}"}))
	page, err := svc.List(ctx, requests.DoorListRequest{FloorID: floorID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	page, err = svc.List(ctx, requests.DoorListRequest{FloorID: floorID, DoorType: "double"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// BatchDelete + Delete
	require.NoError(t, svc.BatchDelete(ctx, []string{door.ID}))
	require.NoError(t, svc.Delete(ctx, door.ID))
	_, err = svc.GetByID(ctx, door.ID)
	require.Error(t, err)
}

func TestServerRoomService_CRUD(t *testing.T) {
	db := newCRUDTestDB(t)
	buildingID, floorID := seedBuildingFloor(t, db, "b1")
	svc := NewServerRoomService(db)
	ctx := context.Background()

	// 楼层校验失败
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsServerRoom{Name: "r", BuildingID: buildingID, FloorID: "missing"}))

	room := &operationsmodels.OpsServerRoom{Name: "机房A", BuildingID: buildingID, FloorID: floorID}
	require.NoError(t, svc.Create(ctx, room))
	got, err := svc.GetByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, "机房A", got.Name)

	got.Name = "机房A2"
	require.NoError(t, svc.Update(ctx, got))

	// List：JOIN 楼宇/楼层名 + name/building/floor/status 过滤
	require.NoError(t, svc.Create(ctx, &operationsmodels.OpsServerRoom{Name: "机房B", BuildingID: buildingID, FloorID: floorID, Status: operationsmodels.RoomStatusStopped}))
	stopped := 1
	page, err := svc.List(ctx, requests.ServerRoomListRequest{Name: "机房A"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Equal(t, "机房A2", page.List.([]operationsmodels.OpsServerRoom)[0].Name)

	page, err = svc.List(ctx, requests.ServerRoomListRequest{BuildingID: buildingID, FloorID: floorID, StatusRequest: requests.StatusRequest{Status: &stopped}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// OrgID 过滤：sys_dept 命中
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status) VALUES ('d1','总部','D1','',0)`).Error)
	page, err = svc.List(ctx, requests.ServerRoomListRequest{OrgID: "d1"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// OrgID 无命中 → 空
	page, err = svc.List(ctx, requests.ServerRoomListRequest{OrgID: "nope"})
	require.NoError(t, err)
	assert.Zero(t, page.Total)

	// Statistics
	stats, err := svc.Statistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Total)
	assert.Equal(t, int64(1), stats.Active)
	assert.Equal(t, int64(1), stats.Inactive)

	// SearchOptions：name 模糊 + status
	opts, err := svc.SearchServerRoomOptions(ctx, map[string]interface{}{"name": "机房A"})
	require.NoError(t, err)
	require.Len(t, opts, 1)
	opts, err = svc.SearchServerRoomOptions(ctx, map[string]interface{}{"status": 1, "buildingId": buildingID})
	require.NoError(t, err)
	assert.Len(t, opts, 1)
	opts, err = svc.SearchServerRoomOptions(ctx, map[string]interface{}{"orgId": "d1"})
	require.NoError(t, err)
	assert.Len(t, opts, 2)

	// BatchDelete / Delete
	require.NoError(t, svc.BatchDelete(ctx, []string{room.ID}))
	require.NoError(t, svc.Delete(ctx, room.ID))
	_, err = svc.GetByID(ctx, room.ID)
	require.Error(t, err)
}

func TestDedicatedLineService_CRUD(t *testing.T) {
	db := newCRUDTestDB(t)
	svc := NewDedicatedLineService(db)
	ctx := context.Background()

	line := &operationsmodels.OpsDedicatedLine{
		Name: "专线-1", LineType: "mpls", ISP: "电信",
		SourceRoomID: strPtr("r1"), SourceRoomName: strPtr("机房1"),
		DestRoomID: strPtr("r2"), DestRoomName: strPtr("机房2"),
	}
	require.NoError(t, svc.Create(ctx, line))
	got, err := svc.GetByID(ctx, line.ID)
	require.NoError(t, err)
	assert.Equal(t, "电信", got.ISP)

	got.ISP = "联通"
	require.NoError(t, svc.Update(ctx, got))

	// List 过滤矩阵
	require.NoError(t, svc.Create(ctx, &operationsmodels.OpsDedicatedLine{
		Name: "专线-2", LineType: "sdh", ISP: "移动", Status: 1,
		SourceDeviceName: strPtr("dev-a"), DestDeviceName: strPtr("dev-b"),
		CarrierContactName: strPtr("张三"),
	}))
	page, err := svc.List(ctx, requests.DedicatedLineListRequest{Name: "专线-1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{LineType: "sdh"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{ISP: "联通"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{SourceRoomId: "r1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{SourceRoomName: "机房1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{DestRoomId: "r2"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{DestRoomName: "机房2"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{SourceDeviceName: "dev-a"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{DestDeviceName: "dev-b"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{CarrierContactName: "张三"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	stopped := 1
	page, err = svc.List(ctx, requests.DedicatedLineListRequest{StatusRequest: requests.StatusRequest{Status: &stopped}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// Statistics
	stats, err := svc.Statistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Total)

	// SearchOptions
	opts, err := svc.SearchDedicatedLineOptions(ctx, map[string]interface{}{"name": "专线-1"})
	require.NoError(t, err)
	assert.Len(t, opts, 1)

	// Delete / BatchDelete
	require.NoError(t, svc.Delete(ctx, line.ID))
	require.NoError(t, svc.BatchDelete(ctx, []string{line.ID}))
	_, err = svc.GetByID(ctx, line.ID)
	require.Error(t, err)
}

func TestInfoPointService_CRUD(t *testing.T) {
	db := newCRUDTestDB(t)
	svc := NewInfoPointService(db)
	ctx := context.Background()

	ip := &operationsmodels.OpsInfoPoint{Name: "信息点-1", InfoPointType: "network", WorkstationID: "w1"}
	require.NoError(t, svc.Create(ctx, ip))
	got, err := svc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	assert.Equal(t, "network", string(got.InfoPointType))

	got.Name = "信息点-1b"
	require.NoError(t, svc.Update(ctx, got))

	require.NoError(t, svc.Create(ctx, &operationsmodels.OpsInfoPoint{Name: "信息点-2", InfoPointType: "voice", WorkstationID: "w2", Status: 2}))

	page, err := svc.List(ctx, requests.InfoPointListRequest{Name: "信息点-1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.InfoPointListRequest{WorkstationID: "w1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.InfoPointListRequest{InfoPointType: "voice"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	// 旧字段兼容路径
	page, err = svc.List(ctx, requests.InfoPointListRequest{WorkID: "w1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.InfoPointListRequest{PointType: "voice"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// Statistics
	stats, err := svc.Statistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Total)

	// SearchOptions
	opts, err := svc.SearchInfoPointOptions(ctx, map[string]interface{}{"name": "信息点-1"})
	require.NoError(t, err)
	assert.Len(t, opts, 1)

	require.NoError(t, svc.Delete(ctx, ip.ID))
	require.NoError(t, svc.BatchDelete(ctx, []string{ip.ID}))
	_, err = svc.GetByID(ctx, ip.ID)
	require.Error(t, err)
}
