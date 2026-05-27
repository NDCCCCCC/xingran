package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleUserLoginAD_UserNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建映射表
	db.Exec(`CREATE TABLE sys_dept_ou_mapping (
		id TEXT PRIMARY KEY,
		dept_id TEXT NOT NULL,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL,
		ou_name TEXT NOT NULL,
		parent_ou_dn TEXT,
		sync_enabled INTEGER DEFAULT 1,
		sync_status TEXT DEFAULT 'pending',
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		UNIQUE(dept_id, ad_config_id)
	)`)

	// 创建用户表
	db.Exec(`CREATE TABLE sys_user (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		dept_id TEXT,
		ad_dn TEXT,
		ad_ou_dn TEXT,
		ad_synced_at DATETIME
	)`)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 测试用户不存在场景（应返回nil，不报错）
	err = service.HandleUserLoginAD(ctx, "nonexistent", "CN=user,OU=TestDept,DC=company,DC=com", "OU=TestDept,DC=company,DC=com")
	assert.NoError(t, err) // 不应该报错，只记录日志
}

func TestHandleUserLoginAD_MappingNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建映射表
	db.Exec(`CREATE TABLE sys_dept_ou_mapping (
		id TEXT PRIMARY KEY,
		dept_id TEXT NOT NULL,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL,
		ou_name TEXT NOT NULL,
		parent_ou_dn TEXT,
		sync_enabled INTEGER DEFAULT 1,
		sync_status TEXT DEFAULT 'pending',
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		UNIQUE(dept_id, ad_config_id)
	)`)

	// 创建用户表
	db.Exec(`CREATE TABLE sys_user (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		dept_id TEXT,
		ad_dn TEXT,
		ad_ou_dn TEXT,
		ad_synced_at DATETIME
	)`)

	// 插入测试用户
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('user-1', 'testuser')`)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 测试映射不存在场景（应返回nil，不报错）
	err = service.HandleUserLoginAD(ctx, "testuser", "CN=testuser,OU=NonExistent,DC=company,DC=com", "OU=NonExistent,DC=company,DC=com")
	assert.NoError(t, err) // 不应该报错，只记录警告日志
}

func TestHandleUserLoginAD_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建映射表
	db.Exec(`CREATE TABLE sys_dept_ou_mapping (
		id TEXT PRIMARY KEY,
		dept_id TEXT NOT NULL,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL,
		ou_name TEXT NOT NULL,
		parent_ou_dn TEXT,
		sync_enabled INTEGER DEFAULT 1,
		sync_status TEXT DEFAULT 'pending',
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		UNIQUE(dept_id, ad_config_id)
	)`)

	// 创建用户表
	db.Exec(`CREATE TABLE sys_user (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		dept_id TEXT,
		ad_dn TEXT,
		ad_ou_dn TEXT,
		ad_synced_at DATETIME
	)`)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 准备映射数据
	db.Exec(`INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, sync_enabled, sync_status)
		VALUES ('map-1', 'dept-1', 'config-1', 'OU=TestDept,DC=company,DC=com', 'TestDept', 1, 'synced')`)

	// 插入测试用户
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('user-1', 'testuser')`)

	// 测试成功场景
	err = service.HandleUserLoginAD(ctx, "testuser", "CN=testuser,OU=TestDept,DC=company,DC=com", "OU=TestDept,DC=company,DC=com")
	assert.NoError(t, err)

	// 验证用户信息已更新
	var deptID, adUserDN, adOUDN string
	row := db.Raw(`SELECT dept_id, ad_dn, ad_ou_dn FROM sys_user WHERE username = ?`, "testuser").Row()
	row.Scan(&deptID, &adUserDN, &adOUDN)

	assert.Equal(t, "dept-1", deptID)
	assert.Equal(t, "CN=testuser,OU=TestDept,DC=company,DC=com", adUserDN)
	assert.Equal(t, "OU=TestDept,DC=company,DC=com", adOUDN)
}

func TestGetUserDeptByADOU(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建用户表
	db.Exec(`CREATE TABLE sys_user (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		dept_id TEXT
	)`)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 测试用户不存在场景
	dept, err := service.GetUserDeptByADOU(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, dept)
}

func TestUpdateUserDeptAndADInfo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建用户表
	db.Exec(`CREATE TABLE sys_user (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		dept_id TEXT,
		ad_dn TEXT,
		ad_ou_dn TEXT,
		ad_synced_at DATETIME
	)`)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 插入测试用户
	db.Exec(`INSERT INTO sys_user (id, username) VALUES ('user-1', 'testuser')`)

	// 测试更新用户信息
	err = service.updateUserDeptAndADInfo(ctx, "user-1", "dept-1", "CN=testuser,OU=TestDept,DC=company,DC=com", "OU=TestDept,DC=company,DC=com")
	assert.NoError(t, err)

	// 验证更新
	var deptID, adUserDN, adOUDN string
	row := db.Raw(`SELECT dept_id, ad_dn, ad_ou_dn FROM sys_user WHERE id = ?`, "user-1").Row()
	row.Scan(&deptID, &adUserDN, &adOUDN)

	assert.Equal(t, "dept-1", deptID)
	assert.Equal(t, "CN=testuser,OU=TestDept,DC=company,DC=com", adUserDN)
	assert.Equal(t, "OU=TestDept,DC=company,DC=com", adOUDN)
}
