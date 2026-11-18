package system

// anchor_service_coverage_test.go —— 补齐 role_service.go / post_service.go /
// user_service.go 在机构层级数据权限 PR 中新增但未覆盖的分支。
//
// 仅新增测试，不修改任何生产代码；夹具在测试内自建，避免污染既有 fixtures。

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormio "gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

// newAnchorSvcDB 建角色/岗位锚点预览及配置校验所需的全部表。
func newAnchorSvcDB(t *testing.T) *gormio.DB {
	t.Helper()
	db, err := gormio.Open(sqlite.Open(filepath.Join(t.TempDir(), "anchor-svc.db")), &gormio.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	for _, ddl := range []string{
		`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT,
			ancestors TEXT DEFAULT '', org_level INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_user (id TEXT PRIMARY KEY, username TEXT, nickname TEXT, dept_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_role (id TEXT PRIMARY KEY, role_name TEXT, role_key TEXT, role_sort INTEGER DEFAULT 0,
			data_scope INTEGER DEFAULT 1, anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '',
			menu_check_strictly INTEGER DEFAULT 1, dept_check_strictly INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0, remark TEXT, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME,
			created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0)`,
		`CREATE TABLE sys_post (id TEXT PRIMARY KEY, post_code TEXT, post_name TEXT, post_sort INTEGER DEFAULT 0,
			anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '', status INTEGER DEFAULT 0,
			remark TEXT, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME,
			created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0)`,
		`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`,
		`CREATE TABLE sys_user_post (user_id TEXT, post_id TEXT)`,
		`CREATE TABLE sys_role_menu (role_id TEXT, menu_id TEXT)`,
		`CREATE TABLE sys_role_dept (role_id TEXT, dept_id TEXT)`,
		`CREATE TABLE sys_menu (id TEXT PRIMARY KEY, menu_name TEXT, perms TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, ancestors, org_level, is_external_org, status) VALUES
		('D-hb','湖北分公司','',1,1,0),
		('D-wh','武汉中支','D-hb',2,1,0),
		('D-zh','综合管理部','D-hb,D-wh',0,0,0),
		('D-qk','硚口支公司','D-hb,D-wh',3,1,0)`).Error)
	return db
}

func TestValidateAnchorForWriteAndSanitize(t *testing.T) {
	db := newAnchorSvcDB(t)
	ctx := context.Background()

	// 合法配置：透传 nil。
	require.NoError(t, validateAnchorForWrite(ctx, db, 0, "ignored"))
	require.NoError(t, validateAnchorForWrite(ctx, db, 2, "D-wh"))
	require.NoError(t, validateAnchorForWrite(ctx, db, 3, ""))
	require.NoError(t, validateAnchorForWrite(ctx, db, 5, "2"))

	// 非法配置：被转成 400 BadRequest（apperrors.BadRequest）。
	for _, tc := range []struct {
		mode  int
		param string
	}{
		{1, "2"}, {4, ""}, {2, ""}, {2, "D-nope"}, {3, "D-wh"}, {5, "abc"}, {5, "4"},
	} {
		err := validateAnchorForWrite(ctx, db, tc.mode, tc.param)
		require.Error(t, err, "mode=%d param=%q 应被拒绝", tc.mode, tc.param)
		assert.True(t, strings.Contains(err.Error(), "非法") ||
			strings.Contains(err.Error(), "必须") ||
			strings.Contains(err.Error(), "不存在") ||
			strings.Contains(err.Error(), "不需要参数") ||
			strings.Contains(err.Error(), "未标记") ||
			strings.Contains(err.Error(), "未知"),
			"错误应是 BadRequest 中文提示: %v", err)
	}

	// sanitizeAnchorParam：0/3 恒空，其它 TrimSpace。
	assert.Empty(t, sanitizeAnchorParam(0, "  x  "))
	assert.Empty(t, sanitizeAnchorParam(3, "  x  "))
	assert.Equal(t, "D-wh", sanitizeAnchorParam(2, "  D-wh  "))
	assert.Equal(t, "2", sanitizeAnchorParam(5, " 2 "))
	assert.Equal(t, "", sanitizeAnchorParam(2, "   "))
}

func TestRoleServiceCreateUpdateRejectBadAnchor(t *testing.T) {
	db := newAnchorSvcDB(t)
	svc := NewRoleService(db)
	ctx := context.Background()

	// Create 入参非法锚点 → ValidateAnchorForWrite 报错，行 111/112。
	err := svc.Create(ctx, &requests.RoleCreateRequest{
		RoleName: "bad-anchor", RoleKey: "bad-anchor", AnchorMode: 1, AnchorParam: "2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非法")

	// 合法锚点 → 成功，且落库参数已 sanitize。
	require.NoError(t, svc.Create(ctx, &requests.RoleCreateRequest{
		RoleName: "level-role", RoleKey: "level-role", AnchorMode: 5, AnchorParam: " 2 ",
	}))
	var role models.Role
	require.NoError(t, db.Where("role_key = ?", "level-role").First(&role).Error)
	assert.Equal(t, models.AnchorModeLevel, role.AnchorMode)
	assert.Equal(t, "2", role.AnchorParam)

	// Update 入参非法 → 行 170/171。
	err = svc.Update(ctx, &requests.RoleUpdateRequest{
		ID: role.ID, RoleName: "bad", RoleKey: "bad", AnchorMode: 4,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非法")
}

func TestRolePreviewAnchorBranches(t *testing.T) {
	db := newAnchorSvcDB(t)
	svc := NewRoleService(db)
	ctx := context.Background()

	// 角色不存在 → RoleNotFound（行 608/609）。
	_, err := svc.PreviewAnchor(ctx, "R-missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "角色不存在")

	// 存在角色 + 两个任职人命中同一锚点 → warnings 分支（行 686-700）。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R-mid','中支管理员',5,'2',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES
		('U-a','a','甲','D-zh',0), ('U-b','b','乙','D-zh',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-a','R-mid'),('U-b','R-mid')`).Error)

	res, err := svc.PreviewAnchor(ctx, "R-mid")
	require.NoError(t, err)
	assert.Equal(t, 2, res.TotalCount)
	assert.Equal(t, 2, res.ResolvedCount)
	assert.Len(t, res.Unresolved, 0)
	require.Len(t, res.Warnings, 1, "两名任职人命中同一锚点应告警一次")
	assert.Contains(t, res.Warnings[0], "D-wh")
	assert.Contains(t, res.Warnings[0], "2 名任职人")

	// mode=0：未命中项的 reason 被替换成明确文案（行 659-665）。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R-none','普通角色',0,'legacy-param',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-a','R-none')`).Error)
	res2, err := svc.PreviewAnchor(ctx, "R-none")
	require.NoError(t, err)
	require.Len(t, res2.Items, 1)
	assert.False(t, res2.Items[0].Resolved)
	assert.Contains(t, res2.Items[0].Reason, "未配置机构锚点")
	assert.Len(t, res2.Unresolved, 1)

	// mode=2（显式部门）不告警。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R-dept','指定部门角色',2,'D-wh',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-a','R-dept'),('U-b','R-dept')`).Error)
	res3, err := svc.PreviewAnchor(ctx, "R-dept")
	require.NoError(t, err)
	assert.Empty(t, res3.Warnings, "模式2显式同一部门属预期，不告警")
}

func TestListAndNormalizeHelpers(t *testing.T) {
	db := newAnchorSvcDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES
		('U-b','R'),('U-a','R'),('U-a','R'),('','R')`).Error)
	ids, err := listRoleAssigneeIDs(ctx, db, "R")
	require.NoError(t, err)
	assert.Equal(t, []string{"U-a", "U-b"}, ids, "去重+去空+排序")

	require.NoError(t, db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES
		('U-c','P'),('U-c','P'),('','P')`).Error)
	pids, err := listPostAssigneeIDs(ctx, db, "P")
	require.NoError(t, err)
	assert.Equal(t, []string{"U-c"}, pids)

	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, status) VALUES
		('U-a','a','甲',0), ('U-b','b','',0)`).Error)
	usernames, nicknames, err := queryAssigneeNames(ctx, db, ids)
	require.NoError(t, err)
	assert.Equal(t, "a", usernames["U-a"])
	assert.Equal(t, "甲", nicknames["U-a"])
	assert.Empty(t, nicknames["U-b"], "NULL nickname 归空串")

	emptyU, emptyN, err := queryAssigneeNames(ctx, db, nil)
	require.NoError(t, err)
	assert.Empty(t, emptyU)
	assert.Empty(t, emptyN)

	// 缺 sys_user 表 → 错误传播。
	bareDB, err := gormio.Open(sqlite.Open(filepath.Join(t.TempDir(), "bare-svc.db")), &gormio.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := bareDB.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	_, _, err = queryAssigneeNames(ctx, bareDB, []string{"U-a"})
	assert.Error(t, err)
	_, err = listRoleAssigneeIDs(ctx, bareDB, "R")
	assert.Error(t, err)
	_, err = listPostAssigneeIDs(ctx, bareDB, "P")
	assert.Error(t, err)

	assert.Empty(t, normalizeUserIDs(nil))
}

func TestPostServiceCreateUpdateRejectBadAnchorAndPreview(t *testing.T) {
	db := newAnchorSvcDB(t)
	svc := NewPostService(db)
	ctx := context.Background()

	err := svc.Create(ctx, &requests.PostCreateRequest{PostCode: "bad", PostName: "bad", AnchorMode: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非法")

	require.NoError(t, svc.Create(ctx, &requests.PostCreateRequest{
		PostCode: "mid", PostName: "中支管理员", AnchorMode: 5, AnchorParam: " 2 ",
	}))
	var post models.Post
	require.NoError(t, db.Where("post_code = ?", "mid").First(&post).Error)
	assert.Equal(t, "2", post.AnchorParam)

	err = svc.Update(ctx, &requests.PostUpdateRequest{
		ID: post.ID, PostName: "bad", AnchorMode: 4,
	})
	require.Error(t, err)

	// PreviewAnchor: 不存在 → 普通 error；存在 + 任职人 → 成功。
	_, err = svc.PreviewAnchor(ctx, "P-missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "岗位不存在")

	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, dept_id, status)
		VALUES ('U-p','p','戊','D-zh',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES ('U-p',?)`, post.ID).Error)
	res, err := svc.PreviewAnchor(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, res.ResolvedCount)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "D-wh", res.Items[0].AnchorDeptID)

	// 缺表（DROP sys_user）→ 任职人查询报错传播。
	require.NoError(t, db.Exec(`DROP TABLE sys_user`).Error)
	_, err = svc.PreviewAnchor(ctx, post.ID)
	assert.Error(t, err)
}

func TestUserServiceFillPostsAndDataScopeContext(t *testing.T) {
	db := newAnchorSvcDB(t)
	svc := newTestUserService(t, db)
	ctx := context.Background()

	// 无用户 → 直接返回。
	svc.(*userService).fillUserPosts(ctx, nil)

	// 用户 ID 均空 → 早退（行 617/618）。
	users := []models.User{{}, {}}
	svc.(*userService).fillUserPosts(ctx, users)
	assert.NotNil(t, users[0].Posts)
	assert.Empty(t, users[0].Posts)

	// 正常回填：岗位按 post_sort 升序，锚点字段保留。
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_code, post_name, post_sort, anchor_mode, anchor_param, status)
		VALUES ('P-1','p1','岗位一',2,5,'2',0), ('P-2','p2','岗位二',1,0,'',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, status) VALUES
		('U-1','u1',0), ('U-2','u2',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES
		('U-1','P-1'),('U-1','P-2'),('U-2','P-1')`).Error)
	users = []models.User{{BaseModel: models.BaseModel{ID: "U-1"}}, {BaseModel: models.BaseModel{ID: "U-2"}}}
	svc.(*userService).fillUserPosts(ctx, users)
	require.Len(t, users[0].Posts, 2)
	assert.Equal(t, "岗位二", users[0].Posts[0].PostName, "按 post_sort 升序")
	assert.Equal(t, "岗位一", users[0].Posts[1].PostName)
	assert.Equal(t, models.AnchorModeLevel, users[0].Posts[1].AnchorMode)
	require.Len(t, users[1].Posts, 1)

	// 查询异常（缺 sys_user_post）→ 静默降级，Posts 为空切片。
	require.NoError(t, db.Exec(`DROP TABLE sys_user_post`).Error)
	users2 := []models.User{{BaseModel: models.BaseModel{ID: "U-1"}}}
	svc.(*userService).fillUserPosts(ctx, users2)
	assert.NotNil(t, users2[0].Posts)
	assert.Empty(t, users2[0].Posts)

	// isMissingTableErr 分支（首行 nil + 三种缺表文案 + 非缺表）。
	assert.False(t, isMissingTableErr(nil))
	assert.True(t, isMissingTableErr(errors.New("no such table: x")))
	assert.True(t, isMissingTableErr(errors.New("relation does not exist")))
	assert.True(t, isMissingTableErr(errors.New("undefined table")))
	assert.False(t, isMissingTableErr(errors.New("boom")))

	// applyUserDataScope：ctx 注入的 Merge 结果优先（行 732-748）。
	allCtx := datascope.WithContext(context.Background(), datascope.AllScope("u"))
	qAll := svc.(*userService).applyUserDataScope(db.Model(&models.User{}), allCtx, "u", models.DataScopeSelf)
	var allUsers []models.User
	sqlAll := qAll.Session(&gormio.Session{DryRun: true}).Find(&allUsers).Statement.SQL.String()
	assert.NotContains(t, sqlAll, "1=0", "ctx 注入 All 应不过滤（行 733/734）")
	assert.NotContains(t, sqlAll, "dept_id", "ctx 注入 All 不应按部门过滤")

	for _, tc := range []struct {
		name  string
		scope datascope.ScopeResult
	}{
		{"fail-closed", datascope.FailClosedScope("u")},
		{"self-without-uid", datascope.ScopeResult{ResourceHook: datascope.HookSelfUser}},
		{"self-with-uid", datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "u"}},
		{"empty-dept", datascope.ScopeResult{}},
		{"dept-ids", datascope.ScopeResult{DeptIDs: []string{"D-1"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := svc.(*userService).applyUserDataScope(db.Model(&models.User{}),
				datascope.WithContext(context.Background(), tc.scope), "ignored", models.DataScopeAll)
			require.NotNil(t, q)
			var users []models.User
			sql := q.Session(&gormio.Session{DryRun: true}).Find(&users).Statement.SQL.String()
			if tc.name == "fail-closed" || tc.name == "self-without-uid" || tc.name == "empty-dept" {
				assert.Contains(t, sql, "1=0", "应 fail-closed")
			} else if tc.name == "self-with-uid" {
				assert.Contains(t, sql, "sys_user.id")
			} else {
				assert.Contains(t, sql, "dept_id")
			}
		})
	}
}

func TestPreviewAnchor_AssigneeQueryErrors(t *testing.T) {
	// role：角色存在但缺 sys_user_role → listRoleAssigneeIDs 报错（行 615-617）。
	dbR := newAnchorSvcDB(t)
	require.NoError(t, dbR.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R','r',0,'',0)`).Error)
	require.NoError(t, dbR.Exec(`DROP TABLE sys_user_role`).Error)
	_, err := NewRoleService(dbR).PreviewAnchor(context.Background(), "R")
	assert.Error(t, err)

	// post：岗位存在但缺 sys_user_post → listPostAssigneeIDs 报错（行 281-283）。
	dbP := newAnchorSvcDB(t)
	require.NoError(t, dbP.Exec(`INSERT INTO sys_post (id, post_code, post_name, anchor_mode, anchor_param, status)
		VALUES ('P','p','p',0,'',0)`).Error)
	require.NoError(t, dbP.Exec(`DROP TABLE sys_user_post`).Error)
	_, err = NewPostService(dbP).PreviewAnchor(context.Background(), "P")
	assert.Error(t, err)
}
