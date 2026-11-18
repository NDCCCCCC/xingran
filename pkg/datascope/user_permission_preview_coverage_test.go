package datascope

// user_permission_preview_coverage_test.go —— 补齐 PreviewUserScope / 标签函数的
// PR diff 未覆盖分支：无角色（Self）、Dept 档、Custom 档、标签 default、All 分支
// MenuIds 为 nil 的兜底，以及各查询出错路径。

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func newPreviewCoverageDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "preview-cov.db")), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
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
		`CREATE TABLE sys_role_dept (role_id TEXT, dept_id TEXT)`,
		`CREATE TABLE sys_menu (id TEXT PRIMARY KEY, perms TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, ancestors, org_level, status) VALUES
		('D-hb','湖北分公司','',1,0),
		('D-wh','武汉中支','D-hb',2,0),
		('D-zh','综合管理部','D-hb,D-wh',0,0),
		('D-qk','硚口支公司','D-hb,D-wh',3,0)`).Error)
	return db
}

func TestDataScopeLabelAndAnchorModeLabelDefaults(t *testing.T) {
	cases := []struct {
		scope models.DataScope
		want  string
	}{
		{models.DataScopeAll, "全部数据"},
		{models.DataScopeCustom, "自定义数据"},
		{models.DataScopeDept, "本部门数据"},
		{models.DataScopeDeptChild, "本部门及子部门数据"},
		{models.DataScopeSelf, "仅本人数据"},
		{models.DataScope(99), "未知档位(99)"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, DataScopeLabel(c.scope))
	}

	modes := []struct {
		mode models.AnchorMode
		want string
	}{
		{models.AnchorModeNone, "不参与"},
		{models.AnchorModeDept, "指定部门（含下级）"},
		{models.AnchorModeExternalOrg, "外部机构（最近）"},
		{models.AnchorModeLevel, "指定机构层级（最近）"},
		{models.AnchorMode(9), "未知模式(9)"},
	}
	for _, c := range modes {
		assert.Equal(t, c.want, AnchorModeLabel(c.mode))
	}
}

func TestPreviewUserScope_NoRolesAndEmptyDept(t *testing.T) {
	db := newPreviewCoverageDB(t)
	res, err := PreviewUserScope(context.Background(), db, "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, models.DataScopeSelf, res.DataScope, "无角色 → 仅本人")
	assert.Equal(t, "仅本人数据", res.DataScopeLabel)
	assert.False(t, res.All)
	assert.Empty(t, res.AnchorSources)
	assert.Empty(t, res.VisibleDeptIDs)
	assert.Contains(t, res.Notes[0], "仅本人") // 未解析到任何锚点
	assert.Nil(t, res.Permissions, "无 roleIDs 时不查菜单权限，保持 nil（与调用方约定一致）")
	assert.Nil(t, res.RoleNames)
	assert.Nil(t, res.PostNames)
}

func TestPreviewUserScope_DeptScopeUsesFormDept(t *testing.T) {
	db := newPreviewCoverageDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, data_scope, status)
		VALUES ('R-dept','本部门角色',3,0)`).Error)

	res, err := PreviewUserScope(context.Background(), db, "D-zh", []string{"R-dept"}, nil)
	require.NoError(t, err)
	assert.Equal(t, models.DataScopeDept, res.DataScope)
	assert.Equal(t, []string{"D-zh"}, res.VisibleDeptIDs)
	assert.Equal(t, []string{"D-zh"}, res.DeptIds)
	assert.Equal(t, []string{"综合管理部"}, res.VisibleDeptNames)
}

func TestPreviewUserScope_CustomUsesRoleDept(t *testing.T) {
	db := newPreviewCoverageDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, data_scope, status)
		VALUES ('R-custom','自定义角色',2,0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES
		('R-custom','D-wh'), ('R-custom','D-qk')`).Error)

	res, err := PreviewUserScope(context.Background(), db, "D-zh", []string{"R-custom"}, nil)
	require.NoError(t, err)
	assert.Equal(t, models.DataScopeCustom, res.DataScope)
	assert.Equal(t, []string{"D-qk", "D-wh"}, res.VisibleDeptIDs)
}

func TestPreviewUserScope_AllRoleWithoutMenus(t *testing.T) {
	db := newPreviewCoverageDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, data_scope, status)
		VALUES ('R-admin','超管',1,0)`).Error)

	res, err := PreviewUserScope(context.Background(), db, "D-zh", []string{"R-admin"}, nil)
	require.NoError(t, err)
	assert.True(t, res.All)
	assert.True(t, res.AllDepts)
	assert.NotNil(t, res.MenuIds, "All 分支无菜单时也要返回空切片而非 nil")
	assert.Empty(t, res.MenuIds)
	assert.Empty(t, res.DeptIds)
}

func TestPreviewUserScope_UnknownScopeHasNoDeptSources(t *testing.T) {
	db := newPreviewCoverageDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, data_scope, status)
		VALUES ('R-weird','异常档位',99,0)`).Error)

	res, err := PreviewUserScope(context.Background(), db, "D-zh", []string{"R-weird"}, nil)
	require.NoError(t, err)
	assert.Equal(t, models.DataScope(99), res.DataScope)
	assert.Equal(t, "未知档位(99)", res.DataScopeLabel)
	assert.False(t, res.All)
	assert.Empty(t, res.VisibleDeptIDs)
}

func TestPreviewUserScope_ErrorsOnMissingTables(t *testing.T) {
	bare := newBareCoverDB(t, "preview-bare.db")
	ctx := context.Background()

	// 角色表缺失 → MAX(data_scope) 查询报错。
	_, err := PreviewUserScope(ctx, bare, "", []string{"R"}, nil)
	assert.Error(t, err)

	// 建 sys_role、sys_role_menu、sys_menu：部门链查询缺 sys_dept → 报错。
	require.NoError(t, bare.Exec(`CREATE TABLE sys_role (id TEXT PRIMARY KEY, data_scope INTEGER, status INTEGER, deleted_at DATETIME)`).Error)
	_, err = PreviewUserScope(ctx, bare, "D-1", nil, nil)
	assert.Error(t, err, "查询部门祖先链失败应返回 error")
}

func TestPreviewUserScope_SourceQueryErrors(t *testing.T) {
	ctx := context.Background()

	// 建 sys_dept + sys_user（部门链查询可过），但缺 sys_role → 角色档位查询报错。
	db := newBareCoverDB(t, "src-role-err.db")
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, ancestors TEXT, deleted_at DATETIME, dept_name TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_role (id TEXT PRIMARY KEY, data_scope INTEGER, status INTEGER, deleted_at DATETIME, role_name TEXT, anchor_mode INTEGER, anchor_param TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_menu (id TEXT PRIMARY KEY, perms TEXT, status INTEGER, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_role_menu (role_id TEXT, menu_id TEXT)`).Error)
	// 缺 sys_role_menu 的锚点查询？此处 role 表存在但 sys_role_menu 有 → 走正常；
	// 为触发行 176/177，DROP sys_role 后 roleIDs 查询报错（但第一步 MAX 也报错）。
	// 改为：保留 sys_role，DROP sys_post → 岗位锚点查询失败（行 186/187）。
	_, err := PreviewUserScope(ctx, db, "D-1", nil, []string{"P"})
	assert.Error(t, err, "查询岗位锚点失败应返回 error")

	// 角色锚点查询失败：sys_role 存在但缺 role_name 列（MAX 查询只用 data_scope/status/deleted_at，可过）。
	db2 := newBareCoverDB(t, "src-role-anchor-err.db")
	require.NoError(t, db2.Exec(`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, ancestors TEXT, deleted_at DATETIME, dept_name TEXT)`).Error)
	require.NoError(t, db2.Exec(`CREATE TABLE sys_role (id TEXT PRIMARY KEY, data_scope INTEGER, status INTEGER, deleted_at DATETIME)`).Error)
	_, err = PreviewUserScope(ctx, db2, "D-1", []string{"R"}, nil)
	assert.Error(t, err, "查询角色锚点失败应返回 error")
}
