package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// setupTestDB 创建内存数据库用于测试
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移测试模型
	err = db.AutoMigrate(&models.Department{}, &models.ADGroup{}, &models.ADConfig{}, &models.DeptGroupMapping{})
	require.NoError(t, err)

	return db
}

// createTestData 创建测试数据
func createTestData(t *testing.T, db *gorm.DB) (deptID, configID, groupID string) {
	ctx := context.Background()

	// 创建测试部门
	dept := &models.Department{
		DeptName: "科技创新部",
		Status:   models.DeptStatusNormal,
	}
	require.NoError(t, db.WithContext(ctx).Create(dept).Error)
	deptID = dept.ID

	// 创建测试AD配置
	config := &models.ADConfig{
		ConfigName:    "测试AD",
		ServerAddress: "test.local",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "DC=test,DC=com",
		AdminUsername: "admin",
		Status:        models.ADConfigStatusEnabled,
	}
	require.NoError(t, db.WithContext(ctx).Create(config).Error)
	configID = config.ID

	// 创建测试AD组
	group := &models.ADGroup{
		ADConfigID:  configID,
		GroupName:   "cxhub-科技创新",
		GroupDN:     "CN=cxhub-科技创新,OU=Groups,DC=test,DC=com",
		Description: "科技创新部门组",
	}
	require.NoError(t, db.WithContext(ctx).Create(group).Error)
	groupID = group.ID

	return deptID, configID, groupID
}

// TestDeptGroupMappingService_CreateMapping 测试创建映射
func TestDeptGroupMappingService_CreateMapping(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, groupID := createTestData(t, db)

	// 测试创建映射
	req := &CreateMappingRequest{
		DeptID:      deptID,
		ADGroupID:   groupID,
		ADConfigID:  configID,
		MappingType: models.MappingTypeAuto,
		SyncEnabled: true,
	}

	mapping, err := service.CreateMapping(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, deptID, mapping.DeptID)
	assert.Equal(t, groupID, mapping.ADGroupID)
	assert.Equal(t, configID, mapping.ADConfigID)
	assert.Equal(t, models.MappingTypeAuto, mapping.MappingType)
	assert.True(t, mapping.SyncEnabled)
	assert.Equal(t, "cxhub-科技创新", mapping.GroupName)
}

// TestDeptGroupMappingService_CreateMapping_DuplicateError 测试重复映射错误
func TestDeptGroupMappingService_CreateMapping_DuplicateError(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, groupID := createTestData(t, db)

	// 第一次创建
	req := &CreateMappingRequest{
		DeptID:     deptID,
		ADGroupID:  groupID,
		ADConfigID: configID,
	}

	_, err := service.CreateMapping(ctx, req)
	require.NoError(t, err)

	// 第二次创建相同映射应该失败
	_, err = service.CreateMapping(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在映射")
}

// TestDeptGroupMappingService_AutoMapDepartment 测试自动映射
func TestDeptGroupMappingService_AutoMapDepartment(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, _ := createTestData(t, db)

	// 测试自动映射
	mapping, err := service.AutoMapDepartment(ctx, deptID, configID)
	require.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, deptID, mapping.DeptID)
	assert.Equal(t, models.MappingTypeAuto, mapping.MappingType)
	assert.Equal(t, "cxhub-科技创新", mapping.GroupName)
}

// TestDeptGroupMappingService_AutoMapDepartment_NoMatchingGroup 测试无匹配组
func TestDeptGroupMappingService_AutoMapDepartment_NoMatchingGroup(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, _ := createTestData(t, db)

	// 删除匹配的组
	db.Exec("DELETE FROM sys_ad_group WHERE group_name = ?", "cxhub-科技创新")

	// 应该找不到匹配组
	_, err := service.AutoMapDepartment(ctx, deptID, configID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未找到匹配的AD组")
}

// TestDeptGroupMappingService_ListMappings 测试查询映射列表
func TestDeptGroupMappingService_ListMappings(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, groupID := createTestData(t, db)

	// 创建多个映射
	req := &CreateMappingRequest{
		DeptID:     deptID,
		ADGroupID:  groupID,
		ADConfigID: configID,
	}

	_, err := service.CreateMapping(ctx, req)
	require.NoError(t, err)

	// 测试列表查询
	listReq := &ListMappingsRequest{
		ADConfigID:   &configID,
		MappingType:  (*models.MappingType)(&[]string{"auto"}[0]),
		MappingStatus: (*models.MappingStatus)(&[]string{"active"}[0]),
		Current:      1,
		PageSize:     10,
	}

	resp, err := service.ListMappings(ctx, listReq)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.List, 1)
}

// TestDeptGroupMappingService_UpdateMapping 测试更新映射
func TestDeptGroupMappingService_UpdateMapping(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, groupID := createTestData(t, db)

	// 创建映射
	req := &CreateMappingRequest{
		DeptID:     deptID,
		ADGroupID:  groupID,
		ADConfigID: configID,
	}

	mapping, err := service.CreateMapping(ctx, req)
	require.NoError(t, err)

	// 更新映射
	inactiveStatus := models.MappingStatusInactive
	syncDisabled := false
	updateReq := &UpdateMappingRequest{
		MappingStatus: &inactiveStatus,
		SyncEnabled:   &syncDisabled,
		UpdatedBy:     "test-user",
	}

	err = service.UpdateMapping(ctx, mapping.ID, updateReq)
	require.NoError(t, err)

	// 验证更新
	updated, err := service.GetMapping(ctx, mapping.ID)
	require.NoError(t, err)
	assert.Equal(t, models.MappingStatusInactive, updated.MappingStatus)
	assert.False(t, updated.SyncEnabled)
}

// TestDeptGroupMappingService_DeleteMapping 测试删除映射
func TestDeptGroupMappingService_DeleteMapping(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, groupID := createTestData(t, db)

	// 创建映射
	req := &CreateMappingRequest{
		DeptID:     deptID,
		ADGroupID:  groupID,
		ADConfigID: configID,
	}

	mapping, err := service.CreateMapping(ctx, req)
	require.NoError(t, err)

	// 删除映射
	err = service.DeleteMapping(ctx, mapping.ID)
	require.NoError(t, err)

	// 验证软删除
	_, err = service.GetMapping(ctx, mapping.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

// TestDeptGroupMappingService_GetMappingByDept 测试根据部门获取映射
func TestDeptGroupMappingService_GetMappingByDept(t *testing.T) {
	db := setupTestDB(t)
	service := NewDeptGroupMappingService(db)
	ctx := context.Background()

	deptID, configID, groupID := createTestData(t, db)

	// 创建映射
	req := &CreateMappingRequest{
		DeptID:     deptID,
		ADGroupID:  groupID,
		ADConfigID: configID,
	}

	_, err := service.CreateMapping(ctx, req)
	require.NoError(t, err)

	// 根据部门ID查询映射
	mapping, err := service.GetMappingByDept(ctx, deptID)
	require.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, deptID, mapping.DeptID)
}
