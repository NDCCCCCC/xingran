package topology

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// FilterRuleService MAC过滤规则服务接口
type FilterRuleService interface {
	Create(ctx context.Context, req *CreateFilterRuleRequest) (*models.MACFilterRule, error)
	Update(ctx context.Context, id string, req *UpdateFilterRuleRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.MACFilterRule, error)
	List(ctx context.Context, params ListFilterRulesParams) (*PageResult, error)
	GetEffectiveRule(ctx context.Context, device *models.NetworkDevice) (*models.MACFilterRule, error)
}

// CreateFilterRuleRequest 创建过滤规则请求
type CreateFilterRuleRequest struct {
	RuleName        string             `json:"ruleName" binding:"required"`
	DeviceType      models.DeviceType  `json:"deviceType" binding:"required"`
	Vendor          models.DeviceVendor `json:"vendor"`
	MACThreshold    int                `json:"macThreshold" binding:"required,min=0"`
	EnableLLDPFilter bool              `json:"enableLLDPFilter"`
	Priority        int                `json:"priority" binding:"required,min=0"`
	Remark          string             `json:"remark"`
	CreatedBy       string             `json:"-"`
}

// UpdateFilterRuleRequest 更新过滤规则请求
type UpdateFilterRuleRequest struct {
	ID              string
	RuleName        string
	DeviceType      models.DeviceType
	Vendor          models.DeviceVendor
	MACThreshold    int
	EnableLLDPFilter bool
	Priority        int
	Remark          string
	UpdatedBy       string
}

// ListFilterRulesParams 查询过滤规则列表参数
type ListFilterRulesParams struct {
	DeviceType *models.DeviceType
	Vendor     *models.DeviceVendor
	Current    int
	PageSize   int
}

// PageResult 分页结果
type PageResult struct {
	List  []*models.MACFilterRule `json:"list"`
	Total int64                   `json:"total"`
}

// filterRuleService MAC过滤规则服务实现
type filterRuleService struct {
	db *gorm.DB
}

// NewFilterRuleService 创建MAC过滤规则服务实例
func NewFilterRuleService(db *gorm.DB) FilterRuleService {
	return &filterRuleService{db: db}
}

// Create 创建过滤规则
func (s *filterRuleService) Create(ctx context.Context, req *CreateFilterRuleRequest) (*models.MACFilterRule, error) {
	// 检查是否已存在相同的设备类型和厂商规则
	var existRule models.MACFilterRule
	vendorValue := string(req.Vendor)
	if vendorValue == "" {
		vendorValue = "NULL" // 用 NULL 表示任意厂商
	}
	err := s.db.WithContext(ctx).
		Where("device_type = ? AND (vendor = ? OR vendor = '' OR vendor IS NULL)", req.DeviceType, vendorValue).
		First(&existRule).Error
	if err == nil {
		return nil, fmt.Errorf("该设备类型和厂商的规则已存在")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查规则是否存在失败: %w", err)
	}

	rule := models.MACFilterRule{
		RuleName:         req.RuleName,
		DeviceType:       req.DeviceType,
		Vendor:           req.Vendor,
		MACThreshold:     req.MACThreshold,
		EnableLLDPFilter: req.EnableLLDPFilter,
		Priority:         req.Priority,
		IsSystem:         false,
		Remark:           req.Remark,
		CreatedBy:        req.CreatedBy,
	}

	// 验证规则数据
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(&rule).Error; err != nil {
		return nil, fmt.Errorf("创建过滤规则失败: %w", err)
	}

	return &rule, nil
}

// Update 更新过滤规则
func (s *filterRuleService) Update(ctx context.Context, id string, req *UpdateFilterRuleRequest) error {
	var rule models.MACFilterRule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("过滤规则不存在")
		}
		return fmt.Errorf("查询过滤规则失败: %w", err)
	}

	// 系统规则不能修改
	if rule.IsSystem {
		return fmt.Errorf("系统内置规则不能修改")
	}

	// 检查是否与其他规则冲突
	var existRule models.MACFilterRule
	vendorValue := string(req.Vendor)
	if vendorValue == "" {
		vendorValue = "NULL"
	}
	err := s.db.WithContext(ctx).
		Where("device_type = ? AND (vendor = ? OR vendor = '' OR vendor IS NULL) AND id != ?", req.DeviceType, vendorValue, id).
		First(&existRule).Error
	if err == nil {
		return fmt.Errorf("该设备类型和厂商的规则已存在")
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查规则冲突失败: %w", err)
	}

	// 构建更新数据
	updates := map[string]interface{}{
		"rule_name":           req.RuleName,
		"device_type":         req.DeviceType,
		"vendor":              req.Vendor,
		"mac_threshold":       req.MACThreshold,
		"enable_lldp_filter":  req.EnableLLDPFilter,
		"priority":            req.Priority,
		"remark":              req.Remark,
		"updated_by":          req.UpdatedBy,
	}

	if err := s.db.WithContext(ctx).Model(&rule).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新过滤规则失败: %w", err)
	}

	return nil
}

// Delete 删除过滤规则
func (s *filterRuleService) Delete(ctx context.Context, id string) error {
	var rule models.MACFilterRule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("过滤规则不存在")
		}
		return fmt.Errorf("查询过滤规则失败: %w", err)
	}

	// 系统规则不能删除
	if rule.IsSystem {
		return fmt.Errorf("系统内置规则不能删除")
	}

	// 软删除
	if err := s.db.WithContext(ctx).Delete(&rule).Error; err != nil {
		return fmt.Errorf("删除过滤规则失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取过滤规则
func (s *filterRuleService) GetByID(ctx context.Context, id string) (*models.MACFilterRule, error) {
	var rule models.MACFilterRule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("过滤规则不存在")
		}
		return nil, fmt.Errorf("查询过滤规则失败: %w", err)
	}
	return &rule, nil
}

// List 查询过滤规则列表
func (s *filterRuleService) List(ctx context.Context, params ListFilterRulesParams) (*PageResult, error) {
	query := s.db.WithContext(ctx).Model(&models.MACFilterRule{})

	// 添加过滤条件
	if params.DeviceType != nil {
		query = query.Where("device_type = ?", *params.DeviceType)
	}
	if params.Vendor != nil {
		query = query.Where("vendor = ?", *params.Vendor)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询过滤规则总数失败: %w", err)
	}

	// 分页查询
	var rules []*models.MACFilterRule
	offset := (params.Current - 1) * params.PageSize
	if err := query.Order("priority DESC, created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("查询过滤规则列表失败: %w", err)
	}

	return &PageResult{
		List:  rules,
		Total: total,
	}, nil
}

// GetEffectiveRule 获取设备的有效过滤规则（优先级解析）
func (s *filterRuleService) GetEffectiveRule(ctx context.Context, device *models.NetworkDevice) (*models.MACFilterRule, error) {
	var rule models.MACFilterRule

	// 1. 最具体规则：厂商 + 设备类型
	err := s.db.WithContext(ctx).
		Where("device_type = ? AND vendor = ?", device.DeviceType, device.Vendor).
		Order("priority DESC").
		First(&rule).Error

	if err == nil {
		return &rule, nil
	}

	// 2. 次具体规则：仅设备类型（厂商为空或NULL）
	err = s.db.WithContext(ctx).
		Where("device_type = ? AND (vendor = '' OR vendor IS NULL)", device.DeviceType).
		Order("priority DESC").
		First(&rule).Error

	if err == nil {
		return &rule, nil
	}

	// 3. 默认规则：硬编码兜底
	return &models.MACFilterRule{
		MACThreshold:     10,
		EnableLLDPFilter: true,
		Priority:         0,
	}, nil
}
