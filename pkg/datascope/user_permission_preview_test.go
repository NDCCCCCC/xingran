package datascope

// user_permission_preview_test.go —— 用户权限预览（菜单权限 + 数据范围）回归。
//
// 需求（用户 2026-11-17）：编辑用户加"权限预览"，除菜单权限也要显示权限范围。
// 关键契约：**预览结果 = 保存后运行期结果**——复用同一锚点解析内核。
//
// 覆盖：
//  1. All 角色 → all=true、锚点不参与、notes 有说明，权限码汇总正确；
//  2. 仅本人 + 中支岗位锚点 → 可见部门 = 武汉中支子树（含硚口支公司）；
//  3. 仅本人 + 中支角色锚点 → 与岗位路径结果一致（两条来源等价）；
//  4. 未命中（target 层级在链上不存在）→ resolved=false + reason，可见集空；
//  5. 角色维度 DeptChild 以表单 deptId 为锚点（未保存值也能预览）；
//  6. 菜单权限去重排序、角色/岗位名回显。

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

func newPreviewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "preview.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	for _, ddl := range []string{
		`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, ancestors TEXT DEFAULT '',
			org_level INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_role (id TEXT PRIMARY KEY, role_name TEXT, data_scope INTEGER DEFAULT 1,
			anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '', status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_post (id TEXT PRIMARY KEY, post_name TEXT, anchor_mode INTEGER DEFAULT 0,
			anchor_param TEXT DEFAULT '', status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_role_menu (role_id TEXT, menu_id TEXT)`,
		`CREATE TABLE sys_menu (id TEXT PRIMARY KEY, perms TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}

	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, ancestors, org_level, status) VALUES
		('D-hb','湖北分公司','',1,0),
		('D-wh','武汉中心支公司','D-hb',2,0),
		('D-zh','综合管理部','D-hb,D-wh',0,0),
		('D-qk','硚口支公司','D-hb,D-wh',3,0),
		('D-cs','长沙中心支公司','D-hb',2,0)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, perms, status) VALUES
		('m1','system:user:list',0), ('m2','system:user:edit',0), ('m3','ops:device:list',0)`).Error)
	return db
}

// TestPreviewUserScope_AllRole 全库角色 → All 优先、锚点不参与、权限汇总。
func TestPreviewUserScope_AllRole(t *testing.T) {
	db := newPreviewDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id,role_name,data_scope,anchor_mode,anchor_param,status)
		VALUES ('R-admin','超级管理员',1,0,'',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id,menu_id) VALUES ('R-admin','m2'),('R-admin','m1'),('R-admin','m3')`).Error)

	res, err := PreviewUserScope(ctx, db, "D-zh", []string{"R-admin"}, nil)
	require.NoError(t, err)
	assert.True(t, res.All)
	assert.Equal(t, models.DataScopeAll, res.DataScope)
	assert.Equal(t, "全部数据", res.DataScopeLabel)
	assert.Empty(t, res.VisibleDeptIDs, "All 档不应给出具体可见集")
	assert.Equal(t, []string{"ops:device:list", "system:user:edit", "system:user:list"}, res.Permissions, "权限码去重排序")
	assert.True(t, len(res.Notes) > 0 && res.Notes[0] != "", "All 档应有说明")
	assert.True(t, res.AllDepts, "All 档部门树应标记全库可见")
	assert.Empty(t, res.DeptIds, "All 档 DeptIds 为空，前端渲染为全选提示")
	assert.Equal(t, []string{"m1", "m2", "m3"}, res.MenuIds, "菜单树勾选集（含权限码对应的 menu id）")
}

// TestPreviewUserScope_SelfWithPostLevelAnchor 仅本人 + 中支岗位锚点 → 可见集 = 武汉子树。
func TestPreviewUserScope_SelfWithPostLevelAnchor(t *testing.T) {
	db := newPreviewDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id,role_name,data_scope,status) VALUES ('R-self','普通用户',5,0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id,post_name,anchor_mode,anchor_param,status)
		VALUES ('P-mid','中支管理员',5,'2',0)`).Error)

	res, err := PreviewUserScope(ctx, db, "D-zh", []string{"R-self"}, []string{"P-mid"})
	require.NoError(t, err)
	assert.False(t, res.All)
	assert.Equal(t, models.DataScopeSelf, res.DataScope)
	require.Len(t, res.AnchorSources, 1)
	src := res.AnchorSources[0]
	assert.Equal(t, "post", src.Source)
	assert.Equal(t, "中支管理员", src.SourceName)
	assert.True(t, src.Resolved)
	assert.Equal(t, "D-wh", src.AnchorDeptID)
	assert.Equal(t, "武汉中心支公司", src.AnchorDeptName)
	assert.Equal(t, 3, src.DeptCount) // 武汉中支 + 综合管理部 + 硚口支公司
	assert.ElementsMatch(t, []string{"D-wh", "D-zh", "D-qk"}, res.VisibleDeptIDs)
	assert.Contains(t, res.VisibleDeptNames, "硚口支公司")
}

// TestPreviewUserScope_RoleAnchorEqualsPostAnchor 角色锚点与岗位锚点结果一致（两条来源等价）。
func TestPreviewUserScope_RoleAnchorEqualsPostAnchor(t *testing.T) {
	db := newPreviewDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id,role_name,data_scope,anchor_mode,anchor_param,status)
		VALUES ('R-mid','机构管理员',5,5,'2',0)`).Error)

	res, err := PreviewUserScope(ctx, db, "D-zh", []string{"R-mid"}, nil)
	require.NoError(t, err)
	require.Len(t, res.AnchorSources, 1)
	assert.Equal(t, "role", res.AnchorSources[0].Source)
	assert.Equal(t, "D-wh", res.AnchorSources[0].AnchorDeptID)
	assert.ElementsMatch(t, []string{"D-wh", "D-zh", "D-qk"}, res.VisibleDeptIDs)
}

// TestPreviewUserScope_UnmatchedLevel 目标层级在链上不存在 → 未命中 + 原因，可见集空。
func TestPreviewUserScope_UnmatchedLevel(t *testing.T) {
	db := newPreviewDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id,role_name,data_scope,status) VALUES ('R-self','普通用户',5,0)`).Error)
	// 支公司级锚点：刘洋在综合管理部，链上无 level=3
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id,post_name,anchor_mode,anchor_param,status)
		VALUES ('P-off','支公司管理员',5,'3',0)`).Error)

	res, err := PreviewUserScope(ctx, db, "D-zh", []string{"R-self"}, []string{"P-off"})
	require.NoError(t, err)
	require.Len(t, res.AnchorSources, 1)
	assert.False(t, res.AnchorSources[0].Resolved)
	assert.Contains(t, res.AnchorSources[0].Reason, "没有指定层级")
	assert.Empty(t, res.VisibleDeptIDs)
}

// TestPreviewUserScope_DeptChildUsesFormDept 角色 DeptChild 档以**表单 deptId** 为锚点（未保存值）。
func TestPreviewUserScope_DeptChildUsesFormDept(t *testing.T) {
	db := newPreviewDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id,role_name,data_scope,status)
		VALUES ('R-deptchild','本部门及下级',4,0)`).Error)

	// 表单 deptId 选的是武汉中支（尚未保存），应能预览到子树——不能依赖库里 user 的 dept_id。
	res, err := PreviewUserScope(ctx, db, "D-wh", []string{"R-deptchild"}, nil)
	require.NoError(t, err)
	assert.Equal(t, models.DataScopeDeptChild, res.DataScope)
	assert.ElementsMatch(t, []string{"D-wh", "D-zh", "D-qk"}, res.VisibleDeptIDs)
}

// TestPreviewUserScope_NamesAndNoData All 档之外角色/岗位名回显、无权限时空数组非 nil。
func TestPreviewUserScope_NamesAndNoData(t *testing.T) {
	db := newPreviewDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id,role_name,data_scope,status) VALUES ('R-self','普通用户',5,0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id,post_name,anchor_mode,anchor_param,status)
		VALUES ('P-mid','中支管理员',5,'2',0)`).Error)

	res, err := PreviewUserScope(ctx, db, "D-zh", []string{"R-self"}, []string{"P-mid"})
	require.NoError(t, err)
	assert.Equal(t, []string{"普通用户"}, res.RoleNames)
	assert.Equal(t, []string{"中支管理员"}, res.PostNames)
	assert.NotNil(t, res.Permissions)
	assert.Len(t, res.Permissions, 0, "未配菜单权限时为空数组（非 nil）")
}
