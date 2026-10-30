package addomain

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// setupOUMapperDB 为 dept_ou_mapper 5 个补测用例构造 sqlite 内存库与 mapper，
// 避免每个测试重复 15 行 DDL。原 3 个测试保留旧结构以最小化 diff。
func setupOUMapperDB(t *testing.T) (*DeptOUmapper, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`CREATE TABLE sys_dept_ou_mapping (
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
	)`).Error)

	return NewDeptOUmapper(db), context.Background()
}

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

// ─── Phase 58 Plan 04 Task 1：F-B 配置作用域 OU 查询 ──────────────────────

// setupOUMapperDBWithHandle 与 setupOUMapperDB 同构，但额外把 *gorm.DB 暴露出来，
// 供需要按 ad_config_id 铺多行（同 dept_id 不同配置）的用例直接写库。
// 复用 user_ou_service_test.go 的 createDeptOUMappingTable（含 ad_config_id 列）。
func setupOUMapperDBWithHandle(t *testing.T) (*DeptOUmapper, *gorm.DB, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(createDeptOUMappingTable).Error)
	return NewDeptOUmapper(db), db, context.Background()
}

// seedOUMappingRow 直接插一行 sys_dept_ou_mapping（同 dept_id 多配置场景的必要铺法：
// UpsertMapping 的 ON CONFLICT 目标也会命中 (dept_id, ad_config_id)，但直插更直白可控）。
func seedOUMappingRow(t *testing.T, db *gorm.DB, id, deptID, adConfigID, ouDN string, syncEnabled bool) {
	t.Helper()
	enabled := 0
	if syncEnabled {
		enabled = 1
	}
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, sync_enabled, sync_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'synced', '2026-01-01 00:00:00', '2026-01-01 00:00:00')`,
		id, deptID, adConfigID, ouDN, ouDN, enabled,
	).Error)
}

// TestPhase58_FindOUDNByDeptIDForConfig_ScopedByConfig F-B：同 dept_id 在 N 个配置下
// 有 N 行合法记录，按 ad_config_id 收敛必须各取各的。
func TestPhase58_FindOUDNByDeptIDForConfig_ScopedByConfig(t *testing.T) {
	mapper, db, ctx := setupOUMapperDBWithHandle(t)

	// 刻意让 cfg-b 的行 id 排序在前（GORM 的 First() 会补 `ORDER BY id LIMIT 1`）：
	// 不收敛 ad_config_id 的旧实现会稳定返回 cfg-b 的 ou_dn，
	// 因此下面"cfg-a 必须取到 cfg-a 自己的 ou_dn"是真判别器，而不是碰巧命中。
	seedOUMappingRow(t, db, "m-aaa-cfgb", "dept-1", "cfg-b", "OU=IT,DC=na,DC=corp,DC=com", true)
	seedOUMappingRow(t, db, "m-zzz-cfga", "dept-1", "cfg-a", "OU=IT,DC=corp,DC=com", true)

	gotA, err := mapper.FindOUDNByDeptIDForConfig(ctx, "dept-1", "cfg-a")
	require.NoError(t, err)
	assert.Equal(t, "OU=IT,DC=corp,DC=com", gotA, "cfg-a 必须取到 cfg-a 自己的 ou_dn（F-B：禁跨配置取首行）")

	gotB, err := mapper.FindOUDNByDeptIDForConfig(ctx, "dept-1", "cfg-b")
	require.NoError(t, err)
	assert.Equal(t, "OU=IT,DC=na,DC=corp,DC=com", gotB, "cfg-b 必须取到 cfg-b 自己的 ou_dn")
}

// TestPhase58_FindOUDNByDeptIDForConfig_IgnoresDisabledMapping 唯一行 sync_enabled=0
// → 视为"无映射"（与排水器的 no-ou-mapping 终态对齐）。
func TestPhase58_FindOUDNByDeptIDForConfig_IgnoresDisabledMapping(t *testing.T) {
	mapper, db, ctx := setupOUMapperDBWithHandle(t)

	seedOUMappingRow(t, db, "m-disabled", "dept-1", "cfg-a", "OU=IT,DC=corp,DC=com", false)

	_, err := mapper.FindOUDNByDeptIDForConfig(ctx, "dept-1", "cfg-a")
	require.Error(t, err, "sync_enabled=0 的映射不得被当成有效目标 OU")
	assert.Contains(t, err.Error(), "未找到部门", "文案须与旧方法的无映射语义一致（排水器靠 error 判定 no-ou-mapping）")
}

// TestPhase58_FindOUDNByDeptIDForConfig_WrongConfigIsNotFound 配置作用域真的生效：
// 用不属于该配置的 adConfigID 调用必须查不到（而不是回落到任一其它配置的行）。
func TestPhase58_FindOUDNByDeptIDForConfig_WrongConfigIsNotFound(t *testing.T) {
	mapper, db, ctx := setupOUMapperDBWithHandle(t)

	seedOUMappingRow(t, db, "m-cfga", "dept-1", "cfg-a", "OU=IT,DC=corp,DC=com", true)

	_, err := mapper.FindOUDNByDeptIDForConfig(ctx, "dept-1", "cfg-other")
	require.Error(t, err, "不属于该配置的 dept 映射必须查不到（证明 ad_config_id 真的进了 WHERE）")
	assert.Contains(t, err.Error(), "未找到部门")
}
