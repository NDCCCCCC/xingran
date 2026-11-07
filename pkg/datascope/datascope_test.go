package datascope

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newScopeTestDB 建最小夹具：sys_dept(id, ancestors, status) / sys_user(id, dept_id)
// / sys_role(id, data_scope, status) / sys_user_role(user_id, role_id) / sys_role_dept。
// 全部 id 用 TEXT 列，验证 CAST AS TEXT 双侧比较口径在 SQLite 下等价成立。
func newScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "datascope.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	stmts := []string{
		`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, ancestors TEXT, status INTEGER DEFAULT 0)`,
		`CREATE TABLE sys_user (id TEXT PRIMARY KEY, dept_id TEXT, deleted_at TEXT)`,
		`CREATE TABLE sys_role (id TEXT PRIMARY KEY, data_scope INTEGER, status INTEGER DEFAULT 0)`,
		`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`,
		`CREATE TABLE sys_role_dept (role_id TEXT, dept_id TEXT)`,
		// d1 → d2 → d3 三级子树；d9 为无关部门（越界对照）
		`INSERT INTO sys_dept (id, ancestors, status) VALUES
			('d1','0',0), ('d2','0,d1',0), ('d3','0,d1,d2',0), ('d9','0',0)`,
		`INSERT INTO sys_user (id, dept_id, deleted_at) VALUES ('u-d2','d2',NULL), ('u-none',NULL,NULL)`,
		`INSERT INTO sys_role (id, data_scope, status) VALUES
			('r-all',1,0), ('r-custom',2,0), ('r-dept',3,0), ('r-child',4,0), ('r-self',5,0), ('r-disabled',1,1)`,
		`INSERT INTO sys_user_role (user_id, role_id) VALUES ('u-d2','r-dept')`,
		`INSERT INTO sys_role_dept (role_id, dept_id) VALUES ('r-custom','d1'), ('r-custom','d9'), ('r-custom','d1')`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error, "夹具建表失败: %s", s)
	}
	return db
}

// TestScopeResolve_AllShortCircuit 覆盖裁定 §3 用例 1/2/3：
// 超管早退(0) / 未挂注入(0) / All(1) 都映射 All 且零 DB 查询。
func TestScopeResolve_AllShortCircuit(t *testing.T) {
	cases := []struct {
		name  string
		scope models.DataScope
	}{
		{"超管早退未注入=0", 0},
		{"未挂注入中间件=0", 0},
		{"显式All=1", models.DataScopeAll},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// db 传 nil：若实现误去查部门集，会直接返回 error / panic，从而暴露非短路。
			got, err := Resolve(context.Background(), nil, "u-d2", tc.scope)
			require.NoError(t, err)
			assert.True(t, got.All, "0/1 必须映射 All（超管零回归）")
			assert.False(t, got.FailClosed, "0/1 绝不能进 fail-closed 分支")
			assert.Empty(t, got.DeptIDs)
			assert.Equal(t, HookNone, got.ResourceHook)
			assert.Equal(t, "u-d2", got.UserID)
		})
	}
}

// TestScopeResolve_FailClosed 覆盖裁定 §3 用例 4：仅非 0 异常枚举 fail-closed。
func TestScopeResolve_FailClosed(t *testing.T) {
	db := newScopeTestDB(t)
	for _, bad := range []models.DataScope{6, -1, 99} {
		got, err := Resolve(context.Background(), db, "u-d2", bad)
		require.NoError(t, err)
		assert.True(t, got.FailClosed, "异常枚举 %d 必须 fail-closed", bad)
		assert.False(t, got.All)
		assert.Empty(t, got.DeptIDs)
		assert.Equal(t, HookNone, got.ResourceHook)
	}
	// 对照：0 不 fail-closed（防把超管锁死）
	got, err := Resolve(context.Background(), db, "u-d2", 0)
	require.NoError(t, err)
	assert.True(t, got.All)
	assert.False(t, got.FailClosed)
}

// TestScopeResolve_Self 覆盖裁定 §2 + §3 用例 5/6 的下游语义：
// Self 档返回空部门集 + HookSelfUser + UserID，由各域决定按用户列过滤。
func TestScopeResolve_Self(t *testing.T) {
	db := newScopeTestDB(t)
	got, err := Resolve(context.Background(), db, "u-d2", models.DataScopeSelf)
	require.NoError(t, err)
	assert.False(t, got.All)
	assert.False(t, got.FailClosed)
	assert.Empty(t, got.DeptIDs, "Self 不纳入部门集展开")
	assert.Equal(t, HookSelfUser, got.ResourceHook)
	assert.Equal(t, "u-d2", got.UserID, "Self 档必须带登录人 UserID 供资源列过滤")
}

// TestScopeResolve_MultiRoleMax 覆盖裁定 §3 用例 5/6/7：
// MAX = 数值最大 = 最严档（1=All…5=Self）；无角色 COALESCE→Self(5)；
// 停用角色不计入。本包不改 getUserMaxDataScope 语义，这里验证
// 「传入最严档 5」时 Resolve 的行为，并把 MAX 取值本身留给 middleware 既有测试守护。
func TestScopeResolve_MultiRoleMaxSemantics(t *testing.T) {
	db := newScopeTestDB(t)
	// 组合 {1 All, 3 Dept, 5 Self} 的 MAX 结果 = 5（最严）
	got, err := Resolve(context.Background(), db, "u-d2", models.DataScopeSelf)
	require.NoError(t, err)
	assert.Equal(t, HookSelfUser, got.ResourceHook, "MAX=5 → Self 档，不得放宽为 All/Dept")

	// 无角色 COALESCE 兜底 5 → Self（同样走 Self 分支）
	got2, err := Resolve(context.Background(), db, "u-none", models.DataScopeSelf)
	require.NoError(t, err)
	assert.Equal(t, HookSelfUser, got2.ResourceHook)
	assert.Equal(t, "u-none", got2.UserID)
}

// TestScopeResolve_Dept 本部门档：集合 = {用户 dept_id}。
func TestScopeResolve_Dept(t *testing.T) {
	db := newScopeTestDB(t)
	got, err := Resolve(context.Background(), db, "u-d2", models.DataScopeDept)
	require.NoError(t, err)
	assert.False(t, got.All)
	assert.False(t, got.FailClosed)
	assert.Equal(t, []string{"d2"}, got.DeptIDs)
}

// TestScopeResolve_DeptChild 本部门及子部门档：ancestors 四条件展开。
// d2 的子树 = {d2, d3}；d1（父）与 d9（无关）都不应进入集合。
func TestScopeResolve_DeptChild(t *testing.T) {
	db := newScopeTestDB(t)
	got, err := Resolve(context.Background(), db, "u-d2", models.DataScopeDeptChild)
	require.NoError(t, err)
	assert.False(t, got.All)
	assert.False(t, got.FailClosed)
	assert.Equal(t, []string{"d2", "d3"}, got.DeptIDs, "子树展开须含自身与后代，排除父与无关部门")
}

// TestScopeResolve_DeptChild_AncestorsMiddle 验证「deptId 位于 ancestors 路径中间」
// 的场景（d1 在 "0,d1,d2" 中间仍能命中），防三条件旧实现回归。
func TestScopeResolve_DeptChild_AncestorsMiddle(t *testing.T) {
	db := newScopeTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, dept_id, deleted_at) VALUES ('u-d1','d1',NULL)`).Error)
	got, err := Resolve(context.Background(), db, "u-d1", models.DataScopeDeptChild)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"d1", "d2", "d3"}, got.DeptIDs)
}

// TestScopeResolve_Custom 自定义档：sys_role_dept ⋈ sys_user_role 且只计启用角色，
// 结果去重排序（normalizeIDs），供缓存 key 稳定派生。
func TestScopeResolve_Custom(t *testing.T) {
	db := newScopeTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('u-custom','r-custom')`).Error)
	got, err := Resolve(context.Background(), db, "u-custom", models.DataScopeCustom)
	require.NoError(t, err)
	assert.False(t, got.All)
	assert.False(t, got.FailClosed)
	assert.Equal(t, []string{"d1", "d9"}, got.DeptIDs, "应去重且排序，不含重复 d1")
}

// TestScopeResolve_Custom_DisabledRoleExcluded 停用角色（status=1）不计入。
func TestScopeResolve_Custom_DisabledRoleExcluded(t *testing.T) {
	db := newScopeTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES ('r-disabled','d9')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('u-disabled','r-disabled')`).Error)
	got, err := Resolve(context.Background(), db, "u-disabled", models.DataScopeCustom)
	require.NoError(t, err)
	assert.True(t, got.FailClosed, "仅有停用角色 → 空集 → fail-closed，不放行全库")
	assert.Empty(t, got.DeptIDs)
}

// TestScopeResolve_NoDeptFailClosed 用户无部门时不得放行全库。
func TestScopeResolve_NoDeptFailClosed(t *testing.T) {
	db := newScopeTestDB(t)
	for _, scope := range []models.DataScope{models.DataScopeDept, models.DataScopeDeptChild} {
		got, err := Resolve(context.Background(), db, "u-none", scope)
		require.NoError(t, err)
		assert.True(t, got.FailClosed, "scope=%d 且用户无部门时必须 fail-closed", scope)
		assert.False(t, got.All)
	}
}

// TestScopeContext_RoundTrip 验证 WithContext / FromContext 往返与缺省行为。
func TestScopeContext_RoundTrip(t *testing.T) {
	want := ScopeResult{All: true, UserID: "u-1"}
	ctx := WithContext(context.Background(), want)
	got, ok := FromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, want, got)

	// 未注入 → (零值, false)，调用方须按「未挂注入」处理。
	got2, ok2 := FromContext(context.Background())
	assert.False(t, ok2)
	assert.Equal(t, ScopeResult{}, got2)
	assert.False(t, got2.All, "零值 ScopeResult 不得被误当成 All")

	// nil ctx 不 panic（FromContext 内部有 nil 防御）。
	// 用变量传递而非直接传 nil 字面量：staticcheck SA1012 禁止把 nil Context
	// 直接传给接收 context.Context 的函数，但这里测的正是该防御分支本身。
	var nilCtx context.Context
	got3, ok3 := FromContext(nilCtx)
	assert.False(t, ok3)
	assert.Equal(t, ScopeResult{}, got3)
}

// TestAllScopeAndFailClosedScope 构造器的不变量。
func TestAllScopeAndFailClosedScope(t *testing.T) {
	all := AllScope("u-1")
	assert.True(t, all.All)
	assert.False(t, all.FailClosed)
	assert.Equal(t, "u-1", all.UserID)

	fc := FailClosedScope("u-2")
	assert.False(t, fc.All)
	assert.True(t, fc.FailClosed)
	assert.Empty(t, fc.DeptIDs)
	assert.Equal(t, "u-2", fc.UserID)
}
