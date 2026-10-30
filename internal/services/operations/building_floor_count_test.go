package operations

// =====================================================================
// Quick 261013-jf1: buildingService.List 每行楼层数注入（单条 GROUP BY，零 N+1）。
//
// 口径锁定（orchestrator）：楼层数 = ops_floors 中该楼栋非软删行数（实时 COUNT，
// 与 total_floors 反范式持久化字段并存且互不影响）。
//
// 覆盖：基本聚合 / 软删楼层排除 / 无楼层 nil+JSON omitempty / 分页隔离。
// 复用 floor_service_crud_test.go 的 newFloor91TestDB（已 AutoMigrate
// OpsBuilding + OpsFloor）与 seedFloor91Building，零新增基建。
// =====================================================================

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// TestBuildingFloorCount_BasicAggregation B1 挂 2 层 → 2；B2 挂 1 层 → 1；
// B3 挂 0 层 → nil 且 JSON 序列化不含 floorCount 键（omitempty）。
func TestBuildingFloorCount_BasicAggregation(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	b1 := seedFloor91Building(t, db, "BFC1", "org-fc")
	b2 := seedFloor91Building(t, db, "BFC2", "org-fc")
	seedFloor91Building(t, db, "BFC3", "org-fc")

	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC1-1", FloorNo: "1", BuildingID: b1.ID}).Error)
	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC1-2", FloorNo: "2", BuildingID: b1.ID}).Error)
	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC2-1", FloorNo: "1", BuildingID: b2.ID}).Error)

	page, err := svc.List(ctx, map[string]interface{}{})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsBuilding)
	require.Len(t, list, 3)

	byName := make(map[string]operationsmodels.OpsBuilding, len(list))
	for _, b := range list {
		byName[b.Name] = b
	}

	b1Got := byName["BFC1"]
	require.NotNil(t, b1Got.FloorCount, "有楼层楼栋应注入 FloorCount")
	assert.Equal(t, int64(2), *b1Got.FloorCount)

	b2Got := byName["BFC2"]
	require.NotNil(t, b2Got.FloorCount)
	assert.Equal(t, int64(1), *b2Got.FloorCount)

	b3Got := byName["BFC3"]
	assert.Nil(t, b3Got.FloorCount, "无楼层楼栋 FloorCount 应保持 nil")
	data, err := json.Marshal(&b3Got)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "floorCount",
		"omitempty 下无楼层楼栋 JSON 不应含 floorCount 键（前端渲染 \"-\" 依赖键缺省）")

	data, err = json.Marshal(&b1Got)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"floorCount":2`,
		"有楼层楼栋 JSON 应带出 floorCount")
}

// TestBuildingFloorCount_SoftDeleteExcluded 软删楼层不计入统计
//（Model(&operations.OpsFloor{}) 自动附加 deleted_at IS NULL）。
func TestBuildingFloorCount_SoftDeleteExcluded(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "BFC-SD", "org-fc")
	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC-SD-1", FloorNo: "1", BuildingID: b.ID}).Error)
	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC-SD-2", FloorNo: "2", BuildingID: b.ID}).Error)

	// 软删其一 → 只剩 1 层
	var victim operationsmodels.OpsFloor
	require.NoError(t, db.Where("name = ?", "FC-SD-2").First(&victim).Error)
	require.NoError(t, db.Delete(&victim).Error)

	page, err := svc.List(ctx, map[string]interface{}{})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsBuilding)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].FloorCount)
	assert.Equal(t, int64(1), *list[0].FloorCount, "软删楼层不应计入楼层数")
}

// TestBuildingFloorCount_PaginationIsolation pageSize=1 只取第 1 页时，
// 聚合只覆盖页内 building_id，其他楼栋楼层不混入。
func TestBuildingFloorCount_PaginationIsolation(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	bA := seedFloor91Building(t, db, "BFC-PG-A", "org-fc")
	bB := seedFloor91Building(t, db, "BFC-PG-B", "org-fc")
	// 默认序 order_num ASC：显式设置排序号保证页序可预期
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).Where("id = ?", bA.ID).Update("order_num", 1).Error)
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).Where("id = ?", bB.ID).Update("order_num", 2).Error)

	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC-A-1", FloorNo: "1", BuildingID: bA.ID}).Error)
	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC-A-2", FloorNo: "2", BuildingID: bA.ID}).Error)
	require.NoError(t, db.Create(&operationsmodels.OpsFloor{Name: "FC-B-1", FloorNo: "1", BuildingID: bB.ID}).Error)

	page, err := svc.List(ctx, map[string]interface{}{"current": 1, "pageSize": 1})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsBuilding)
	require.Len(t, list, 1)
	assert.Equal(t, "BFC-PG-A", list[0].Name, "默认排序 order_num ASC 第 1 页应为 BFC-PG-A")
	require.NotNil(t, list[0].FloorCount)
	assert.Equal(t, int64(2), *list[0].FloorCount,
		"聚合只覆盖页内 building_id：BFC-PG-B 的楼层不应混入")
}
