package operations

// =====================================================================
// 261014-ctp: buildingService Create/Update 末尾实时覆盖 total_floors。
// 关闭 261013-lqb SUMMARY 记录的最后残余向量：楼栋表单手工 Create+Update
// 仍写用户声明值到派生列 total_floors。
//
// 覆盖语义锁定（total_floors 是派生列，不是用户输入位）：
//   ① Create 带声明 totalFloors=999 → 落库后=0（新楼无楼层，实时 COUNT 覆盖）；
//   ② Update 带声明 totalFloors=999 → 落库后=当前真实楼层数；
//   ③ 软删楼层排除（实时 COUNT 口径，与 updateBuildingFloorCount 一致）。
//
// 复用 newFloor91TestDB / seedFloor91Building / seedLQBFloor /
// floor91BuildingTotalFloors 既有基建，零新增。OrgID 留空——validateOrg
// 对空 orgID 短路返回 nil（测试库无 sys_dept 表）。
// =====================================================================

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// TestBuildingTotalFloorsOverride_CreateOverridesDeclaredValue
// Create 带声明 totalFloors=999 → 落库后被实时 COUNT 覆盖为 0（新楼无楼层）。
func TestBuildingTotalFloorsOverride_CreateOverridesDeclaredValue(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	b := &operationsmodels.OpsBuilding{Name: "CTP-C1", TotalFloors: 999}
	require.NoError(t, svc.Create(ctx, b))
	require.NotEmpty(t, b.ID, "create 后 BaseModel UUID 应回填")

	assert.Equal(t, 0, floor91BuildingTotalFloors(t, db, b.ID),
		"声明的 999 应被实时 COUNT 覆盖为 0（新建楼栋无楼层）")
}

// TestBuildingTotalFloorsOverride_UpdateOverridesWithRealCount
// Update 带声明 totalFloors=999 → 落库后=当前真实楼层数（3）。
func TestBuildingTotalFloorsOverride_UpdateOverridesWithRealCount(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	b := &operationsmodels.OpsBuilding{Name: "CTP-U1"}
	require.NoError(t, svc.Create(ctx, b))
	seedLQBFloor(t, db, "F-CTP-1", "1", b.ID)
	seedLQBFloor(t, db, "F-CTP-2", "2", b.ID)
	seedLQBFloor(t, db, "F-CTP-3", "3", b.ID)

	// 模拟用户表单声明 totalFloors=999（Update 入参携带声明值）
	var got operationsmodels.OpsBuilding
	require.NoError(t, db.Where("id = ?", b.ID).First(&got).Error)
	got.TotalFloors = 999
	require.NoError(t, svc.Update(ctx, &got))

	assert.Equal(t, 3, floor91BuildingTotalFloors(t, db, b.ID),
		"声明的 999 应被实时 COUNT 覆盖为真实楼层数 3")
}

// TestBuildingTotalFloorsOverride_UpdateExcludesSoftDeletedFloors
// 楼栋挂 2 层（其中 1 层软删）→ Update 后=1（软删排除的实时 COUNT 口径）。
func TestBuildingTotalFloorsOverride_UpdateExcludesSoftDeletedFloors(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	b := &operationsmodels.OpsBuilding{Name: "CTP-SD1"}
	require.NoError(t, svc.Create(ctx, b))
	seedLQBFloor(t, db, "F-CTP-SD-1", "1", b.ID)
	fVictim := seedLQBFloor(t, db, "F-CTP-SD-2", "2", b.ID)

	// 软删其一 → 实时活行数 1
	require.NoError(t, db.Delete(fVictim).Error)

	// 声明 999 + Update → 覆盖为软删排除后的 1
	var got operationsmodels.OpsBuilding
	require.NoError(t, db.Where("id = ?", b.ID).First(&got).Error)
	got.TotalFloors = 999
	require.NoError(t, svc.Update(ctx, &got))

	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b.ID),
		"软删楼层不应计入，Update 后应为活行数 1")
}
