package middleware

// anchor_mount_test.go —— 机构锚点维度在 DataScopePermission 上的接线断言（v1.54）。
//
// 断言重点（契约 C1/C2）：
//   - 未挂任何锚点来源时，注入的 ScopeResult 与 datascope.Resolve 原输出逐字段相等；
//   - 锚点来源生效时，被注入的 ScopeResult.DeptIDs 含锚点子树；
//   - All 用户叠加锚点后仍为 All（绝不收窄）。
//
// 说明：中间件的锚点解析依赖 sys_role/sys_post 的 anchor_mode 列。若夹具未建这些列，
// 查询会报错并走"按无锚点降级"路径 —— 本测试因此同时验证降级不破坏原语义。

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

// TestMergeIsAppliedByContract 断言合并规则的纯函数契约（中间件接线的前提）。
//
// 中间件里真正的接线是 `scopeResult = datascope.Merge(scopeResult, anchorIDs)`，
// 该行的正确性完全由 Merge 的纯函数语义决定，故在中间件包内锁死这条依赖。
func TestMergeIsAppliedByContract(t *testing.T) {
	// 无锚点：恒等
	resolved := datascope.ScopeResult{DeptIDs: []string{"d1"}, UserID: "u1"}
	assert.Equal(t, resolved, datascope.Merge(resolved, nil))

	// 有锚点：并集
	got := datascope.Merge(resolved, []string{"d2"})
	assert.Equal(t, []string{"d1", "d2"}, got.DeptIDs)

	// All：不收窄
	all := datascope.AllScope("u1")
	assert.True(t, datascope.Merge(all, []string{"d2"}).All)
}

// TestAnchorColumnsDoNotBreakOriginalScope 断言"夹具无锚点列"时降级正确：
// 锚点查询失败 → 中间件按无锚点处理 → 注入值必须与原 Resolve 输出一致。
//
// 这是新增维度最重要的安全属性：**任何锚点侧异常都不能影响既有角色维度**。
func TestAnchorColumnsDoNotBreakOriginalScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mw.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	// 只建最小表：故意**不建** anchor_mode/anchor_param 列，模拟老库/未迁移状态。
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, ancestors TEXT, status INTEGER DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user (id TEXT PRIMARY KEY, dept_id TEXT, deleted_at TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_role (id TEXT PRIMARY KEY, data_scope INTEGER, status INTEGER DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_post (user_id TEXT, post_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_post (id TEXT PRIMARY KEY, post_code TEXT, post_name TEXT, status INTEGER DEFAULT 0)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, ancestors, status) VALUES ('d1','0',0),('d2','0,d1',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, dept_id, deleted_at) VALUES ('u1','d2',NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, data_scope, status) VALUES ('r1',3,0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('u1','r1')`).Error)

	// 锚点侧查询必然失败（缺列）→ 必须返回空锚点，而不是返回错误把整个 scope 打掉。
	ids, anchorErr := datascope.ResolveAnchorGrants(context.Background(), db, "u1")
	assert.Error(t, anchorErr, "缺列时应报错，由中间件降级处理")
	assert.Empty(t, ids)

	// 降级后合并结果必须与原 Resolve 完全一致
	base := datascope.ScopeResult{DeptIDs: []string{"d2"}, UserID: "u1"}
	assert.Equal(t, base, datascope.Merge(base, ids), "锚点侧异常不得改变角色维度")
}

// ===========================================================================
// 中间件接线级断言（verifier 补充，v1.54 核验）
//
// 上面的 TestMergeIsAppliedByContract 只锁死**纯函数** Merge 的语义，
// TestAnchorColumnsDoNotBreakOriginalScope 只锁死**降级**路径 —— 两者都
// 没有证明「DataScopePermission 真的把锚点并进了注入的 ScopeResult」。
// 契约 C1/C2 的最终生效点是中间件那一行，故这里用真实 sqlite 夹具 + 真实
// 中间件跑端到端断言，逐条对应契约 §九 用例 A/C/D/E。
// ===========================================================================

// newAnchorWiringDB 建"机构树 + 用户 + 角色 + 岗位"的中间件级夹具。
//
// 树形与生产一致：湖北分公司(level=1) → 武汉中支(level=2) → 硚口支公司(level=3)，
// 刘洋在武汉中支下的科室「综合管理部」(level=0)，ancestors 含武汉中支。
func newAnchorWiringDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "anchor_wiring.db")
	gormDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := gormDB.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	for _, s := range []string{
		`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, parent_id TEXT,
			ancestors TEXT DEFAULT '', org_level INTEGER DEFAULT 0,
			is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_user (id TEXT PRIMARY KEY, username TEXT, nickname TEXT,
			dept_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_role (id TEXT PRIMARY KEY, role_name TEXT, role_key TEXT,
			data_scope INTEGER DEFAULT 1, anchor_mode INTEGER DEFAULT 0,
			anchor_param TEXT DEFAULT '', status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`,
		`CREATE TABLE sys_post (id TEXT PRIMARY KEY, post_code TEXT, post_name TEXT,
			anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '',
			status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_user_post (user_id TEXT, post_id TEXT)`,
	} {
		require.NoError(t, gormDB.Exec(s).Error)
	}

	require.NoError(t, gormDB.Exec(`INSERT INTO sys_dept
		(id, dept_name, parent_id, ancestors, org_level, is_external_org, status) VALUES
		('D-root','湖北分公司',NULL,   '',            1, 1, 0),
		('D-wh',  '武汉中支',  'D-root','D-root',      2, 1, 0),
		('D-qk',  '硚口支公司','D-wh',  'D-root,D-wh', 3, 1, 0),
		('D-zh',  '综合管理部','D-wh',  'D-root,D-wh', 0, 0, 0)`).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('U-liu','liuyang','刘洋','D-zh',0)`,
	).Error)

	return gormDB
}

// anchorWiringCore 用上面夹具包成 middleware 需要的 core.Core。
func anchorWiringCore(gormDB *gorm.DB) *core.Core {
	return &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
	}
}

// TestDataScopePermission_AnchorWiring_Level2IncludesBranch 用例 A（中间件级）。
//
// 刘洋（dept=综合管理部，ancestors 含武汉中支）持 anchor_mode=5/param="2" 的岗位，
// 角色 data_scope=3（本部门）。经 DataScopePermission 注入的可见集必须
// **包含 level=3 的硚口支公司** —— 这是"中支管得住支公司"的关键语义。
func TestDataScopePermission_AnchorWiring_Level2IncludesBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.NewString()
	roleID := uuid.NewString()
	postID := uuid.NewString()
	gormDB := newAnchorWiringDB(t)

	require.NoError(t, gormDB.Exec(`INSERT INTO sys_role
		(id, role_name, role_key, data_scope, anchor_mode, anchor_param, status)
		VALUES (?, '本部门角色', 'dept_role', 3, 0, '', 0)`, roleID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID,
	).Error)
	require.NoError(t, gormDB.Exec(`INSERT INTO sys_post
		(id, post_code, post_name, anchor_mode, anchor_param, status, deleted_at)
		VALUES (?, 'mid_admin', '中支管理员', 5, '2', 0, NULL)`, postID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_post (user_id, post_id) VALUES (?, ?)`, userID, postID,
	).Error)
	require.NoError(t, gormDB.Exec(
		`UPDATE sys_user SET id = ? WHERE id = 'U-liu'`, userID,
	).Error)

	mwCore := anchorWiringCore(gormDB)
	c, _ := newMWContext("GET", "/ops/building/list")
	c.Set("user_id", userID)

	DataScopePermission(mwCore)(c)

	assert.False(t, c.IsAborted(), "非超管不应被中间件 abort")

	got, ok := datascope.FromContext(c.Request.Context())
	require.True(t, ok, "必须注入 ScopeResult")
	assert.False(t, got.All, "data_scope=3 不是 All")
	assert.False(t, got.FailClosed)
	assert.Equal(t, datascope.HookNone, got.ResourceHook, "机构维度胜出后 Hook 必须是 None")

	// 角色维度给的是本部门 D-zh；锚点维度给的是武汉中支子树。
	assert.Contains(t, got.DeptIDs, "D-zh", "角色维度（本部门）必须保留")
	assert.Contains(t, got.DeptIDs, "D-wh", "锚点必须命中武汉中支")
	assert.Contains(t, got.DeptIDs, "D-qk", "中支子树必须包含 level=3 的硚口支公司（用例 A 核心）")
	assert.NotContains(t, got.DeptIDs, "D-root", "武汉中支的子树不得包含其父级分公司")
}

// TestDataScopePermission_AnchorWiring_AllSurvives 用例 C（中间件级，C1 回归）。
//
// All(1) 角色用户叠加有效锚点后，注入的 ScopeResult 必须仍是 All=true，
// 且 DeptIDs 保持为空（绝不被锚点维度替换成有限部门集）。
func TestDataScopePermission_AnchorWiring_AllSurvives(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.NewString()
	roleID := uuid.NewString()
	postID := uuid.NewString()
	gormDB := newAnchorWiringDB(t)

	require.NoError(t, gormDB.Exec(`INSERT INTO sys_role
		(id, role_name, role_key, data_scope, anchor_mode, anchor_param, status)
		VALUES (?, '全局角色', 'all_role', 1, 0, '', 0)`, roleID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID,
	).Error)
	require.NoError(t, gormDB.Exec(`INSERT INTO sys_post
		(id, post_code, post_name, anchor_mode, anchor_param, status, deleted_at)
		VALUES (?, 'mid_admin', '中支管理员', 5, '2', 0, NULL)`, postID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_post (user_id, post_id) VALUES (?, ?)`, userID, postID,
	).Error)
	require.NoError(t, gormDB.Exec(`UPDATE sys_user SET id = ? WHERE id = 'U-liu'`, userID).Error)

	mwCore := anchorWiringCore(gormDB)
	c, _ := newMWContext("GET", "/ops/building/list")
	c.Set("user_id", userID)

	DataScopePermission(mwCore)(c)

	got, ok := datascope.FromContext(c.Request.Context())
	require.True(t, ok)
	assert.True(t, got.All, "All(1) 用户叠加锚点后必须仍为 All（C1 静默降权回归）")
	assert.Empty(t, got.DeptIDs, "All 档不得被锚点维度填入有限部门集")
	assert.False(t, got.FailClosed)
	// 既有 c.Set 行为零变化
	ds, exists := c.Get("data_scope")
	require.True(t, exists)
	assert.Equal(t, models.DataScope(1), ds, "c.Set(\"data_scope\") 原值不得被锚点改动")
}

// TestDataScopePermission_AnchorWiring_SelfOverridden 用例 D（中间件级）。
//
// 仅本人（data_scope=5）用户持有效锚点 → Hook 必须变 None 且 DeptIDs 非空。
func TestDataScopePermission_AnchorWiring_SelfOverridden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.NewString()
	roleID := uuid.NewString()
	postID := uuid.NewString()
	gormDB := newAnchorWiringDB(t)

	require.NoError(t, gormDB.Exec(`INSERT INTO sys_role
		(id, role_name, role_key, data_scope, anchor_mode, anchor_param, status)
		VALUES (?, '仅本人角色', 'self_role', 5, 0, '', 0)`, roleID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID,
	).Error)
	require.NoError(t, gormDB.Exec(`INSERT INTO sys_post
		(id, post_code, post_name, anchor_mode, anchor_param, status, deleted_at)
		VALUES (?, 'mid_admin', '中支管理员', 5, '2', 0, NULL)`, postID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_post (user_id, post_id) VALUES (?, ?)`, userID, postID,
	).Error)
	require.NoError(t, gormDB.Exec(`UPDATE sys_user SET id = ? WHERE id = 'U-liu'`, userID).Error)

	mwCore := anchorWiringCore(gormDB)
	c, _ := newMWContext("GET", "/ops/building/list")
	c.Set("user_id", userID)

	DataScopePermission(mwCore)(c)

	got, ok := datascope.FromContext(c.Request.Context())
	require.True(t, ok)
	assert.False(t, got.All)
	assert.False(t, got.FailClosed)
	assert.Equal(t, datascope.HookNone, got.ResourceHook, "显式机构授予必须覆盖「仅本人」")
	assert.NotEmpty(t, got.DeptIDs, "Self 被锚点覆盖后必须有非空部门集")
	assert.Contains(t, got.DeptIDs, "D-qk")
}

// TestDataScopePermission_AnchorWiring_NoAnchorIsIdentity 用例 E（中间件级，C2 零回归）。
//
// 无任何锚点来源时，中间件注入的 ScopeResult 必须与 datascope.Resolve 原输出
// **逐字段相等**（UserID / All / DeptIDs / ResourceHook / FailClosed 五字段）。
func TestDataScopePermission_AnchorWiring_NoAnchorIsIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.NewString()
	roleID := uuid.NewString()
	gormDB := newAnchorWiringDB(t)

	require.NoError(t, gormDB.Exec(`INSERT INTO sys_role
		(id, role_name, role_key, data_scope, anchor_mode, anchor_param, status)
		VALUES (?, '本部门角色', 'dept_role', 3, 0, '', 0)`, roleID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID,
	).Error)
	require.NoError(t, gormDB.Exec(`UPDATE sys_user SET id = ? WHERE id = 'U-liu'`, userID).Error)

	mwCore := anchorWiringCore(gormDB)
	c, _ := newMWContext("GET", "/ops/building/list")
	c.Set("user_id", userID)

	DataScopePermission(mwCore)(c)

	got, ok := datascope.FromContext(c.Request.Context())
	require.True(t, ok)

	// 原 Resolve 输出（与中间件同一入参：userID + data_scope=3）
	want, err := datascope.Resolve(c.Request.Context(), gormDB, userID, models.DataScopeDept)
	require.NoError(t, err)
	assert.Equal(t, want, got, "无锚点来源时注入值必须与 Resolve 逐字段相等（C2 零回归）")
}
