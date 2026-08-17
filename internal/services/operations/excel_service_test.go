package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/glebarez/sqlite"
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
	db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('role-user', 'user')`)
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u1', 'alice')`)
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u2', 'bob')`)

	svc := &ExcelService{db: db}
	err := svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice", "bob"})
	assert.NoError(t, err)

	var count int64
	db.Table("sys_user_role").Where("role_id = ?", "role-user").Count(&count)
	assert.Equal(t, int64(2), count, "两个新用户都应分配 user 角色")
}

func TestAssignDefaultRolesToNewUsers_SkipExistingRole(t *testing.T) {
	db := setupUserRoleTestDB(t)
	db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('role-user', 'user')`)
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
	assert.Equal(t, []string{"role-user"}, aliceRoles, "无角色的新用户应分配 user")
	assert.Equal(t, []string{"role-other"}, bobRoles, "已有角色的用户不应被覆盖")
}

func TestAssignDefaultRolesToNewUsers_NoUserRole(t *testing.T) {
	db := setupUserRoleTestDB(t)
	// 不插 user 角色
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u1', 'alice')`)

	svc := &ExcelService{db: db}
	err := svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice"})
	assert.NoError(t, err, "无 user 角色应跳过，不报错")

	var count int64
	db.Table("sys_user_role").Count(&count)
	assert.Equal(t, int64(0), count, "无 user 角色不应写入任何关联")
}

func TestAssignDefaultRolesToNewUsers_Idempotent(t *testing.T) {
	db := setupUserRoleTestDB(t)
	db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('role-user', 'user')`)
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

// TestIsEmptyValue 验证 isEmptyValue helper（first-non-empty-wins 基础）。
func TestIsEmptyValue(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"whitespace string", "   ", true},
		{"non-empty string", "x", false},
		{"nil pointer", (*string)(nil), true},
		{"empty pointer", ptr(""), true},
		{"non-empty pointer", ptr("x"), false},
		{"int zero", int(0), false},  // 0 不是"空"
		{"int non-zero", int(1), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isEmptyValue(c.in), "input=%v", c.in)
		})
	}
}

func ptr(s string) *string { return &s }

// TestPrepareRecordsForUpsert_FirstNonEmptyWins 验证同 DBField 多列时
// 保留先写非空值（解决 P0-4: machineIP + domainIP 同写 machine_ip 时真实 IP 被覆盖）。
func TestPrepareRecordsForUpsert_FirstNonEmptyWins(t *testing.T) {
	// 简化 config: 模拟 asset config 的 machineIP (列 32) + machineBS (列 33) + domainIP (列 34)
	// machineIP 与 domainIP 同 DBField, 期望 first-non-empty-wins。
	cfg := ExcelConfig{
		TableName: "ops_asset",
		Columns: []ExcelColumn{
			{Field: "machineIP", Header: "IP地址", DBField: "machine_ip"},
			{Field: "machineBS", Header: "加域标识", DBField: "machine_bs"},
			{Field: "domainIP", Header: "加域ip地址", DBField: "machine_ip"},
		},
	}
	svc := &ExcelService{}

	t.Run("A: machineIP 非空 + domainIP 空 → 取 machineIP", func(t *testing.T) {
		records := []map[string]any{{
			"machineIP": "10.62.101.143",
			"machineBS": "PR.intra.cpic.com.cn",
		}}
		got := svc.prepareRecordsForUpsert(records, cfg)
		assert.Equal(t, "10.62.101.143", got[0]["machine_ip"], "应保留先写非空值")
		assert.Equal(t, "PR.intra.cpic.com.cn", got[0]["machine_bs"])
	})

	t.Run("B: machineIP 空 + domainIP 非空 → 取 domainIP", func(t *testing.T) {
		records := []map[string]any{{
			"machineIP": "",
			"machineBS": "PR.intra.cpic.com.cn",
			"domainIP":  "PR.intra.cpic.com.cn",
		}}
		got := svc.prepareRecordsForUpsert(records, cfg)
		assert.Equal(t, "PR.intra.cpic.com.cn", got[0]["machine_ip"])
		assert.Equal(t, "PR.intra.cpic.com.cn", got[0]["machine_bs"])
	})

	t.Run("C: 两者都非空 → 取 machineIP（先写优先）", func(t *testing.T) {
		records := []map[string]any{{
			"machineIP": "10.62.101.143",
			"machineBS": "PR.intra.cpic.com.cn",
			"domainIP":  "PR.intra.cpic.com.cn",
		}}
		got := svc.prepareRecordsForUpsert(records, cfg)
		assert.Equal(t, "10.62.101.143", got[0]["machine_ip"],
			"first-non-empty-wins: 先写 machineIP=10.62.101.143 应保留, 后续 domainIP 跳过")
	})

	t.Run("D: 两者都空 → machine_ip 空", func(t *testing.T) {
		records := []map[string]any{{
			"machineBS": "PR.intra.cpic.com.cn",
		}}
		got := svc.prepareRecordsForUpsert(records, cfg)
		_, exists := got[0]["machine_ip"]
		assert.False(t, exists, "两个都空时, preparedRecord 不应有 machine_ip 键")
		assert.Equal(t, "PR.intra.cpic.com.cn", got[0]["machine_bs"])
	})

	t.Run("E: machineIP 全空白 → 当作空处理, 取 domainIP", func(t *testing.T) {
		records := []map[string]any{{
			"machineIP": "   ",
			"domainIP":  "PR.intra.cpic.com.cn",
		}}
		got := svc.prepareRecordsForUpsert(records, cfg)
		assert.Equal(t, "PR.intra.cpic.com.cn", got[0]["machine_ip"])
	})
}

// TestPrepareRecordsForUpsert_NoSameDBField_NoRegression 验证没有同 DBField
// 多列的场景下行为不变（向后兼容, 不影响其他 entityType）。
func TestPrepareRecordsForUpsert_NoSameDBField_NoRegression(t *testing.T) {
	cfg := ExcelConfig{
		TableName: "sys_workstation",
		Columns: []ExcelColumn{
			{Field: "name", Header: "工位名称", DBField: "workstation_name"},
			{Field: "buildingName", Header: "所属楼宇", DBField: "building_id"},
			{Field: "floorName", Header: "所属楼层", DBField: "floor_id"},
		},
	}
	svc := &ExcelService{}
	records := []map[string]any{{
		"name":         "工位A",
		"buildingName": "总部大楼",
		"floorName":    "3楼",
	}}
	got := svc.prepareRecordsForUpsert(records, cfg)
	assert.Equal(t, "工位A", got[0]["workstation_name"])
	assert.Equal(t, "总部大楼", got[0]["building_id"])
	assert.Equal(t, "3楼", got[0]["floor_id"])
}
