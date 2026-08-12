package system

import (
	"context"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestUserList_StatusNotAmbiguous_AfterJoinSysDept
// 回归测试:验证 List() 在 status 过滤条件 + LEFT JOIN sys_dept 同时出现时,
// 生成的 SQL 不会产生 "column reference 'status' is ambiguous" (SQLSTATE 42702)。
//
// 原因:sys_user 和 sys_dept 都有 status 列,未限定的 status 引用在 PostgreSQL 上
// 会因歧义而失败。修复后 Where 条件使用 sys_user.status 限定符。
func TestUserList_StatusNotAmbiguous_AfterJoinSysDept(t *testing.T) {
	// 使用 SQLite 内存数据库,模式与 PostgreSQL 接近(对 column ambiguity 也会触发)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	// 重建 sys_user 和 sys_dept 表,确保两个表都有 status 列(模拟 PostgreSQL 行为)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			nickname TEXT,
			phone TEXT,
			status INTEGER NOT NULL,
			dept_id TEXT,
			password TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			ancestors TEXT,
			status INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (id TEXT PRIMARY KEY, role_name TEXT, status INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)
	`).Error)

	// 插入测试数据
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('d1','研发部','',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('d2','已停用部门','',1,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u1','alice','Alice','13800000000',0,'d1','x',?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u2','bob','Bob','13900000000',1,'d1','x',?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u3','carol','Carol','13700000000',0,'d2','x',?,?,NULL)`, now, now).Error)

	// 构造 userService(用最小依赖,只关心 db)
	svc := &userService{db: db, pwdManager: nil}

	status0 := 0
	params := requests.UserListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  1,
			PageSize: 10,
		},
		Status: &status0,
	}

	// 关键断言:这次 List 调用不再因 "column reference 'status' is ambiguous" 而失败
	// 修复前: PostgreSQL 报 SQLSTATE 42702;SQLite 也会因为解析时报错。
	// 修复后: 只匹配 sys_user.status=0 的用户 (u1, u3)。
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err, "List should not fail with ambiguous column error")
	require.NotNil(t, result)

	users, ok := result.List.([]models.User)
	require.True(t, ok, "result.List should be []models.User")
	require.Len(t, users, 2, "expected 2 users with status=0 (u1, u3)")

	// 验证返回的是 sys_user.status=0 的行,不是 sys_dept.status
	gotUsernames := map[string]bool{}
	for _, u := range users {
		gotUsernames[u.Username] = true
	}
	require.True(t, gotUsernames["alice"], "alice (sys_user.status=0) must be returned")
	require.True(t, gotUsernames["carol"], "carol (sys_user.status=0, sys_dept.status=1) must be returned")
	require.False(t, gotUsernames["bob"], "bob (sys_user.status=1) must NOT be returned")
}
