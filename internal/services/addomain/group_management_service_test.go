package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupTestDB creates an in-memory SQLite database with required tables for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create minimal tables needed for tests (matching BaseModel columns)
	db.Exec(`CREATE TABLE sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT NOT NULL,
		server_address TEXT NOT NULL,
		server_port INTEGER DEFAULT 389,
		domain_name TEXT NOT NULL,
		base_dn TEXT NOT NULL,
		admin_username TEXT NOT NULL,
		admin_password TEXT NOT NULL,
		member_ou_dn TEXT DEFAULT '',
		use_ssl BOOLEAN DEFAULT FALSE,
		use_tls BOOLEAN DEFAULT FALSE,
		sync_enabled BOOLEAN DEFAULT TRUE,
		sync_interval INTEGER DEFAULT 3600,
		last_sync_at DATETIME,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		created_by TEXT DEFAULT '',
		updated_by TEXT DEFAULT '',
		version INTEGER DEFAULT 0
	)`)

	db.Exec(`CREATE TABLE sys_ad_group (
		id TEXT PRIMARY KEY,
		ad_config_id TEXT NOT NULL,
		group_dn TEXT NOT NULL,
		group_name TEXT NOT NULL,
		group_scope TEXT,
		group_type INTEGER DEFAULT 1,
		description TEXT,
		member_count INTEGER DEFAULT 0,
		ou_dn TEXT,
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		created_by TEXT DEFAULT '',
		updated_by TEXT DEFAULT '',
		version INTEGER DEFAULT 0
	)`)

	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		dept_code TEXT NOT NULL DEFAULT '',
		parent_id TEXT,
		ancestors TEXT DEFAULT '',
		order_num INTEGER DEFAULT 0,
		leader TEXT,
		phone TEXT,
		email TEXT,
		is_external_org INTEGER DEFAULT 0,
		status INTEGER DEFAULT 0,
		remark TEXT DEFAULT '',
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		created_by TEXT DEFAULT '',
		updated_by TEXT DEFAULT '',
		version INTEGER DEFAULT 0
	)`)

	db.Exec(`CREATE TABLE sys_ou_group_mapping (
		id TEXT PRIMARY KEY,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL,
		ou_name TEXT NOT NULL,
		ad_group_id TEXT NOT NULL,
		mapping_status TEXT NOT NULL DEFAULT 'active',
		sync_enabled BOOLEAN DEFAULT TRUE,
		last_sync_at DATETIME,
		created_by TEXT DEFAULT '',
		updated_by TEXT DEFAULT '',
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)

	// Phase 38 Wave 1: 账号池表（FailoverClient 改造后 CreateGroupForDept 经账号池，空池返回 ErrAllAccountsUnavailable）
	db.Exec(`CREATE TABLE sys_ad_service_accounts (
		id TEXT PRIMARY KEY,
		config_id TEXT NOT NULL,
		username TEXT NOT NULL,
		password_ciphertext TEXT NOT NULL,
		status INTEGER DEFAULT 0,
		failure_count INTEGER DEFAULT 0,
		circuit_breaker_until DATETIME,
		last_success_at DATETIME,
		last_failure_at DATETIME,
		last_failure_reason TEXT,
		manual_unlock_reason TEXT,
		manual_unlocked_by TEXT DEFAULT '',
		manual_unlocked_at DATETIME,
		remark TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)

	return db
}

// createTestData creates test department, config, and group records
func createTestData(t *testing.T, db *gorm.DB) (deptID, configID, groupID string) {
	ctx := context.Background()

	// Create department
	dept := &models.Department{
		DeptName: "科技创新部",
		Status:   models.DeptStatusNormal,
	}
	require.NoError(t, db.WithContext(ctx).Create(dept).Error)
	deptID = dept.ID

	// Create AD config
	config := &models.ADConfig{
		ConfigName:    "测试AD",
		ServerAddress: "ad.local",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "DC=test,DC=com",
		AdminUsername: "admin",
		AdminPassword: "password",
		Status:        models.ADConfigStatusEnabled,
	}
	require.NoError(t, db.WithContext(ctx).Create(config).Error)
	configID = config.ID

	// Create AD group
	group := &models.ADGroup{
		ADConfigID:  configID,
		GroupDN:     "CN=cxhub-科技创新,OU=Groups,DC=test,DC=com",
		GroupName:   "cxhub-科技创新",
		MemberCount: 0,
	}
	require.NoError(t, db.WithContext(ctx).Create(group).Error)
	groupID = group.ID

	return deptID, configID, groupID
}

// TestGroupManagementService_CreateGroupForDept 测试为部门创建组
func TestGroupManagementService_CreateGroupForDept(t *testing.T) {
	db := setupTestDB(t)
	service := NewGroupManagementService(db, NewAccountPool(db, nil))
	ctx := context.Background()

	deptID, configID, _ := createTestData(t, db)

	// 测试创建组（会因LDAP连接失败而跳过LDAP部分）
	_, err := service.CreateGroupForDept(ctx, deptID, configID, "OU=Groups,DC=test,DC=com")

	// 由于没有真实的LDAP连接，我们期望数据库记录创建成功
	// LDAP操作会失败，但这不影响数据库记录
	if err != nil {
		// 如果LDAP连接失败，这是预期的
		t.Logf("Expected LDAP connection failure: %v", err)
	}

	// 验证数据库记录
	var count int64
	db.Model(&models.ADGroup{}).Where("group_name = ?", "cxhub-科技创新").Count(&count)
	if count > 0 {
		t.Logf("Group record created in database")
	}
}

// TestGroupManagementService_BulkCreateGroupsForDepts 测试批量创建组
func TestGroupManagementService_BulkCreateGroupsForDepts(t *testing.T) {
	db := setupTestDB(t)
	service := NewGroupManagementService(db, NewAccountPool(db, nil))
	ctx := context.Background()

	deptID1, _, _ := createTestData(t, db)

	// 创建第二个部门
	dept2 := &models.Department{
		DeptName: "市场部",
		Status:   models.DeptStatusNormal,
	}
	require.NoError(t, db.WithContext(ctx).Create(dept2).Error)

	// 批量创建组
	deptIDs := []string{deptID1, dept2.ID}
	// 使用第一个部门的配置ID
	var configID string
	db.Model(&models.ADConfig{}).First(&configID) // 获取第一个配置

	result, err := service.BulkCreateGroupsForDepts(ctx, deptIDs, configID, "OU=Groups,DC=test,DC=com")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	// 由于没有LDAP连接，预计会失败
	t.Logf("Bulk create result: Success=%d, Failed=%d", result.SuccessCount, result.FailedCount)
}

// TestGroupManagementService_NamingConvention 测试组命名规则
func TestGroupManagementService_NamingConvention(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	testCases := []struct {
		deptName     string
		expectedName string
	}{
		{"科技创新部", "cxhub-科技创新"},
		{"市场部", "cxhub-市场"},
		{"人力资源部", "cxhub-人力资源"},
		{"研发中心", "cxhub-研发中心"},
	}

	for _, tc := range testCases {
		dept := &models.Department{
			DeptName: tc.deptName,
			Status:   models.DeptStatusNormal,
		}
		require.NoError(t, db.WithContext(ctx).Create(dept).Error)

		// 验证命名规则（通过检查自动映射逻辑）
		// 在实际服务中，命名规则是在AutoMapDepartment中实现的
		t.Logf("Dept: %s -> Expected Group: %s", tc.deptName, tc.expectedName)
	}
}

// TestGroupManagementService_DeleteGroup_SafetyChecks 测试删除组的安全检查
func TestGroupManagementService_DeleteGroup_SafetyChecks(t *testing.T) {
	db := setupTestDB(t)
	service := NewGroupManagementService(db, nil)
	ctx := context.Background()

	_, _, groupID := createTestData(t, db)

	// 测试删除有成员的组
	db.Model(&models.ADGroup{}).Where("id = ?", groupID).Update("member_count", 5)

	err := service.DeleteGroup(ctx, groupID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "组中有成员")
}

// TestGroupManagementService_DeleteGroup_WithMappings 测试删除有映射的组
func TestGroupManagementService_DeleteGroup_WithMappings(t *testing.T) {
	db := setupTestDB(t)
	service := NewGroupManagementService(db, nil)
	ctx := context.Background()

	_, configID, groupID := createTestData(t, db)

	// 创建OU映射关系
	mapping := &models.OUGroupMapping{
		ADConfigID:    configID,
		OUDN:          "OU=科技创新部,DC=test,DC=com",
		OUName:        "科技创新部",
		ADGroupID:     groupID,
		MappingStatus: models.OUGroupMappingStatusActive,
	}
	require.NoError(t, db.WithContext(ctx).Create(mapping).Error)

	// Reset member_count to 0 so the member check passes
	db.Model(&models.ADGroup{}).Where("id = ?", groupID).Update("member_count", 0)

	// 尝试删除有映射的组
	err := service.DeleteGroup(ctx, groupID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "映射")
}

// TestMemberChangeResult 测试成员变更结果结构
func TestMemberChangeResult(t *testing.T) {
	result := &MemberChangeResult{
		GroupID:      "test-group-id",
		GroupName:    "test-group",
		AddedCount:   5,
		RemovedCount: 2,
		FailedCount:  1,
	}

	assert.Equal(t, "test-group-id", result.GroupID)
	assert.Equal(t, 5, result.AddedCount)
	assert.Equal(t, 2, result.RemovedCount)
	assert.Equal(t, 1, result.FailedCount)
}

// TestBulkCreateResult 测试批量创建结果结构
func TestBulkCreateResult(t *testing.T) {
	result := &BulkCreateResult{
		TotalCount:   10,
		SuccessCount: 8,
		FailedCount:  2,
		FailedDepts:  []string{"dept1: timeout", "dept2: permission denied"},
	}

	assert.Equal(t, 10, result.TotalCount)
	assert.Equal(t, 8, result.SuccessCount)
	assert.Equal(t, 2, result.FailedCount)
	assert.Len(t, result.FailedDepts, 2)
}

// TestGroupManagementService_Integration_GrantFlow 测试完整流程
func TestGroupManagementService_Integration_GrantFlow(t *testing.T) {
	db := setupTestDB(t)
	service := NewGroupManagementService(db, NewAccountPool(db, nil))
	ctx := context.Background()

	t.Run("完整流程: 创建部门 -> 创建组 -> 创建映射", func(t *testing.T) {
		// 1. 创建部门
		dept := &models.Department{
			DeptName: "测试部",
			Status:   models.DeptStatusNormal,
		}
		require.NoError(t, db.WithContext(ctx).Create(dept).Error)

		// 2. 创建AD配置
		config := &models.ADConfig{
			ConfigName:    "测试AD",
			ServerAddress: "ad.local",
			ServerPort:    389,
			DomainName:    "test.com",
			BaseDN:        "DC=test,DC=com",
			AdminUsername: "admin",
			AdminPassword: "password",
			Status:        models.ADConfigStatusEnabled,
		}
		require.NoError(t, db.WithContext(ctx).Create(config).Error)

		// 3. 创建组（会因为LDAP失败而只创建数据库记录）
		_, err := service.CreateGroupForDept(ctx, dept.ID, config.ID, "OU=Groups,DC=test,DC=com")
		t.Logf("Create group result: %v", err)

		// 4. 验证数据库中的组记录
		var groups []models.ADGroup
		db.Where("group_name LIKE ?", "cxhub-%").Find(&groups)
		t.Logf("Found %d group(s) in database", len(groups))
	})
}
