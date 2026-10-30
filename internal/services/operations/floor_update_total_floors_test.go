package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// =====================================================================
// 261013-lqb Task 1: floor_service.Update 触发楼栋 total_floors 重算。
// 修复漂移路径 A：Update 换楼此前全程零重算（91-03 只修了换楼工位同步，漏了
// 计数）；普通编辑重算当前楼栋 = 自愈设计（幂等实时 COUNT，部分失败后重试
// 走 Save 分支仍触发重算，历史漂移楼栋在任意楼层编辑后归正）。
//
// 楼层用 db.Create 直建（绕过 svc.Create 的重算）以获得可控的初始漂移态；
// Update 入参须带全字段——换楼分支按 map 写 name/floor_no/area/order_num/
// status/remark，缺字段会把 DB 中已有值覆盖为零值。
// =====================================================================

// seedLQBFloor 直建楼层行（不走 service Create，total_floors 保持不动）。
func seedLQBFloor(t *testing.T, db *gorm.DB, name, floorNo, buildingID string) *operationsmodels.OpsFloor {
	t.Helper()
	f := &operationsmodels.OpsFloor{Name: name, FloorNo: floorNo, BuildingID: buildingID}
	require.NoError(t, db.Create(f).Error)
	return f
}

// buildLQBFloorMoveInput 构造换楼 Update 入参（带全字段，见文件头注释）。
func buildLQBFloorMoveInput(id, newBuildingID, name, floorNo string) *operationsmodels.OpsFloor {
	return &operationsmodels.OpsFloor{
		BaseModel:  sysmodels.BaseModel{ID: id},
		BuildingID: newBuildingID,
		Name:       name,
		FloorNo:    floorNo,
		Status:     operationsmodels.FloorStatusNormal,
	}
}

// TestFloorUpdateTotalFloors_BuildingChangeRecountsBothSides 换楼后新旧楼栋
// total_floors 都要重算（修复路径 A 主断言）。
func TestFloorUpdateTotalFloors_BuildingChangeRecountsBothSides(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b1 := seedFloor91Building(t, db, "LQB-B1", "org-lqb")
	b2 := seedFloor91Building(t, db, "LQB-B2", "org-lqb")
	seedLQBFloor(t, db, "F-LQB-1", "1", b1.ID)
	fMoved := seedLQBFloor(t, db, "F-LQB-2", "2", b1.ID)

	// fMoved 从 B1 换到 B2
	require.NoError(t, svc.Update(ctx, buildLQBFloorMoveInput(fMoved.ID, b2.ID, fMoved.Name, fMoved.FloorNo)))

	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b1.ID), "旧楼失一层应重算为 1")
	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b2.ID), "新楼得一层应重算为 1")
}

// TestFloorUpdateTotalFloors_SameBuildingEditSelfHeals 同楼普通编辑（Save 分支）
// 触发所在楼栋重算——人工制造的漂移值被编辑动作自愈。
func TestFloorUpdateTotalFloors_SameBuildingEditSelfHeals(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b1 := seedFloor91Building(t, db, "LQB-BH", "org-lqb")
	f1 := seedLQBFloor(t, db, "F-H1", "1", b1.ID)
	seedLQBFloor(t, db, "F-H2", "2", b1.ID)

	// 人工制造漂移（模拟 Excel 导入直写/历史换楼遗留脏值）
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).
		Where("id = ?", b1.ID).Update("total_floors", 99).Error)
	require.Equal(t, 99, floor91BuildingTotalFloors(t, db, b1.ID))

	// 同楼改名（BuildingID 不变 → Save 分支）→ 自愈重算
	var floor operationsmodels.OpsFloor
	require.NoError(t, db.Where("id = ?", f1.ID).First(&floor).Error)
	floor.Name = "F-H1改"
	require.NoError(t, svc.Update(ctx, &floor))

	assert.Equal(t, 2, floor91BuildingTotalFloors(t, db, b1.ID), "同楼编辑应重算，漂移 99 被自愈为实际 2 层")
}

// TestFloorUpdateTotalFloors_MoveLastFloorOldBuildingZero 旧楼最后一层被移走的
// 清零边界：旧楼重算为 0（非保留旧值），新楼为 1。
func TestFloorUpdateTotalFloors_MoveLastFloorOldBuildingZero(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b1 := seedFloor91Building(t, db, "LQB-BZ1", "org-lqb")
	b2 := seedFloor91Building(t, db, "LQB-BZ2", "org-lqb")
	fLast := seedLQBFloor(t, db, "F-Z1", "1", b1.ID)

	require.NoError(t, svc.Update(ctx, buildLQBFloorMoveInput(fLast.ID, b2.ID, fLast.Name, fLast.FloorNo)))

	assert.Equal(t, 0, floor91BuildingTotalFloors(t, db, b1.ID), "旧楼最后一层移走应重算清零")
	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b2.ID))
}
