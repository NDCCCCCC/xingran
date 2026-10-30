package topology

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/tests/fixtures"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试用的内存数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&models.MACFilterRule{})
	require.NoError(t, err)

	return db
}

// TestCreateRule 测试创建过滤规则
func TestCreateRule(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	req := &CreateFilterRuleRequest{
		RuleName:         "Test Rule",
		DeviceType:       models.DeviceTypeSwitch,
		Vendor:           models.VendorHuawei,
		MACThreshold:     20,
		EnableLLDPFilter: true,
		Priority:         10,
		Remark:           "Test remark",
		CreatedBy:        "test-user",
	}

	rule, err := svc.Create(context.Background(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID, "Rule ID should not be empty")
	assert.Equal(t, "Test Rule", rule.RuleName)
	assert.Equal(t, models.DeviceTypeSwitch, rule.DeviceType)
	assert.Equal(t, models.VendorHuawei, rule.Vendor)
	assert.Equal(t, 20, rule.MACThreshold)
	assert.True(t, rule.EnableLLDPFilter)
	assert.Equal(t, 10, rule.Priority)
}

// TestCreateDuplicateRule 测试创建重复规则
func TestCreateDuplicateRule(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	req := &CreateFilterRuleRequest{
		RuleName:   "Duplicate Rule",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		// ... 其他字段
		MACThreshold:     10,
		EnableLLDPFilter: true,
		Priority:         0,
		CreatedBy:        "test-user",
	}

	// 第一次创建
	_, err1 := svc.Create(context.Background(), req)
	assert.NoError(t, err1, "First creation should succeed")

	// 重复创建
	_, err2 := svc.Create(context.Background(), req)
	assert.Error(t, err2, "Duplicate creation should fail")
	assert.Contains(t, err2.Error(), "已存在", "Error should mention rule already exists")
}

// TestGetEffectiveRule 测试获取有效规则(优先级解析)
func TestGetEffectiveRule(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 创建厂商特定规则(高优先级)
	_, err := svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:         "Huawei Switch Rule",
		DeviceType:       models.DeviceTypeSwitch,
		Vendor:           models.VendorHuawei,
		MACThreshold:     50,
		EnableLLDPFilter: true,
		Priority:         10,
		CreatedBy:        "test-user",
	})
	require.NoError(t, err)

	// 创建设备类型规则(低优先级)
	_, err = svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:         "Generic Switch Rule",
		DeviceType:       models.DeviceTypeSwitch,
		Vendor:           "",
		MACThreshold:     10,
		EnableLLDPFilter: true,
		Priority:         0,
		CreatedBy:        "test-user",
	})
	require.NoError(t, err)

	device := &models.NetworkDevice{
		BaseModel:  models.BaseModel{ID: "device-1"},
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
	}

	rule, err := svc.GetEffectiveRule(context.Background(), device)

	require.NoError(t, err)
	assert.Equal(t, 50, rule.MACThreshold, "Should pick vendor-specific rule with higher priority")
	assert.Equal(t, models.VendorHuawei, rule.Vendor)
}

// TestPriorityResolution 测试优先级解析
func TestPriorityResolution(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 创建多个不同优先级的规则(不同厂商,避免冲突)
	_, err := svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:     "High Priority Rule",
		DeviceType:   models.DeviceTypeRouter,
		Vendor:       models.VendorHuawei,
		MACThreshold: 100,
		Priority:     100,
		CreatedBy:    "test-user",
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:     "Low Priority Rule",
		DeviceType:   models.DeviceTypeRouter,
		Vendor:       models.VendorRuijie, // 不同厂商
		MACThreshold: 50,
		Priority:     50,
		CreatedBy:    "test-user",
	})
	require.NoError(t, err)

	device := &models.NetworkDevice{
		BaseModel:  models.BaseModel{ID: "device-1"},
		DeviceType: models.DeviceTypeRouter,
		Vendor:     models.VendorHuawei, // 应该匹配高优先级规则
	}

	rule, err := svc.GetEffectiveRule(context.Background(), device)

	require.NoError(t, err)
	assert.Equal(t, 100, rule.MACThreshold, "Should pick highest priority rule")
}

// TestDefaultRuleFallback 测试默认规则兜底
func TestDefaultRuleFallback(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db) // 空数据库

	device := &models.NetworkDevice{
		BaseModel:  models.BaseModel{ID: "device-1"},
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
	}

	rule, err := svc.GetEffectiveRule(context.Background(), device)

	require.NoError(t, err)
	assert.Equal(t, 10, rule.MACThreshold, "Default threshold should be 10")
	assert.True(t, rule.EnableLLDPFilter, "Default LLDP filter should be enabled")
	assert.Equal(t, 0, rule.Priority, "Default priority should be 0")
}

// TestDeleteSystemRule 测试系统规则保护
func TestDeleteSystemRule(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 直接创建系统规则(通过db.Model避免验证问题)
	rule := models.MACFilterRule{
		RuleName:         "System Rule",
		DeviceType:       models.DeviceTypeSwitch,
		Vendor:           models.VendorRuijie,
		MACThreshold:     10,
		EnableLLDPFilter: true,
		Priority:         0,
		IsSystem:         true,
		CreatedBy:        "system",
	}
	err := db.Create(&rule).Error
	require.NoError(t, err)

	// 尝试删除
	err = svc.Delete(context.Background(), rule.ID)
	assert.Error(t, err, "System rule deletion should fail")
	assert.Contains(t, err.Error(), "系统内置规则不能删除", "Error should mention system rule protection")
}

// TestUpdateRule 测试更新规则
func TestUpdateRule(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 创建规则
	rule, err := svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:     "Original Name",
		MACThreshold: 10,
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       "",
		Priority:     0,
		CreatedBy:    "test-user",
	})
	require.NoError(t, err)

	// 更新规则
	err = svc.Update(context.Background(), rule.ID, &UpdateFilterRuleRequest{
		ID:           rule.ID,
		RuleName:     "Updated Name",
		MACThreshold: 20,
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       "",
		Priority:     5,
		UpdatedBy:    "test-user",
	})

	require.NoError(t, err)

	// 验证更新
	updated, err := svc.GetByID(context.Background(), rule.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.RuleName)
	assert.Equal(t, 20, updated.MACThreshold)
	assert.Equal(t, 5, updated.Priority)
}

// TestUpdateSystemRule 测试更新系统规则
func TestUpdateSystemRule(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 直接创建系统规则
	rule := models.MACFilterRule{
		RuleName:         "System Rule",
		DeviceType:       models.DeviceTypeFirewall,
		Vendor:           "",
		MACThreshold:     10,
		EnableLLDPFilter: true,
		Priority:         0,
		IsSystem:         true,
		CreatedBy:        "system",
	}
	err := db.Create(&rule).Error
	require.NoError(t, err)

	// 尝试更新
	err = svc.Update(context.Background(), rule.ID, &UpdateFilterRuleRequest{
		ID:           rule.ID,
		RuleName:     "Updated Name",
		MACThreshold: 20,
		UpdatedBy:    "test-user",
	})

	assert.Error(t, err, "System rule update should fail")
	assert.Contains(t, err.Error(), "系统内置规则不能修改", "Error should mention system rule protection")
}

// TestGetByID 测试根据ID获取规则
func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 创建规则
	rule, err := svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:     "Test Rule",
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       "",
		MACThreshold: 10,
		Priority:     0,
		CreatedBy:    "test-user",
	})
	require.NoError(t, err)

	// 获取规则
	retrieved, err := svc.GetByID(context.Background(), rule.ID)

	require.NoError(t, err)
	assert.Equal(t, rule.ID, retrieved.ID)
	assert.Equal(t, "Test Rule", retrieved.RuleName)
}

// TestGetByIDNotFound 测试获取不存在的规则
func TestGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	_, err := svc.GetByID(context.Background(), "non-existent-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "过滤规则不存在", "Error should mention rule not found")
}

// TestListRules 测试查询规则列表
func TestListRules(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 创建多个规则(不同厂商,避免冲突)
	vendors := []models.DeviceVendor{
		models.VendorHuawei,
		models.VendorRuijie,
		models.VendorH3C,
	}

	for i, vendor := range vendors {
		_, err := svc.Create(context.Background(), &CreateFilterRuleRequest{
			RuleName:     fmt.Sprintf("Rule %d", i+1),
			DeviceType:   models.DeviceTypeSwitch,
			Vendor:       vendor,
			MACThreshold: (i + 1) * 10,
			Priority:     i,
			CreatedBy:    "test-user",
		})
		require.NoError(t, err)
	}

	// 查询列表
	result, err := svc.List(context.Background(), ListFilterRulesParams{
		Current:  1,
		PageSize: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total, "Should have 3 rules")
	assert.Len(t, result.List, 3, "Should return 3 rules")
}

// TestListRulesWithFilter 测试带过滤条件的列表查询
func TestListRulesWithFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 创建不同类型的规则
	_, err := svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:     "Switch Rule",
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       "",
		MACThreshold: 10,
		Priority:     0,
		CreatedBy:    "test-user",
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:     "Router Rule",
		DeviceType:   models.DeviceTypeRouter,
		Vendor:       "",
		MACThreshold: 500,
		Priority:     0,
		CreatedBy:    "test-user",
	})
	require.NoError(t, err)

	// 按设备类型过滤
	deviceType := models.DeviceTypeSwitch
	result, err := svc.List(context.Background(), ListFilterRulesParams{
		DeviceType: &deviceType,
		Current:    1,
		PageSize:   10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "Should have 1 switch rule")
	assert.Equal(t, "Switch Rule", result.List[0].RuleName)
}

// TestMACFilterRuleValidation 测试规则验证
func TestMACFilterRuleValidation(t *testing.T) {
	tests := []struct {
		name        string
		threshold   int
		priority    int
		shouldError bool
	}{
		{"Valid rule", 10, 0, false},
		{"Negative threshold", -1, 0, true},
		{"Negative priority", 10, -1, true},
		{"Zero threshold", 0, 0, false}, // 0是有效的
		{"High threshold", 10000, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := models.MACFilterRule{
				MACThreshold: tt.threshold,
				Priority:     tt.priority,
			}

			err := rule.Validate()

			if tt.shouldError {
				assert.Error(t, err, "Should return validation error")
			} else {
				assert.NoError(t, err, "Should pass validation")
			}
		})
	}
}

// TestDeleteRule 测试删除规则
func TestDeleteRule(t *testing.T) {
	db := setupTestDB(t)
	svc := NewFilterRuleService(db)

	// 创建规则
	rule, err := svc.Create(context.Background(), &CreateFilterRuleRequest{
		RuleName:     "Rule to Delete",
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       "",
		MACThreshold: 10,
		Priority:     0,
		CreatedBy:    "test-user",
	})
	require.NoError(t, err)

	// 删除规则
	err = svc.Delete(context.Background(), rule.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = svc.GetByID(context.Background(), rule.ID)
	assert.Error(t, err, "Deleted rule should not be found")
}

// TestMockFilterRulesFixtures 测试过滤规则fixtures
func TestMockFilterRulesFixtures(t *testing.T) {
	rules := fixtures.MockFilterRules

	assert.NotEmpty(t, rules, "Mock filter rules should not be empty")

	// 验证规则结构
	for _, rule := range rules {
		assert.NotEmpty(t, rule.ID, "Rule ID should not be empty")
		assert.NotEmpty(t, rule.RuleName, "Rule name should not be empty")
		assert.Greater(t, rule.MACThreshold, -1, "Threshold should be non-negative")
		assert.Greater(t, rule.Priority, -1, "Priority should be non-negative")
	}
}

// TestDeviceTypeThresholds 测试设备类型阈值
func TestDeviceTypeThresholds(t *testing.T) {
	thresholds := fixtures.GetDeviceTypeThresholds

	assert.Equal(t, 500, thresholds[models.DeviceTypeRouter])
	assert.Equal(t, 10, thresholds[models.DeviceTypeSwitch])
	assert.Equal(t, 100, thresholds[models.DeviceTypeFirewall])
	assert.Equal(t, 50, thresholds[models.DeviceTypeLoadBalancer])
}
