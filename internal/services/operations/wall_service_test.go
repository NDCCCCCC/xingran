package operations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// TestScope149_Wall_ServiceWiring 就近验证 wall_service.go 的 149 接线：
// Create 提交侧越界 → 显式 403；List 按可见集裁剪；BatchDelete 后端权威裁剪。
// 反向变异：去掉 Create 的 checkEntitySubmittedScope / List 的 entityReadScope /
// BatchDelete 的 filterEntityVisibleIDs 任一处，本用例必红。
func TestScope149_Wall_ServiceWiring(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))

	// Create 提交侧越界 floor_id → 显式 403（service 层接线生效）。
	err := f.WallSvc.Create(ctxA, &operationsmodels.Wall{FloorID: f.FloorB, Type: operationsmodels.WallTypeStraight})
	assert.True(t, IsScopeDenied(err), "wall Create 越界 floor_id → ErrScopeDenied(403)")

	// List 按可见集裁剪：DeptA 只见 1 行（B 档 + 断链剔除）。
	list, err := f.WallSvc.List(ctxA, requests.WallListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total, "wall List 只返回可见集内 1 行")

	// BatchDelete 后端权威裁剪：提交全量 ID，只有可见行被删。
	require.NoError(t, f.WallSvc.BatchDelete(ctxA, []string{f.WallA, f.WallB, f.WallBroken}))
	assert.Equal(t, int64(2), scope149CountVisible(t, f.DB, "ops_walls"),
		"只删可见 1 行；越界 + 断链 2 行不被误删")
}
