package operations

// =====================================================================
// Quick 261012-l7k: floorService.List 每行工位统计注入（单条 GROUP BY，零 N+1）。
//
// 状态语义锁定（sys_workstation.status 独立枚举，非全局 0=正常/1=停用惯例）：
//   0=空闲(未使用 WorkstationStatusAvailable) / 1=占用(已使用 WorkstationStatusOccupied)
//   / 2=维护(WorkstationStatusMaintain)。
//
// 覆盖：基本聚合 / 软删工位排除 / 空楼层 nil+JSON omitempty / 分页隔离。
// 复用 floor_service_crud_test.go 的 newFloor91TestDB（已 AutoMigrate sys_workstation）。
// =====================================================================

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// TestFloorWorkstationStats_BasicAggregation F1 挂 3 工位(0/1/1) → {3,2,1,0}；
// F2 挂 2 工位(2/1) → {2,1,0,1}。锁定状态枚举到统计桶的映射。
func TestFloorWorkstationStats_BasicAggregation(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "BWS1", "org-ws")
	f1 := &operationsmodels.OpsFloor{Name: "F-WS-1", FloorNo: "1", BuildingID: b.ID}
	f2 := &operationsmodels.OpsFloor{Name: "F-WS-2", FloorNo: "2", BuildingID: b.ID}
	require.NoError(t, svc.Create(ctx, f1))
	require.NoError(t, svc.Create(ctx, f2))

	// F1: 空闲 1 + 占用 2 → total=3 used=2 unused=1 maintenance=0
	for _, st := range []sysmodels.WorkstationStatus{
		sysmodels.WorkstationStatusAvailable,
		sysmodels.WorkstationStatusOccupied,
		sysmodels.WorkstationStatusOccupied,
	} {
		require.NoError(t, db.Create(&sysmodels.Workstation{WorkstationName: "f1-ws", FloorID: &f1.ID, Status: st}).Error)
	}
	// F2: 维护 1 + 占用 1 → total=2 used=1 unused=0 maintenance=1
	for _, st := range []sysmodels.WorkstationStatus{
		sysmodels.WorkstationStatusMaintain,
		sysmodels.WorkstationStatusOccupied,
	} {
		require.NoError(t, db.Create(&sysmodels.Workstation{WorkstationName: "f2-ws", FloorID: &f2.ID, Status: st}).Error)
	}

	page, err := svc.List(ctx, map[string]interface{}{})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsFloor)
	require.Len(t, list, 2)

	byName := make(map[string]operationsmodels.OpsFloor, len(list))
	for _, f := range list {
		byName[f.Name] = f
	}

	f1Got := byName["F-WS-1"]
	require.NotNil(t, f1Got.WorkstationStats, "有工位楼层应注入统计")
	assert.Equal(t, int64(3), f1Got.WorkstationStats.Total)
	assert.Equal(t, int64(2), f1Got.WorkstationStats.Used, "status=1 占用 → 已使用")
	assert.Equal(t, int64(1), f1Got.WorkstationStats.Unused, "status=0 空闲 → 未使用")
	assert.Equal(t, int64(0), f1Got.WorkstationStats.Maintenance)

	f2Got := byName["F-WS-2"]
	require.NotNil(t, f2Got.WorkstationStats)
	assert.Equal(t, int64(2), f2Got.WorkstationStats.Total)
	assert.Equal(t, int64(1), f2Got.WorkstationStats.Used)
	assert.Equal(t, int64(0), f2Got.WorkstationStats.Unused)
	assert.Equal(t, int64(1), f2Got.WorkstationStats.Maintenance, "status=2 → 维护")
}

// TestFloorWorkstationStats_SoftDeleteExcluded 软删工位不计入统计
//（Model(&models.Workstation{}) 自动附加 deleted_at IS NULL）。
func TestFloorWorkstationStats_SoftDeleteExcluded(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "BWS2", "org-ws")
	f1 := &operationsmodels.OpsFloor{Name: "F-SD", FloorNo: "1", BuildingID: b.ID}
	require.NoError(t, svc.Create(ctx, f1))
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&sysmodels.Workstation{
			WorkstationName: "ws-sd", FloorID: &f1.ID, Status: sysmodels.WorkstationStatusAvailable,
		}).Error)
	}
	// 再建 1 个占用工位后软删 → 不计入
	deleted := &sysmodels.Workstation{WorkstationName: "ws-del", FloorID: &f1.ID, Status: sysmodels.WorkstationStatusOccupied}
	require.NoError(t, db.Create(deleted).Error)
	require.NoError(t, db.Delete(deleted).Error)

	page, err := svc.List(ctx, map[string]interface{}{})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsFloor)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].WorkstationStats)
	assert.Equal(t, int64(3), list[0].WorkstationStats.Total, "软删工位不应计入 total")
	assert.Equal(t, int64(0), list[0].WorkstationStats.Used, "软删的占用工位不应计入 used")
	assert.Equal(t, int64(3), list[0].WorkstationStats.Unused)
	assert.Equal(t, int64(0), list[0].WorkstationStats.Maintenance)
}

// TestFloorWorkstationStats_EmptyFloorNilAndOmitted 无工位楼层 WorkstationStats
// 保持 nil，且序列化 JSON 无 workstationStats 键（omitempty）。
func TestFloorWorkstationStats_EmptyFloorNilAndOmitted(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "BWS3", "org-ws")
	fEmpty := &operationsmodels.OpsFloor{Name: "F-EMPTY", FloorNo: "1", BuildingID: b.ID, OrderNum: 1}
	fFull := &operationsmodels.OpsFloor{Name: "F-FULL", FloorNo: "2", BuildingID: b.ID, OrderNum: 2}
	require.NoError(t, svc.Create(ctx, fEmpty))
	require.NoError(t, svc.Create(ctx, fFull))
	require.NoError(t, db.Create(&sysmodels.Workstation{
		WorkstationName: "ws-full", FloorID: &fFull.ID, Status: sysmodels.WorkstationStatusOccupied,
	}).Error)

	page, err := svc.List(ctx, map[string]interface{}{})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsFloor)
	require.Len(t, list, 2)

	byName := make(map[string]operationsmodels.OpsFloor, len(list))
	for _, f := range list {
		byName[f.Name] = f
	}

	empty := byName["F-EMPTY"]
	assert.Nil(t, empty.WorkstationStats, "无工位楼层 WorkstationStats 应保持 nil")
	data, err := json.Marshal(&empty)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "workstationStats",
		"omitempty 下无统计楼层 JSON 不应含 workstationStats 键")

	full := byName["F-FULL"]
	require.NotNil(t, full.WorkstationStats)
	data, err = json.Marshal(&full)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"workstationStats":{"total":1,"used":1,"unused":0,"maintenance":0}`,
		"嵌入 *Alias 的 MarshalJSON 应自动带出 workstationStats")
}

// TestFloorWorkstationStats_PaginationIsolation pageSize=1 只取第 1 页时，
// 聚合只覆盖页内 floor_id，其他楼层工位不混入。
func TestFloorWorkstationStats_PaginationIsolation(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "BWS4", "org-ws")
	fA := &operationsmodels.OpsFloor{Name: "F-PG-A", FloorNo: "1", BuildingID: b.ID, OrderNum: 1}
	fB := &operationsmodels.OpsFloor{Name: "F-PG-B", FloorNo: "2", BuildingID: b.ID, OrderNum: 2}
	require.NoError(t, svc.Create(ctx, fA))
	require.NoError(t, svc.Create(ctx, fB))
	require.NoError(t, db.Create(&sysmodels.Workstation{
		WorkstationName: "ws-a", FloorID: &fA.ID, Status: sysmodels.WorkstationStatusOccupied,
	}).Error)
	require.NoError(t, db.Create(&sysmodels.Workstation{
		WorkstationName: "ws-b", FloorID: &fB.ID, Status: sysmodels.WorkstationStatusMaintain,
	}).Error)

	page, err := svc.List(ctx, map[string]interface{}{"current": 1, "pageSize": 1})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsFloor)
	require.Len(t, list, 1)
	assert.Equal(t, "F-PG-A", list[0].Name, "默认排序 order_num ASC 第 1 页应为 F-PG-A")
	require.NotNil(t, list[0].WorkstationStats)
	assert.Equal(t, int64(1), list[0].WorkstationStats.Total)
	assert.Equal(t, int64(1), list[0].WorkstationStats.Used)
	assert.Equal(t, int64(0), list[0].WorkstationStats.Maintenance,
		"聚合只覆盖页内 floor_id：F-PG-B 的维护工位不应混入")
}
