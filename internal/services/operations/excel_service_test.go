package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupUserRoleTestDB 构造内存 sqlite，建角色分配依赖的三张表。
// 自建简化表结构（仅测试所需列 + deleted_at 满足 GORM 软删除过滤），不依赖完整 model。
func setupUserRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	db.Exec(`CREATE TABLE sys_role (id TEXT, role_key TEXT, deleted_at DATETIME)`)
	db.Exec(`CREATE TABLE sys_user (id TEXT, username TEXT, deleted_at DATETIME)`)
	db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`)
	return db
}

func TestAssignDefaultRolesToNewUsers_BasicAssign(t *testing.T) {
	db := setupUserRoleTestDB(t)
	db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('role-common', 'common')`)
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u1', 'alice')`)
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u2', 'bob')`)

	svc := &ExcelService{db: db}
	err := svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice", "bob"})
	assert.NoError(t, err)

	var count int64
	db.Table("sys_user_role").Where("role_id = ?", "role-common").Count(&count)
	assert.Equal(t, int64(2), count, "两个新用户都应分配 common 角色")
}

func TestAssignDefaultRolesToNewUsers_SkipExistingRole(t *testing.T) {
	db := setupUserRoleTestDB(t)
	db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('role-common', 'common')`)
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u1', 'alice')`)
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u2', 'bob')`)
	// bob 已有别的角色 → 不应被覆盖
	db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('u2', 'role-other')`)

	svc := &ExcelService{db: db}
	err := svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice", "bob"})
	assert.NoError(t, err)

	var aliceRoles, bobRoles []string
	db.Table("sys_user_role").Where("user_id = ?", "u1").Pluck("role_id", &aliceRoles)
	db.Table("sys_user_role").Where("user_id = ?", "u2").Pluck("role_id", &bobRoles)
	assert.Equal(t, []string{"role-common"}, aliceRoles, "无角色的新用户应分配 common")
	assert.Equal(t, []string{"role-other"}, bobRoles, "已有角色的用户不应被覆盖")
}

func TestAssignDefaultRolesToNewUsers_NoCommonRole(t *testing.T) {
	db := setupUserRoleTestDB(t)
	// 不插 common 角色
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u1', 'alice')`)

	svc := &ExcelService{db: db}
	err := svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice"})
	assert.NoError(t, err, "无 common 角色应跳过，不报错")

	var count int64
	db.Table("sys_user_role").Count(&count)
	assert.Equal(t, int64(0), count, "无 common 角色不应写入任何关联")
}

func TestAssignDefaultRolesToNewUsers_Idempotent(t *testing.T) {
	db := setupUserRoleTestDB(t)
	db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('role-common', 'common')`)
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u1', 'alice')`)

	svc := &ExcelService{db: db}
	// 重复调用两次（模拟重复导入同一用户）
	assert.NoError(t, svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice"}))
	assert.NoError(t, svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice"}))

	var count int64
	db.Table("sys_user_role").Where("user_id = ?", "u1").Count(&count)
	assert.Equal(t, int64(1), count, "重复调用不应产生重复记录")
}

func TestAssignDefaultRolesToNewUsers_EmptyInput(t *testing.T) {
	db := setupUserRoleTestDB(t)
	svc := &ExcelService{db: db}
	assert.NoError(t, svc.assignDefaultRolesToNewUsers(context.Background(), db, nil))
	assert.NoError(t, svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{}))
}
