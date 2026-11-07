package middleware

// permission_datascope_ctx_test.go —— DS-02 request context 注入契约回归
// （v1.52 Phase 130，Lead 2026-11-07 补充约定）。
//
// DataScopePermission 除保持 c.Set("data_scope"/"user_id") 原行为外，还必须把
// ScopeResult 注入 c.Request 的 context（datascope.WithContext），供 131/132 的
// service 通过 datascope.FromContext(ctx) 取可见集，无需改 handler 签名。
//
// 硬约束：
//   - 超管早退分支**同样**注入 All 的 ScopeResult（All 档缓存 key 也要含 scope 维度），
//     但绝不补写 c.Set（system/users 与 vdi 零回归）。
//   - 非超管 All(1) 的 ScopeResult 必须带 UserID。

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

// TestDataScopePermission_RequestContext_Admin 超管早退：注入 All 但不补 c.Set。
func TestDataScopePermission_RequestContext_Admin(t *testing.T) {
	adminID := uuid.NewString()
	coreAdmin := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		roleID := uuid.NewString()
		require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, 'admin', 'admin', 0, 0)`, roleID).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, adminID, roleID).Error)
	})
	c, _ := newMWContext("GET", "/ops/building/list")
	c.Set("user_id", adminID)

	DataScopePermission(coreAdmin)(c)

	assert.False(t, c.IsAborted(), "超管直接放行")
	// 零回归：超管分支不得写 c.Set("data_scope")。
	_, exists := c.Get("data_scope")
	assert.False(t, exists, "超管早退不写 data_scope（既有行为零变化）")

	// 但 request context 必须注入 All 的 ScopeResult（不能因短路跳过注入）。
	got, ok := datascope.FromContext(c.Request.Context())
	require.True(t, ok, "超管必须注入 request context 的 ScopeResult")
	assert.True(t, got.All, "超管 ScopeResult 必须是 All")
	assert.False(t, got.FailClosed)
	assert.Equal(t, adminID, got.UserID, "UserID 原样透传")
}

// TestDataScopePermission_RequestContext_Normal 普通用户：c.Set 原行为 + ctx 注入。
func TestDataScopePermission_RequestContext_Normal(t *testing.T) {
	userID := uuid.NewString()
	core, _ := seedPermData(t, userID) // 角色 data_scope 默认 1（All）
	c, _ := newMWContext("GET", "/ops/floor/list")
	c.Set("user_id", userID)

	DataScopePermission(core)(c)

	assert.False(t, c.IsAborted())
	// c.Set 原行为一字不改。
	ds, ok := c.Get("data_scope")
	require.True(t, ok)
	assert.Equal(t, models.DataScope(1), ds)
	uid, ok := c.Get("user_id")
	require.True(t, ok)
	assert.Equal(t, userID, uid)

	// request context 注入完整 Resolve 结果；All(1) 短路零 DB 查询但必须带 UserID。
	got, ok := datascope.FromContext(c.Request.Context())
	require.True(t, ok, "普通用户必须注入 request context 的 ScopeResult")
	assert.True(t, got.All, "data_scope=1 → All")
	assert.Equal(t, userID, got.UserID, "非超管 All(1) 必须带 UserID")
	assert.False(t, got.FailClosed)
}

// TestDataScopePermission_RequestContext_Unauthorized 未认证：不注入 ctx、直接 401。
func TestDataScopePermission_RequestContext_Unauthorized(t *testing.T) {
	c, _ := newMWContext("GET", "/ops/asset/list")
	DataScopePermission(nil)(c)
	assert.True(t, c.IsAborted())
	_, ok := datascope.FromContext(c.Request.Context())
	assert.False(t, ok, "未认证请求不得注入 ScopeResult")
}

// TestDataScopeContext_HelpersRoundTrip WithContext / FromContext 往返与缺省语义
// （私有 typed key 不与其他 context 值冲突）。
func TestDataScopeContext_HelpersRoundTrip(t *testing.T) {
	want := datascope.ScopeResult{
		All:          false,
		DeptIDs:      []string{"d1", "d2"},
		ResourceHook: datascope.HookSelfUser,
		FailClosed:   false,
		UserID:       "u-42",
	}
	ctx := datascope.WithContext(context.Background(), want)
	got, ok := datascope.FromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, want, got, "往返（含 UserID）必须逐字段一致")

	// 缺省：未注入 → (零值, false)，调用方按「未挂注入中间件」兜底。
	got2, ok2 := datascope.FromContext(context.Background())
	assert.False(t, ok2)
	assert.Equal(t, datascope.ScopeResult{}, got2)

	// 其它 context 值不污染取值。
	ctx3 := context.WithValue(context.Background(), "unrelated", want)
	_, ok3 := datascope.FromContext(ctx3)
	assert.False(t, ok3, "私有 typed key 不得被普通 key 命中")
}
