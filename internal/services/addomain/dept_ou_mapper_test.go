package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindDeptByOUDN(t *testing.T) {
	// 使用SQLite内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 直接创建表结构（避免AutoMigrate的外键问题）
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
	db.Exec(`CREATE INDEX idx_dept_ou_mapping_dn ON sys_dept_ou_mapping(ou_dn)`)

	mapper := NewDeptOUmapper(db)

	// 准备测试数据
	ctx := context.Background()
	mapping := &models.DeptOUMapping{
		ID:          "test-id-1",
		DeptID:      "dept-1",
		ADConfigID:  "config-1",
		OUDN:        "OU=TestDept,DC=company,DC=com",
		OUName:      "TestDept",
		SyncEnabled: true,
		SyncStatus:  "synced",
	}
	err = mapper.UpsertMapping(ctx, mapping)
	assert.NoError(t, err)

	// 测试查找
	deptID, err := mapper.FindDeptByOUDN(ctx, "OU=TestDept,DC=company,DC=com")
	assert.NoError(t, err)
	assert.Equal(t, "dept-1", deptID)

	// 测试未找到
	_, err = mapper.FindDeptByOUDN(ctx, "OU=NonExistent,DC=company,DC=com")
	assert.Error(t, err)
}

func TestFindOUDNByDeptID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建表
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

	mapper := NewDeptOUmapper(db)

	ctx := context.Background()
	mapping := &models.DeptOUMapping{
		ID:          "test-id-2",
		DeptID:      "dept-1",
		ADConfigID:  "config-1",
		OUDN:        "OU=TestDept,DC=company,DC=com",
		OUName:      "TestDept",
		SyncEnabled: true,
		SyncStatus:  "synced",
	}
	err = mapper.UpsertMapping(ctx, mapping)
	assert.NoError(t, err)

	// 测试查找
	ouDN, err := mapper.FindOUDNByDeptID(ctx, "dept-1")
	assert.NoError(t, err)
	assert.Equal(t, "OU=TestDept,DC=company,DC=com", ouDN)

	// 测试未找到
	_, err = mapper.FindOUDNByDeptID(ctx, "non-existent")
	assert.Error(t, err)
}

func TestUpsertMapping(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建表
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

	mapper := NewDeptOUmapper(db)

	ctx := context.Background()

	// 测试插入
	mapping := &models.DeptOUMapping{
		ID:          "test-id-3",
		DeptID:      "dept-1",
		ADConfigID:  "config-1",
		OUDN:        "OU=TestDept,DC=company,DC=com",
		OUName:      "TestDept",
		SyncEnabled: true,
		SyncStatus:  "synced",
	}
	err = mapper.UpsertMapping(ctx, mapping)
	assert.NoError(t, err)

	// 测试更新（幂等性）
	mapping.OUDN = "OU=UpdatedDept,DC=company,DC=com"
	mapping.OUName = "UpdatedDept"
	err = mapper.UpsertMapping(ctx, mapping)
	assert.NoError(t, err)

	// 验证更新
	result, err := mapper.GetMappingByDept(ctx, "dept-1")
	assert.NoError(t, err)
	assert.Equal(t, "OU=UpdatedDept,DC=company,DC=com", result.OUDN)
	assert.Equal(t, "UpdatedDept", result.OUName)
}

func TestGetMappingByDept(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建表
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

	mapper := NewDeptOUmapper(db)

	ctx := context.Background()

	// 测试未找到
	_, err = mapper.GetMappingByDept(ctx, "non-existent")
	assert.Error(t, err)

	// 准备测试数据
	mapping := &models.DeptOUMapping{
		ID:          "test-id-4",
		DeptID:      "dept-1",
		ADConfigID:  "config-1",
		OUDN:        "OU=TestDept,DC=company,DC=com",
		OUName:      "TestDept",
		SyncEnabled: true,
		SyncStatus:  "synced",
	}
	err = mapper.UpsertMapping(ctx, mapping)
	assert.NoError(t, err)

	// 测试查找
	result, err := mapper.GetMappingByDept(ctx, "dept-1")
	assert.NoError(t, err)
	assert.Equal(t, "TestDept", result.OUName)
	assert.Equal(t, "synced", result.SyncStatus)
}
