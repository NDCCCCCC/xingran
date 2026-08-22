package system

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// asUserList 把 PageResult.List(interface{})还原成 []models.User。
func asUserList(p *PageResult) []models.User {
	users, _ := p.List.([]models.User)
	return users
}

// =====================================================================
// Phase 74-07: user_service Create/Update 事务路径 + fillUserRoles +
// List 排序/时间过滤补测(复用 user_service_test.go 的 DB/service 基建)。
// =====================================================================

// setupUserCRUDDB 在既有最小 schema 上按 GORM schema 动态补全缺失列
// (Create/Save 全字段写入需要;employee_no/created_by/version/avatar 等)。
func setupUserCRUDDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupUserServiceTestDB(t)

	userSchema, err := schema.Parse(&models.User{}, &sync.Map{}, db.NamingStrategy)
	require.NoError(t, err)

	var existing []string
	require.NoError(t, db.Raw(`SELECT name FROM pragma_table_info('sys_user')`).Scan(&existing).Error)
	have := map[string]bool{}
	for _, c := range existing {
		have[c] = true
	}
	timeCols := map[string]bool{"created_at": true, "updated_at": true, "deleted_at": true, "last_login_at": true, "last_pwd_update_time": true}
	for _, name := range userSchema.DBNames {
		if have[name] {
			continue
		}
		ddlType := " TEXT DEFAULT NULL"
		if timeCols[name] {
			ddlType = " DATETIME"
		}
		require.NoError(t, db.Exec("ALTER TABLE sys_user ADD COLUMN "+name+ddlType).Error, "补列失败: "+name)
	}
	// 关联表经 GORM Create 写入时带 created_at/updated_at(UserRole/UserPost 模型审计列)
	require.NoError(t, db.Exec(`ALTER TABLE sys_user_role ADD COLUMN created_at DATETIME`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE sys_user_role ADD COLUMN updated_at DATETIME`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE sys_user_post ADD COLUMN created_at DATETIME`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE sys_user_post ADD COLUMN updated_at DATETIME`).Error)
	return db
}

func TestUserServiceCRUD_CreateWithRelations(t *testing.T) {
	db := setupUserCRUDDB(t)
	svc := newTestUserService(t, db)
	ctx := context.Background()

	// 成功:角色 2 + 岗位 1
	nick := "小明"
	req := &requests.UserCreateRequest{
		Username: "newuser", Password: "pw123456", Nickname: &nick,
		RoleIds: []string{"r1", "r2"}, PostIds: []string{"p1"},
	}
	require.NoError(t, svc.Create(ctx, req))

	var user struct {
		ID       string
		Username string
		Password string
		Nickname string
	}
	require.NoError(t, db.Raw("SELECT id, username, password, nickname FROM sys_user WHERE username = 'newuser'").Scan(&user).Error)
	assert.Equal(t, "hashed:pw123456", user.Password, "密码应经 pwdManager 哈希")
	assert.Equal(t, "小明", user.Nickname)

	var roleRows int64
	require.NoError(t, db.Table("sys_user_role").Where("user_id = ?", user.ID).Count(&roleRows).Error)
	assert.Equal(t, int64(2), roleRows, "角色关联应批量插入")
	var postRows int64
	require.NoError(t, db.Table("sys_user_post").Where("user_id = ?", user.ID).Count(&postRows).Error)
	assert.Equal(t, int64(1), postRows)

	// 重复用户名
	err := svc.Create(ctx, &requests.UserCreateRequest{Username: "newuser", Password: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "newuser")

	// GetByID → fillUserRoles 有角色路径
	got, err := svc.GetByID(ctx, user.ID)
	require.NoError(t, err)
	// sys_role 表无 r1/r2 行 → JOIN 无命中,fillUserRoles 回空切片
	assert.NotNil(t, got.Roles)
}

func TestUserServiceCRUD_FillUserRolesWithRoles(t *testing.T) {
	db := setupUserCRUDDB(t)
	svc := newTestUserService(t, db)
	ctx := context.Background()

	// 种角色 + 用户 + 关联,覆盖 fillUserRoles JOIN 命中分支
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, status, created_at, updated_at, deleted_at) VALUES ('r1', '管理员', 0, ?, ?, NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, status, created_at, updated_at, deleted_at) VALUES ('r2', '运维', 0, ?, ?, NULL)`, now, now).Error)
	id := seedUserRow(t, db, "withroles", 0)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, 'r1'), (?, 'r2')`, id, id).Error)

	got, err := svc.GetByID(ctx, id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"管理员", "运维"}, got.Roles)
	assert.ElementsMatch(t, []string{"r1", "r2"}, got.RoleIds)
}

func TestUserServiceCRUD_UpdateFullPaths(t *testing.T) {
	db := setupUserCRUDDB(t)
	svc := newTestUserService(t, db)
	ctx := context.Background()

	// 种部门 + 用户 + 旧角色关联
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, ancestors, status, created_at, updated_at, deleted_at) VALUES ('d1', '研发部', '', 0, ?, ?, NULL)`, now, now).Error)
	id := seedUserRow(t, db, "updme", 0)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, 'old-role')`, id).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES (?, 'old-post')`, id).Error)

	nick := "新昵称"
	remark := "备注x"
	dept := "d1"
	err := svc.Update(ctx, &requests.UserUpdateRequest{
		ID: id, Nickname: &nick, DeptID: &dept, Status: 1,
		RoleIds: []string{"nr1"}, PostIds: []string{"np1"}, Remark: &remark,
	})
	require.NoError(t, err)

	var row struct {
		Nickname string
		DeptName *string
		Status   int
		Remark   string
	}
	require.NoError(t, db.Raw("SELECT nickname, dept_name, status, remark FROM sys_user WHERE id = ?", id).Scan(&row).Error)
	assert.Equal(t, "新昵称", row.Nickname)
	require.NotNil(t, row.DeptName, "DeptID 变更应同步部门名")
	assert.Equal(t, "研发部", *row.DeptName)
	assert.Equal(t, 1, row.Status)

	var roles []string
	require.NoError(t, db.Raw("SELECT role_id FROM sys_user_role WHERE user_id = ?", id).Scan(&roles).Error)
	assert.Equal(t, []string{"nr1"}, roles, "角色关联应先删后建")
	var posts []string
	require.NoError(t, db.Raw("SELECT post_id FROM sys_user_post WHERE user_id = ?", id).Scan(&posts).Error)
	assert.Equal(t, []string{"np1"}, posts)

	// DeptID 置 nil → DeptName 清空
	err = svc.Update(ctx, &requests.UserUpdateRequest{ID: id, Nickname: &nick})
	require.NoError(t, err)
	var deptName *string
	require.NoError(t, db.Raw("SELECT dept_name FROM sys_user WHERE id = ?", id).Scan(&deptName).Error)
	assert.Nil(t, deptName)

	// 清空角色/岗位(空数组 → 只删不建)
	require.NoError(t, svc.Update(ctx, &requests.UserUpdateRequest{ID: id, Nickname: &nick}))
	var cnt int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sys_user_role WHERE user_id = ?", id).Scan(&cnt).Error)
	assert.Zero(t, cnt)
}

func TestUserServiceCRUD_ListSortAndTimeFilter(t *testing.T) {
	db := setupUserCRUDDB(t)
	svc := newTestUserService(t, db)
	ctx := context.Background()

	// 3 个用户,不同 created_at / username / 状态
	seed := func(id, username string, status int, createdAt string) {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, password, status, created_at, updated_at, deleted_at) VALUES (?, ?, 'x', ?, ?, ?, NULL)`,
			id, username, status, createdAt, createdAt,
		).Error)
	}
	seed("u-a", "alpha", 0, "2024-01-01 00:00:00")
	seed("u-b", "bravo", 1, "2024-06-01 00:00:00")
	seed("u-c", "charlie", 0, "2025-01-01 00:00:00")

	base := requests.DefaultUserListParams()
	base.PageSize = 10

	// username 模糊(alpha/charlie 含 l,bravo 不含)
	p := base
	p.Username = strPtr("l")
	page, err := svc.List(ctx, p)
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// phone 过滤(空号全量)+ status
	st := 1
	p = base
	p.Status = &st
	page, err = svc.List(ctx, p)
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Equal(t, "bravo", asUserList(page)[0].Username)

	// QUIRK(D-12 记录不修):BeginTime/EndTime 过滤在 WHERE 中不带表限定,
	// 而主查询恒 LEFT JOIN sys_dept(两表均有 created_at)→ Find 阶段
	// "ambiguous column name: created_at"(sqlite 与生产 PG 同样报错)。
	// Count 在 Joins 前执行成功,Find 失败 → List 整体报错。
	p = base
	p.BeginTime = strPtr("2024-02-01 00:00:00")
	p.EndTime = strPtr("2024-12-31 23:59:59")
	_, err = svc.List(ctx, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "created_at")

	// 排序:username 升序 / 降序(白名单字段)
	asc := true
	p = base
	p.BaseListRequest.OrderByColumn = "username"
	p.BaseListRequest.IsAsc = &asc
	page, err = svc.List(ctx, p)
	require.NoError(t, err)
	assert.Equal(t, "alpha", asUserList(page)[0].Username)

	desc := false
	p.BaseListRequest.IsAsc = &desc
	page, err = svc.List(ctx, p)
	require.NoError(t, err)
	assert.Equal(t, "charlie", asUserList(page)[0].Username)

	// 非白名单排序字段 → 忽略排序不报错
	p.BaseListRequest.OrderByColumn = "password"
	page, err = svc.List(ctx, p)
	require.NoError(t, err)
	assert.Len(t, asUserList(page), 3)
}

func TestUserServiceCRUD_ToStringPtr(t *testing.T) {
	assert.Equal(t, "", toStringPtr(nil))
	assert.Equal(t, "v", toStringPtr(strPtr("v")))
}
