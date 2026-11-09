package operations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScope149_RoomPhoto_ServiceWiring 就近验证 room_photo_service.go 的
// 149 接线：写端点改前侧越界 → 403；读端点按两级链裁剪；count 同源；
// BatchDeletePhotos 后端权威裁剪（保存侧不误删越界照片）。
// 反向变异：去掉 photoExistingWriteGuard / ListByRoom 的 entityReadScope /
// BatchDeletePhotos 的 filterEntityVisibleIDs 任一处必红。
func TestScope149_RoomPhoto_ServiceWiring(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))

	// DeletePhoto 改前侧越界 photoID → 显式 403。
	err := f.PhotoSvc.DeletePhoto(ctxA, f.PhotoB)
	assert.True(t, IsScopeDenied(err), "photo DeletePhoto 越界 → ErrScopeDenied(403)")

	// ListByRoom 按 room_id → serverRoom → building 两级链裁剪（此处 RoomA 可见）。
	photos, err := f.PhotoSvc.ListByRoom(ctxA, f.RoomA)
	require.NoError(t, err)
	assert.Len(t, photos, 1, "photo ListByRoom DeptA 只返回可见 1 行")

	// CountPhotos 与 ListByRoom 同源（不出现 total 全库 + list 裁剪）。
	cnt, err := f.PhotoSvc.CountPhotos(ctxA, f.RoomA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cnt, "photo CountPhotos 与 List 同源")

	// BatchDeletePhotos 后端权威裁剪：提交全量 ID，只有可见行被删。
	require.NoError(t, f.PhotoSvc.BatchDeletePhotos(ctxA, []string{f.PhotoA, f.PhotoB, f.PhotoBroken}))
	assert.Equal(t, int64(2), scope149CountVisible(t, f.DB, "ops_room_photos"),
		"只删可见 1 行；越界 + 断链 2 行不被误删")
}
