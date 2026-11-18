package datascope

// anchor_coverage_test.go —— 补齐 anchor.go 在 PR diff 中未覆盖的分支。
//
// 与 anchor_test.go 共用 newAnchorTestDB 夹具；本文件聚焦：
//   - 纯函数 splitAncestors / walkAncestorsNearest / isMissingTableError 的边界分支
//   - ResolveAnchor / ResolveAnchorByDept 的 None / BadMode / BadParam / DeptMissing /
//     ExternalMiss / LevelMiss / NoDeptRow 等分支
//   - query* 辅助的空参数早退与错误传播
//   - queryAnchorGrants 的 role 来源错误、post 缺表降级
//   - ResolveAnchorGrants 单来源失败跳过、子树展开失败跳过
//   - PreviewAnchor 的错误传播与空 reason 兜底文案

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// newBareCoverDB 建一个不建任何表的 SQLite DB，并保证测试结束时关闭连接池
// （Windows 下不关闭会导致 t.TempDir() 清理失败）。
func newBareCoverDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), name)), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestSplitAncestorsEdgeCases(t *testing.T) {
	assert.Nil(t, splitAncestors(""))
	assert.Equal(t, []string{"a", "b"}, splitAncestors(", a, ,0, b, 0,"))
	assert.Equal(t, []string{"D-root"}, splitAncestors("0,D-root,0"))
}

func TestWalkAncestorsNearestErrorAndReverse(t *testing.T) {
	// 由近到远：ancestors 根→父序，父先被访问。
	var visited []string
	got, ok, err := walkAncestorsNearest([]string{"root", "mid", "parent"},
		func(id string) (bool, error) {
			visited = append(visited, id)
			return id == "mid", nil
		})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "mid", got)
	assert.Equal(t, []string{"parent", "mid"}, visited, "必须由近到远遍历")

	// match 返回 error 必须立即中断并向外传播。
	sentinel := errors.New("boom")
	_, ok, err = walkAncestorsNearest([]string{"a", "b"},
		func(string) (bool, error) { return false, sentinel })
	assert.False(t, ok)
	assert.ErrorIs(t, err, sentinel)

	// 全部未命中。
	_, ok, err = walkAncestorsNearest([]string{"a", "b"},
		func(string) (bool, error) { return false, nil })
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestIsMissingTableError(t *testing.T) {
	assert.False(t, isMissingTableError(nil))
	for _, msg := range []string{
		"no such table: sys_user_post",
		"relation sys_post does not exist",
		"ERROR: undefined table: 7",
		"NO SUCH TABLE: X",
	} {
		assert.True(t, isMissingTableError(errors.New(msg)), msg)
	}
	assert.False(t, isMissingTableError(errors.New("connection refused")))
}

func TestQueryHelpersEmptyArgs(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	deptID, ancestors, ok, err := queryUserDeptFull(ctx, db, "")
	require.NoError(t, err)
	assert.Empty(t, deptID)
	assert.Empty(t, ancestors)
	assert.False(t, ok)

	lv, err := queryDeptLevel(ctx, db, "")
	require.NoError(t, err)
	assert.Equal(t, models.OrgLevelNone, lv)

	ext, err := queryDeptIsExternal(ctx, db, "")
	require.NoError(t, err)
	assert.False(t, ext)

	exists, err := deptExists(ctx, db, "")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestQueryHelpersErrorsOnMissingTable(t *testing.T) {
	// 不建任何表的裸 DB：query* 应把底层 SQL 错误包装后返回。
	db := newBareCoverDB(t, "empty.db")
	ctx := context.Background()

	_, _, _, err := queryUserDeptFull(ctx, db, "U-1")
	assert.Error(t, err)
	_, err = queryDeptLevel(ctx, db, "D-1")
	assert.Error(t, err)
	_, err = queryDeptIsExternal(ctx, db, "D-1")
	assert.Error(t, err)
	_, err = deptExists(ctx, db, "D-1")
	assert.Error(t, err)
	_, err = queryDeptChildIDs(ctx, db, "D-1")
	assert.Error(t, err)

	_, _, _, err = ResolveAnchor(ctx, db, "U-1", models.AnchorModeExternalOrg, "")
	assert.Error(t, err)
	_, _, _, err = ResolveAnchor(ctx, db, "U-1", models.AnchorModeLevel, "2")
	assert.Error(t, err)
	_, _, _, err = ResolveAnchor(ctx, db, "U-1", models.AnchorModeDept, "D-1")
	assert.Error(t, err)
}

func TestQueryUserDeptFull_UserDeptRowMissing(t *testing.T) {
	db := newAnchorTestDB(t)
	// 用户存在但 dept_id 指向不存在的部门 → ok=false（脏数据不得回退）。
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, dept_id) VALUES ('U-ghost','ghost','D-ghost')`).Error)
	_, _, ok, err := queryUserDeptFull(context.Background(), db, "U-ghost")
	require.NoError(t, err)
	assert.False(t, ok, "用户 dept_id 指向不存在的部门行时不得视为有效部门链")
}

func TestResolveAnchorByDeptBranches(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// mode=0：不解析。
	id, ok, reason, err := ResolveAnchorByDept(ctx, db, "D-zh", "D-root,D-wh", models.AnchorModeNone, "ignored")
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.False(t, ok)
	assert.Equal(t, AnchorOK, reason)

	// 非法 mode：BadMode。
	_, ok, reason, err = ResolveAnchorByDept(ctx, db, "D-zh", "D-root,D-wh", models.AnchorMode(4), "")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonBadMode, reason)

	// 模式2：参数空 → BadParam；部门不存在 → DeptMissing；存在 → 命中。
	_, ok, reason, err = ResolveAnchorByDept(ctx, db, "D-zh", "", models.AnchorModeDept, "  ")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonBadParam, reason)

	_, ok, reason, err = ResolveAnchorByDept(ctx, db, "D-zh", "", models.AnchorModeDept, "D-nope")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonDeptMissing, reason)

	id, ok, reason, err = ResolveAnchorByDept(ctx, db, "D-zh", "", models.AnchorModeDept, " D-wh ")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "D-wh", id)
	assert.Equal(t, AnchorOK, reason)

	// deptID 为空且非模式2 → NoDept。
	_, ok, reason, err = ResolveAnchorByDept(ctx, db, "", "", models.AnchorModeExternalOrg, "")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonNoDept, reason)

	// 外部机构链上无命中 → ExternalMiss；层级参数非法 → BadParam。
	_, ok, _, err = ResolveAnchorByDept(ctx, db, "D-zh", "D-root,D-wh", models.AnchorModeExternalOrg, "")
	require.NoError(t, err)
	assert.True(t, ok, "D-wh 是外部机构，应命中")

	_, ok, reason, err = ResolveAnchorByDept(ctx, db, "D-zh", "D-root,D-wh", models.AnchorModeLevel, "abc")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonBadParam, reason)
}

func TestResolveAnchor_ExtraBranches(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// mode=0 → 不解析、无 reason。
	id, ok, reason, err := ResolveAnchor(ctx, db, "U-liu", models.AnchorModeNone, "anything")
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.False(t, ok)
	assert.Equal(t, AnchorOK, reason)

	// 非法 mode（1/4/未知）→ BadMode。
	for _, m := range []models.AnchorMode{1, 4, 9} {
		_, ok, reason, err = ResolveAnchor(ctx, db, "U-liu", m, "")
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, AnchorReasonBadMode, reason)
	}

	// 模式2：参数空 / 部门不存在。
	_, ok, reason, err = ResolveAnchor(ctx, db, "U-liu", models.AnchorModeDept, "  ")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonBadParam, reason)

	_, ok, reason, err = ResolveAnchor(ctx, db, "U-liu", models.AnchorModeDept, "D-nope")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonDeptMissing, reason)

	// 用户 dept_id 指向不存在的部门行 → NoDept（不回退、不 panic）。
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, dept_id) VALUES ('U-ghost','ghost','D-ghost')`).Error)
	_, ok, reason, err = ResolveAnchor(ctx, db, "U-ghost", models.AnchorModeExternalOrg, "")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonNoDept, reason)

	// 外部机构链上无命中 → ExternalMiss（去掉所有外部机构标记）。
	require.NoError(t, db.Exec(`UPDATE sys_dept SET is_external_org = 0`).Error)
	_, ok, reason, err = ResolveAnchor(ctx, db, "U-liu", models.AnchorModeExternalOrg, "")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonExternalMiss, reason)
}

func TestValidateAnchor_ExternalOrgParamAndDBError(t *testing.T) {
	db := newAnchorTestDB(t)
	// mode=3 带参数 → 拒绝。
	err := ValidateAnchor(context.Background(), db, 3, "D-wh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不需要参数")

	// mode=2 但 sys_dept 缺列（查询报错）→ 包装错误透出。
	bare := newBareCoverDB(t, "bare.db")
	require.NoError(t, bare.Exec(`CREATE TABLE sys_dept (id TEXT PRIMARY KEY)`).Error)
	err = ValidateAnchor(context.Background(), bare, 2, "D-wh")
	assert.Error(t, err)
	err = ValidateAnchor(context.Background(), bare, 3, "")
	assert.Error(t, err) // is_external_org 列不存在
	err = ValidateAnchor(context.Background(), bare, 5, "2")
	assert.Error(t, err) // org_level 列不存在
}

func TestQueryAnchorGrants_RoleAndPostSources(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 角色来源：启用 + anchor_mode<>0 → 一条 role grant；停用角色被忽略。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R-on','中支角色',5,'2',0), ('R-off','停用角色',5,'2',1), ('R-no-anchor','普通角色',0,'',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES
		('U-liu','R-on'), ('U-liu','R-off'), ('U-liu','R-no-anchor')`).Error)
	// 岗位来源同理。
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_name, anchor_mode, anchor_param, status)
		VALUES ('P-on','外部岗位',3,'',0), ('P-off','停用岗位',3,'',1), ('P-no-anchor','普通岗位',0,'',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES
		('U-liu','P-on'), ('U-liu','P-off'), ('U-liu','P-no-anchor')`).Error)

	grants, err := queryAnchorGrants(ctx, db, "U-liu")
	require.NoError(t, err)
	assert.Len(t, grants, 2)
	bySource := map[string]AnchorGrant{}
	for _, g := range grants {
		bySource[g.Source] = g
	}
	assert.Equal(t, "中支角色", bySource["role"].SourceName)
	assert.Equal(t, models.AnchorModeLevel, bySource["role"].Mode)
	assert.Equal(t, "2", bySource["role"].Param)
	assert.Equal(t, "外部岗位", bySource["post"].SourceName)

	// 空 userID → 无来源。
	grants, err = queryAnchorGrants(ctx, db, "")
	require.NoError(t, err)
	assert.Empty(t, grants)
}

func TestQueryAnchorGrants_PostTableMissingDegrades(t *testing.T) {
	// 只建角色侧表，缺 sys_user_post / sys_post：岗位查询失败应降级为空、不报错。
	db := newBareCoverDB(t, "no-post.db")
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, ancestors TEXT, org_level INTEGER, is_external_org INTEGER, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user (id TEXT PRIMARY KEY, dept_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_role (id TEXT PRIMARY KEY, role_name TEXT, anchor_mode INTEGER, anchor_param TEXT, status INTEGER, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)

	// 角色来源仍可用；岗位缺表仅告警降级。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R','r',0,'',0)`).Error) // 无锚点角色，不产生 grant
	grants, err := queryAnchorGrants(context.Background(), db, "U-1")
	require.NoError(t, err, "缺 sys_user_post 不得中断角色侧解析")
	assert.Empty(t, grants)
}

func TestResolveAnchorGrants_SkipsBrokenSources(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 一个来源：mode=2 但 param 指向不存在的部门 → 解析失败（未命中），跳过；
	// 另一个来源：mode=5 param=2 正常命中，保留。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R-bad','坏锚点',2,'D-nope',0), ('R-good','好锚点',5,'2',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES
		('U-liu','R-bad'), ('U-liu','R-good')`).Error)

	ids, err := ResolveAnchorGrants(ctx, db, "U-liu")
	require.NoError(t, err)
	assert.Contains(t, ids, "D-wh")
	assert.NotContains(t, ids, "D-nope")
}

func TestResolveAnchorGrants_QueryErrorAndTreeErrorSkip(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 角色表缺失 → queryAnchorGrants 直接报错，向外传播。
	require.NoError(t, db.Exec(`DROP TABLE sys_role`).Error)
	_, err := ResolveAnchorGrants(ctx, db, "U-liu")
	assert.Error(t, err)
}

func TestPreviewAnchor_ErrorsAndFallbackReason(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 空用户列表：计数为 0，Items 为空非 nil。
	res, err := PreviewAnchor(ctx, db, models.AnchorModeLevel, "2", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, res.TotalCount)
	assert.NotNil(t, res.Items)
	assert.Empty(t, res.Items)

	// mode=0：未命中但 reason 为空 → 兜底文案。
	res, err = PreviewAnchor(ctx, db, models.AnchorModeNone, "ignored",
		[]string{"U-liu"}, map[string]string{"U-liu": "liuyang"}, map[string]string{"U-liu": "刘洋"})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.False(t, res.Items[0].Resolved)
	assert.Equal(t, "未能解析到锚点", res.Items[0].Reason)
	assert.Equal(t, 1, res.UnresolvedCount)
	assert.Equal(t, "liuyang", res.Items[0].Username)
	assert.Equal(t, "刘洋", res.Items[0].Nickname)

	// DB 裸库（缺 sys_user）→ ResolveAnchor 报错向外传播。
	bare := newBareCoverDB(t, "preview-bare.db")
	_, err = PreviewAnchor(ctx, bare, models.AnchorModeExternalOrg, "", []string{"U-1"}, nil, nil)
	assert.Error(t, err)
}

func TestMerge_NoDuplicateAndSort(t *testing.T) {
	got := Merge(ScopeResult{UserID: "u", DeptIDs: []string{"b", "a"}}, []string{"a", "c", "", " b "})
	assert.Equal(t, []string{"a", "b", "c"}, got.DeptIDs)
	assert.False(t, got.FailClosed)
	assert.Equal(t, HookNone, got.ResourceHook)
}

func TestResolveAnchorByDept_DeptExistsError(t *testing.T) {
	// 模式2 下 deptExists 查询失败（缺 sys_dept）→ 错误向外传播（行 306-308）。
	db := newBareCoverDB(t, "by-dept-err.db")
	_, _, reason, err := ResolveAnchorByDept(context.Background(), db, "D-x", "", models.AnchorModeDept, "D-x")
	assert.Error(t, err)
	assert.Equal(t, AnchorOK, reason)
}

func TestQueryAnchorGrants_RoleQueryError(t *testing.T) {
	// 缺 sys_user_role → 角色来源查询失败，错误向外传播（行 499-500）。
	db := newBareCoverDB(t, "grant-role-err.db")
	_, err := queryAnchorGrants(context.Background(), db, "U-1")
	assert.Error(t, err)
}

func TestPreviewAnchor_ResolveErrorForModeNone(t *testing.T) {
	// mode=0 也走 ResolveAnchor（返回未命中），确认空 reason 兜底路径不受 DB 影响（行 690-692）。
	db := newAnchorTestDB(t)
	res, err := PreviewAnchor(context.Background(), db, models.AnchorModeNone, "",
		[]string{"U-liu", "U-wang"}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, res.UnresolvedCount)
	assert.Equal(t, "未能解析到锚点", res.Items[0].Reason)
}
