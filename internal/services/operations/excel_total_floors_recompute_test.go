package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// =====================================================================
// 261013-lqb Task 2: recomputeAllBuildingTotalFloors helper 直测。
// 锁定三组语义：软删楼层不计入（实时 COUNT 口径）/ 未删楼栋全部归正 /
// 软删楼栋不在重算范围（保持脏值）+ 二次执行幂等。
// =====================================================================

// TestRecomputeAllBuildingTotalFloors 全量重算：软删楼层排除、软删楼栋跳过、幂等。
func TestRecomputeAllBuildingTotalFloors(t *testing.T) {
	db := newFloor91TestDB(t)
	ctx := context.Background()

	// B1 挂 2 层（软删其 1）→ 实时活行数 1
	b1 := seedFloor91Building(t, db, "LQBR-B1", "org-lqb")
	f1a := seedLQBFloor(t, db, "F-R1a", "1", b1.ID)
	f1b := seedLQBFloor(t, db, "F-R1b", "2", b1.ID)
	require.NoError(t, db.Delete(f1b).Error)

	// B2 挂 1 层 → 活行数 1
	b2 := seedFloor91Building(t, db, "LQBR-B2", "org-lqb")
	seedLQBFloor(t, db, "F-R2", "1", b2.ID)
	_ = f1a

	// B3 楼栋本身软删（挂 1 层）→ 不在重算范围。
	// 注意顺序：先置脏再软删——GORM Update 自动附 deleted_at IS NULL scope，
	// 软删后置脏是静默 0 行（777 写不进去）。
	b3 := seedFloor91Building(t, db, "LQBR-B3", "org-lqb")
	seedLQBFloor(t, db, "F-R3", "1", b3.ID)
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).Where("id = ?", b3.ID).Update("total_floors", 777).Error)
	require.NoError(t, db.Delete(b3).Error)

	// 人工置脏：B1=99 / B2=0
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).Where("id = ?", b1.ID).Update("total_floors", 99).Error)
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).Where("id = ?", b2.ID).Update("total_floors", 0).Error)

	// 第一次重算
	require.NoError(t, recomputeAllBuildingTotalFloors(ctx, db))
	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b1.ID), "B1 软删 1 层后应重算为活行数 1")
	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b2.ID), "B2 应重算为活行数 1")
	assert.Equal(t, 777, lqbSoftDeletedBuildingTotalFloors(t, db, b3.ID), "软删楼栋不在重算范围，应保持置脏值 777")

	// 第二次重算（幂等：结果不变）
	require.NoError(t, recomputeAllBuildingTotalFloors(ctx, db))
	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b1.ID))
	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b2.ID))
	assert.Equal(t, 777, lqbSoftDeletedBuildingTotalFloors(t, db, b3.ID), "二次重算幂等，B3 仍保持 777")
}

// lqbSoftDeletedBuildingTotalFloors Unscoped 读取软删楼栋的 total_floors
// （floor91BuildingTotalFloors 的 First 带软删过滤，读不到软删行）。
func lqbSoftDeletedBuildingTotalFloors(t *testing.T, db *gorm.DB, buildingID string) int {
	t.Helper()
	var b operationsmodels.OpsBuilding
	require.NoError(t, db.Unscoped().Where("id = ?", buildingID).First(&b).Error)
	return b.TotalFloors
}
