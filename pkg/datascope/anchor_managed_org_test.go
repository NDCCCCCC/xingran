package datascope

// anchor_managed_org_test.go —— 「条线部门→管理归属机构」映射（C8 增补来源）契约回归。
//
// 场景：运营服务本部是独立条线，其下辖部门（如潜江团客共享作业服务中心 D-tx）
// 的 ancestors 链上没有任何中支 → 链上解析 miss。通过 scope='permission' 映射
// （D-tx → 潜江中支 D-qj）后：
//   方向①（miss 兜底）：条线部门的用户挂「中支管理员」岗位能锚定潜江中支；
//   方向②（可见集追加）：潜江中支管理员的可见集包含 D-tx。
//
// C8 关键守护：
//   - 别名表缺失 → 行为与 v1.54 完全一致（存量零回归）；
//   - 链上命中路径不受映射影响（纯加法，C2）；
//   - 映射指向机构已删除 → fail-closed，绝不回退 dept_id（C3）；
//   - scope='workstation' 行不被权限域消费（表复用隔离）。

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// newManagedOrgTestDB 在 anchor 基础夹具上追加条线场景：
//
//	D-ops  运营服务本部            level=0  ancestors="D-root"          ← 独立条线
//	D-tx   潜江团客共享作业服务中心  level=0  ancestors="D-root,D-ops"    ← 陈尔康在此
//	D-qj   潜江中支                level=2  ancestors="D-root"          ← 管理归属机构
//	U-chen 陈尔康                  dept=D-tx
//
// 默认【不建】sys_dept_location_alias 表 —— 用 TestManagedOrg_MissingTable_Degraded
// 单独守护缺表降级；需要映射的用例调 createManagedOrgTable(db) 自建。
func newManagedOrgTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newAnchorTestDB(t)

	require.NoError(t, db.Exec(`INSERT INTO sys_dept
		(id, dept_name, dept_code, parent_id, ancestors, org_level, is_external_org, status) VALUES
		('D-ops','运营服务本部','OPS',  'D-root','D-root',        0, 0, 0),
		('D-tx','潜江团客共享作业服务中心','QJTX','D-ops','D-root,D-ops',0, 0, 0),
		('D-qj','潜江中支','QJ',       'D-root','D-root',        2, 1, 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, dept_id, status)
		VALUES ('U-chen','chenkang','陈尔康','D-tx',0)`).Error)

	return db
}

// createManagedOrgTable 建 sys_dept_location_alias 最小夹具表（与生产列对齐）。
func createManagedOrgTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept_location_alias (
		id TEXT PRIMARY KEY,
		dept_id TEXT NOT NULL,
		location_id TEXT NOT NULL,
		scope TEXT NOT NULL DEFAULT 'workstation',
		remark TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
		created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0
	)`).Error)
}

// insertManagedOrg 插入一条 scope='permission' 映射行。
func insertManagedOrg(t *testing.T, db *gorm.DB, id, deptID, orgID, scope string) {
	t.Helper()
	if scope == "" {
		scope = "permission"
	}
	require.NoError(t, db.Exec(`INSERT INTO sys_dept_location_alias
		(id, dept_id, location_id, scope) VALUES (?, ?, ?, ?)`,
		id, deptID, orgID, scope).Error)
}

// C8-③ 守护：别名表缺失 → 行为与 v1.54 逐字一致（miss → 原 reason，零回归）。
func TestManagedOrg_MissingTable_Degraded(t *testing.T) {
	db := newManagedOrgTestDB(t) // 不建 alias 表
	ctx := context.Background()

	anchorID, resolved, reason, err := ResolveAnchor(ctx, db, "U-chen", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	assert.False(t, resolved)
	assert.Empty(t, anchorID)
	assert.Equal(t, AnchorReasonLevelMiss, reason, "缺表必须降级为原 miss，不得报错")
}

// C8 方向①：链上 miss 后映射兜底 → 陈尔康锚定潜江中支。
func TestManagedOrg_MissFallback_ResolvesMappedOrg(t *testing.T) {
	db := newManagedOrgTestDB(t)
	ctx := context.Background()
	createManagedOrgTable(t, db)
	insertManagedOrg(t, db, "A-1", "D-tx", "D-qj", "")

	anchorID, resolved, reason, err := ResolveAnchor(ctx, db, "U-chen", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	require.True(t, resolved, "陈尔康应经映射兜底锚定潜江中支，reason=%s", reason)
	assert.Equal(t, "D-qj", anchorID)
}

// C2 纯加法守护：链上命中的用户（刘洋→武汉中支）不受映射影响。
func TestManagedOrg_ChainHit_Unaffected(t *testing.T) {
	db := newManagedOrgTestDB(t)
	ctx := context.Background()
	createManagedOrgTable(t, db)
	// 干扰行：给综合管理部配一条映射（但刘洋链上命中武汉中支，不应走映射）
	insertManagedOrg(t, db, "A-2", "D-zh", "D-qj", "")

	anchorID, resolved, _, err := ResolveAnchor(ctx, db, "U-liu", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	require.True(t, resolved)
	assert.Equal(t, "D-wh", anchorID, "链上命中绝不替换为映射锚点（C8-①）")
}

// C3 守护：映射指向的机构已删除 → fail-closed + AnchorReasonManagedOrgInvalid。
func TestManagedOrg_InvalidOrg_FailClosed(t *testing.T) {
	db := newManagedOrgTestDB(t)
	ctx := context.Background()
	createManagedOrgTable(t, db)
	insertManagedOrg(t, db, "A-3", "D-tx", "D-gone", "") // D-gone 不存在

	anchorID, resolved, reason, err := ResolveAnchor(ctx, db, "U-chen", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	assert.False(t, resolved)
	assert.Empty(t, anchorID, "映射失效必须 fail-closed，绝不回退 dept_id")
	assert.Equal(t, AnchorReasonManagedOrgInvalid, reason)
}

// 表复用隔离：scope='workstation' 行不被权限域消费。
func TestManagedOrg_WorkstationScope_Ignored(t *testing.T) {
	db := newManagedOrgTestDB(t)
	ctx := context.Background()
	createManagedOrgTable(t, db)
	insertManagedOrg(t, db, "A-4", "D-tx", "D-qj", "workstation") // 工位行

	anchorID, resolved, reason, err := ResolveAnchor(ctx, db, "U-chen", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	assert.False(t, resolved, "workstation 行不得作为权限映射消费")
	assert.Equal(t, AnchorReasonLevelMiss, reason)
	assert.Empty(t, anchorID)
}

// C8 方向②：潜江中支管理员的可见集包含条线部门 D-tx。
func TestManagedOrg_GrantsIncludeManagedDepts(t *testing.T) {
	db := newManagedOrgTestDB(t)
	ctx := context.Background()
	createManagedOrgTable(t, db)
	insertManagedOrg(t, db, "A-5", "D-tx", "D-qj", "")
	// 干扰：挂到武汉中支的条线部门不得混入潜江的可见集
	require.NoError(t, db.Exec(`INSERT INTO sys_dept
		(id, dept_name, dept_code, parent_id, ancestors, org_level, is_external_org, status) VALUES
		('D-tk','团客运营服务部','TK','D-ops','D-root,D-ops',0,0,0)`).Error)
	insertManagedOrg(t, db, "A-6", "D-tk", "D-wh", "")

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, anchor_mode, anchor_param, status)
		VALUES ('R-qj','潜江中支管理员','qj_admin', 5, '2', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-chen','R-qj')`).Error)

	ids, err := ResolveAnchorGrants(ctx, db, "U-chen")
	require.NoError(t, err)
	assert.Contains(t, ids, "D-qj", "锚点机构子树")
	assert.Contains(t, ids, "D-tx", "映射到潜江中支的条线部门必须可见（方向②）")
	assert.NotContains(t, ids, "D-tk", "挂到其它机构的条线部门不得混入")
}

// 方向② + 方向① 组合：陈尔康自己挂「中支管理员」→ 锚定潜江中支，
// 可见集 = 潜江子树 ∪ 条线部门（含他自己所在的 D-tx）。
func TestManagedOrg_LineMember_AsBranchAdmin(t *testing.T) {
	db := newManagedOrgTestDB(t)
	ctx := context.Background()
	createManagedOrgTable(t, db)
	insertManagedOrg(t, db, "A-7", "D-tx", "D-qj", "")

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, anchor_mode, anchor_param, status)
		VALUES ('R-qj2','中支管理员','branch_admin', 5, '2', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-chen','R-qj2')`).Error)

	ids, err := ResolveAnchorGrants(ctx, db, "U-chen")
	require.NoError(t, err)
	assert.Contains(t, ids, "D-qj")
	assert.Contains(t, ids, "D-tx", "条线成员当管理员：可见集含其条线部门自身")
}

// 专用内存库（无 anchor 夹具依赖）验证缺表降级不影响 queryManagedOrgDeptIDs。
func TestManagedOrg_QueryDeptIDs_MissingTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mo.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	ids, qErr := queryManagedOrgDeptIDs(context.Background(), db, "D-qj")
	require.NoError(t, qErr, "缺表必须降级为空集而非报错")
	assert.Empty(t, ids)
}

// 多对一守护：多个条线部门可归属同一机构（如 团客/个客运营服务部 → 分公司本部）。
// partial unique 是 (dept_id, scope) —— 只限制"一个部门一个归属"，
// 不限制"一个机构接收多个部门"；方向②查询按 location_id 取回全部映射部门。
func TestManagedOrg_ManyDeptsToOneOrg(t *testing.T) {
	db := newManagedOrgTestDB(t)
	ctx := context.Background()
	createManagedOrgTable(t, db)

	// 两个条线部门 → 同一机构 D-qj
	require.NoError(t, db.Exec(`INSERT INTO sys_dept
		(id, dept_name, dept_code, parent_id, ancestors, org_level, is_external_org, status) VALUES
		('D-b','运营服务部子部门B','OPS-B','D-ops','D-root,D-ops',0,0,0)`).Error)
	insertManagedOrg(t, db, "A-m1", "D-tx", "D-qj", "")
	insertManagedOrg(t, db, "A-m2", "D-b", "D-qj", "")

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, anchor_mode, anchor_param, status)
		VALUES ('R-qj3','潜江中支管理员','qj_admin3', 5, '2', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-chen','R-qj3')`).Error)

	ids, err := ResolveAnchorGrants(ctx, db, "U-chen")
	require.NoError(t, err)
	assert.Contains(t, ids, "D-tx", "条线部门A（多对一之一）")
	assert.Contains(t, ids, "D-b", "条线部门B（多对一之二）")
	assert.Contains(t, ids, "D-qj")
}
