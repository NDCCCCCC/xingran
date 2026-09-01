package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/glebarez/sqlite"
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

	// Phase 40 修复后 FindDeptByOUDN JOIN sys_dept 过滤软删除，测试需补建该表
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		dept_code TEXT NOT NULL UNIQUE,
		parent_id TEXT,
		ancestors TEXT DEFAULT '',
		order_num INTEGER DEFAULT 0,
		leader TEXT,
		leader_name TEXT,
		leader_username TEXT,
		phone TEXT,
		email TEXT,
		is_external_org INTEGER DEFAULT 0,
		status INTEGER DEFAULT 0,
		remark TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		created_by TEXT,
		updated_by TEXT,
		version INTEGER DEFAULT 0
	)`)
	db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('dept-1', 'TestDept', 'TEST001', 0)`)

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

// ─── 补测计划 B1-P6: GetMappingByOU ──────────────────────────────

func TestGetMappingByOU(t *testing.T) {
	mapper, ctx := setupOUMapperDB(t)

	// 未找到
	_, err := mapper.GetMappingByOU(ctx, "OU=NonExistent,DC=company,DC=com")
	assert.Error(t, err)

	// 找到
	mapping := &models.DeptOUMapping{
		ID:         "map-ou-1",
		DeptID:     "dept-1",
		ADConfigID: "config-1",
		OUDN:       "OU=TargetOU,DC=company,DC=com",
		OUName:     "TargetDept",
		SyncStatus: "synced",
	}
	require.NoError(t, mapper.UpsertMapping(ctx, mapping))

	result, err := mapper.GetMappingByOU(ctx, "OU=TargetOU,DC=company,DC=com")
	require.NoError(t, err)
	assert.Equal(t, "dept-1", result.DeptID)
}

// ─── 补测计划 B1-P6: ListMappings ────────────────────────────────

func TestListMappings(t *testing.T) {
	mapper, ctx := setupOUMapperDB(t)

	// 空结果
	mappings, err := mapper.ListMappings(ctx, "config-empty")
	require.NoError(t, err)
	assert.Empty(t, mappings)

	// 多条结果
	for i := 0; i < 3; i++ {
		require.NoError(t, mapper.UpsertMapping(ctx, &models.DeptOUMapping{
			ID:         "map-list-" + string(rune('a'+i)),
			DeptID:     "dept-" + string(rune('1'+i)),
			ADConfigID: "config-list",
			OUDN:       "OU=Dept" + string(rune('A'+i)) + ",DC=company,DC=com",
			OUName:     "Dept" + string(rune('A'+i)),
			SyncStatus: "synced",
		}))
	}

	results, err := mapper.ListMappings(ctx, "config-list")
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

// ─── 补测计划 B1-P6: DeleteMapping ───────────────────────────

func TestDeleteMapping(t *testing.T) {
	mapper, ctx := setupOUMapperDB(t)

	require.NoError(t, mapper.UpsertMapping(ctx, &models.DeptOUMapping{
		ID:         "map-del-1",
		DeptID:     "dept-1",
		ADConfigID: "config-1",
		OUDN:       "OU=ToDelete,DC=company,DC=com",
		OUName:     "DeleteMe",
	}))

	require.NoError(t, mapper.DeleteMapping(ctx, "map-del-1"))

	// 已删除，查不到
	_, err := mapper.GetMappingByOU(ctx, "OU=ToDelete,DC=company,DC=com")
	assert.Error(t, err)
}

// ─── 补测计划 B1-P6: DisableMapping ─────────────────────────

func TestDisableMapping(t *testing.T) {
	mapper, ctx := setupOUMapperDB(t)

	require.NoError(t, mapper.UpsertMapping(ctx, &models.DeptOUMapping{
		ID:          "map-dis-1",
		DeptID:      "dept-1",
		ADConfigID:  "config-1",
		OUDN:        "OU=ToDisable,DC=company,DC=com",
		OUName:      "DisableMe",
		SyncEnabled: true,
	}))

	require.NoError(t, mapper.DisableMapping(ctx, "map-dis-1"))

	// 禁用后 sync_enabled = false
	result, err := mapper.GetMappingByOU(ctx, "OU=ToDisable,DC=company,DC=com")
	require.NoError(t, err)
	assert.False(t, result.SyncEnabled)
}

// ─── 补测计划 B1-P6: UpdateSyncStatus ──────────────────────

func TestUpdateSyncStatus(t *testing.T) {
	mapper, ctx := setupOUMapperDB(t)

	require.NoError(t, mapper.UpsertMapping(ctx, &models.DeptOUMapping{
		ID:         "map-status-1",
		DeptID:     "dept-1",
		ADConfigID: "config-1",
		OUDN:       "OU=ToStatus,DC=company,DC=com",
		OUName:     "StatusMe",
		SyncStatus: "pending",
	}))

	require.NoError(t, mapper.UpdateSyncStatus(ctx, "map-status-1", "synced"))

	result, err := mapper.GetMappingByOU(ctx, "OU=ToStatus,DC=company,DC=com")
	require.NoError(t, err)
	assert.Equal(t, "synced", result.SyncStatus)
}
