package system

import (
	"context"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestUserList_RecursiveDeptID
// 验证 RecursiveDeptID 参数把查询扩展为"该部门 + 所有子部门"的用户。
// 复用 sys_dept.ancestors 的 4-条件递归模式(参见 building_service.go)。
//
// 测试场景: 3 级部门 A(根) -> B(子) -> C(孙),每个部门各放 1 个用户。
//   - 用 RecursiveDeptID=A 查询 → 期望返回 3 个用户
//   - 用 RecursiveDeptID=B 查询 → 期望返回 2 个用户(B 自身 + C)
//   - 用 RecursiveDeptID=C 查询 → 期望返回 1 个用户
//   - 用 RecursiveDeptID=不存在的 ID → 期望返回 0 个用户
func TestUserList_RecursiveDeptID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	// 重建 sys_user / sys_dept 表。注意 sys_dept.ancestors 必须能存父链
	// (例如 C 的 ancestors = "A,B"),用于驱动递归子查询。
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
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (
			id TEXT PRIMARY KEY, role_name TEXT, status INTEGER,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)

	// ancestors 格式:无首尾逗号,根部门为空串。
	// A(根,ancestors='') → B(ancestors='A') → C(ancestors='A,B')
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('A','根部门','',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('B','一级子部门','A',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('C','二级子部门','A,B',0,?,?,NULL)`, now, now).Error)
	// 一个无关部门,验证不会误中
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('X','无关部门','',0,?,?,NULL)`, now, now).Error)

	// 每个部门各放 1 个用户
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('uA','alice','Alice','13800000000',0,'A','x',?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('uB','bob','Bob','13900000000',0,'B','x',?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('uC','carol','Carol','13700000000',0,'C','x',?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('uX','dave','Dave','13600000000',0,'X','x',?,?,NULL)`, now, now).Error)

	svc := &userService{db: db, pwdManager: nil}

	cases := []struct {
		name       string
		recursive  string
		wantNames  []string
	}{
		{
			name:      "递归根 A → 应返回 A/B/C 三个用户,排除 X",
			recursive: "A",
			wantNames: []string{"alice", "bob", "carol"},
		},
		{
			name:      "递归 B → 应返回 B/C 两个用户",
			recursive: "B",
			wantNames: []string{"bob", "carol"},
		},
		{
			name:      "递归 C → 应返回 C 一个用户",
			recursive: "C",
			wantNames: []string{"carol"},
		},
		{
			name:      "递归不存在的 ID → 应返回空",
			recursive: "nonexistent-uuid",
			wantNames: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rid := tc.recursive
			params := requests.UserListParams{
				BaseListRequest: base.BaseListRequest{
					Current:  1,
					PageSize: 100,
				},
				RecursiveDeptID: &rid,
			}
			result, err := svc.List(context.Background(), params)
			require.NoError(t, err, "List with RecursiveDeptID should not fail")
			require.NotNil(t, result)

			users, ok := result.List.([]models.User)
			require.True(t, ok, "result.List should be []models.User")
			require.Len(t, users, len(tc.wantNames), "user count mismatch")

			gotNames := make(map[string]bool, len(users))
			for _, u := range users {
				gotNames[u.Username] = true
			}
			for _, want := range tc.wantNames {
				require.True(t, gotNames[want], "expected user %q in result", want)
			}
		})
	}
}

// TestUserList_RecursiveDeptID_DoesNotConflictWithDeptID
// 验证 RecursiveDeptID 和 DeptID 同时提供时不会报错。
// 实际行为:两个条件是 AND 关系(交集),即 `dept_id = X AND dept_id IN recursive_set`。
// 预期结果:只有同时满足"dept_id = A" 且"dept_id 在 A 的递归集内"的 uA 被返回。
func TestUserList_RecursiveDeptID_DoesNotConflictWithDeptID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY, username TEXT, nickname TEXT, phone TEXT,
			status INTEGER NOT NULL, dept_id TEXT, password TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY, dept_name TEXT, ancestors TEXT,
			status INTEGER NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (
			id TEXT PRIMARY KEY, role_name TEXT, status INTEGER,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('A','根','',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('B','子','A',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('uA','alice','A','1',0,'A','x',?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('uB','bob','B','2',0,'B','x',?,?,NULL)`, now, now).Error)

	svc := &userService{db: db, pwdManager: nil}
	rid, did := "A", "A"
	params := requests.UserListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 100},
		DeptID:          &did,
		RecursiveDeptID: &rid,
	}
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err, "both DeptID and RecursiveDeptID should work without error")
	users := result.List.([]models.User)
	// AND 关系:uA 满足 dept_id='A' AND dept_id IN {A,B};uB 不满足 dept_id='A'
	require.Len(t, users, 1, "AND 关系:dept_id='A' ∧ recursive A 应只剩 uA")
	require.Equal(t, "alice", users[0].Username)
}
