package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// =====================================================================
// Phase 91-04: floor_plan_text 六方法 CRUD 冒烟（Wave 0 缺口补齐）。
// 迁移前对旧实现全绿，迁移后逐字节同结果——repo 化的行为采样器。
// 脚手架照 crud_services_test.go 模式（sqlite + AutoMigrate FloorPlanText
// + 关联 Floor 模型）。
// =====================================================================

func newFloorPlanTextTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newCRUDTestDB(t)
	require.NoError(t, db.AutoMigrate(&operationsmodels.FloorPlanText{}))
	return db
}

func TestFloorPlanText91_CRUDSmoke(t *testing.T) {
	db := newFloorPlanTextTestDB(t)
	_, floorID := seedBuildingFloor(t, db, "fpt91")
	svc := NewFloorPlanTextService(db)
	ctx := context.Background()

	// Create → GetByID 含字段
	txt := &operationsmodels.FloorPlanText{FloorID: floorID, Content: "91冒烟A", Position: `{"x":1}`, FontSize: 16}
	require.NoError(t, svc.Create(ctx, txt))
	got, err := svc.GetByID(ctx, txt.ID)
	require.NoError(t, err)
	assert.Equal(t, "91冒烟A", got.Content)
	assert.Equal(t, 16, got.FontSize)
	assert.Equal(t, floorID, got.FloorID)

	// Update
	got.Content = "91冒烟A改"
	got.FontSize = 20
	require.NoError(t, svc.Update(ctx, got))
	got, err = svc.GetByID(ctx, txt.ID)
	require.NoError(t, err)
	assert.Equal(t, "91冒烟A改", got.Content)
	assert.Equal(t, 20, got.FontSize)

	// Create 第二条（不同楼层）供 List 过滤用例区分
	_, floorID2 := seedBuildingFloor(t, db, "fpt91-b")
	require.NoError(t, svc.Create(ctx, &operationsmodels.FloorPlanText{FloorID: floorID2, Content: "91冒烟B", Position: "{}"}))

	// List filter 命中（floorId + content 模糊）
	page, err := svc.List(ctx, requests.FloorPlanTextListRequest{FloorID: floorID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	list, ok := page.List.([]operationsmodels.FloorPlanText)
	require.True(t, ok, "repo.List 装值切片 []T")
	assert.Equal(t, "91冒烟A改", list[0].Content)

	page, err = svc.List(ctx, requests.FloorPlanTextListRequest{Content: "91冒烟B"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// 默认序（B 型排他默认：无 OrderByColumn → created_at DESC，后创建的在前）
	page, err = svc.List(ctx, requests.FloorPlanTextListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	list, _ = page.List.([]operationsmodels.FloorPlanText)
	require.Len(t, list, 2)
	assert.Equal(t, "91冒烟B", list[0].Content, "默认 created_at DESC：后创建的在前")

	// 用户排序（白名单命中 createdAt ASC）→ 排他，无尾随序
	asc := true
	page, err = svc.List(ctx, requests.FloorPlanTextListRequest{
		PaginationParams: requests.PaginationParams{BaseListRequest: base.BaseListRequest{
			OrderByColumn: "createdAt", IsAsc: &asc,
		}},
	})
	require.NoError(t, err)
	list, _ = page.List.([]operationsmodels.FloorPlanText)
	require.Len(t, list, 2)
	assert.Equal(t, "91冒烟A改", list[0].Content, "createdAt ASC：先创建的在前")

	// Delete 软删后 GetByID NotFound
	require.NoError(t, svc.Delete(ctx, txt.ID))
	_, err = svc.GetByID(ctx, txt.ID)
	require.Error(t, err)

	// BatchDelete(nil) NoError（91-01 P1 语义反转锁定）
	require.NoError(t, svc.BatchDelete(ctx, nil))
	// BatchDelete 命中剩余一条
	page, err = svc.List(ctx, requests.FloorPlanTextListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	remaining, _ := page.List.([]operationsmodels.FloorPlanText)
	require.Len(t, remaining, 1)
	require.NoError(t, svc.BatchDelete(ctx, []string{remaining[0].ID}))
	page, err = svc.List(ctx, requests.FloorPlanTextListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), page.Total)
}
