package operations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// TestScope149_Door_ServiceWiring 就近验证 door_service.go 的 149 接线。
// 反向变异：去掉 Create 提交侧校验 / List 可见集 scope / BatchDelete 裁剪任一处必红。
func TestScope149_Door_ServiceWiring(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))

	// Create 提交侧越界 floor_id → 显式 403。
	err := f.DoorSvc.Create(ctxA, &operationsmodels.Door{FloorID: f.FloorB, Type: operationsmodels.DoorTypeSingle})
	assert.True(t, IsScopeDenied(err), "door Create 越界 floor_id → ErrScopeDenied(403)")

	// List 按可见集裁剪。
	list, err := f.DoorSvc.List(ctxA, requests.DoorListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total, "door List 只返回可见集内 1 行")

	// BatchDelete 后端权威裁剪。
	require.NoError(t, f.DoorSvc.BatchDelete(ctxA, []string{f.DoorA, f.DoorB, f.DoorBroken}))
	assert.Equal(t, int64(2), scope149CountVisible(t, f.DB, "ops_doors"),
		"只删可见 1 行；越界 + 断链 2 行不被误删")
}
