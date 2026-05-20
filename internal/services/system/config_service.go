package system

import (
	"context"
	"fmt"
	"strings"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"gorm.io/gorm"
)

// ConfigService 参数配置服务接口
type ConfigService interface {
	Create(ctx context.Context, req *requests.ConfigCreateRequest) error
	Update(ctx context.Context, req *requests.ConfigUpdateRequest) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
	GetByID(ctx context.Context, id string) (*models.Config, error)
	GetByKey(ctx context.Context, configKey string) (*models.Config, error)
	List(ctx context.Context, params requests.ConfigListParams) (*PageResult, error)
	RefreshCache(ctx context.Context) error
}

// configService 参数配置服务实现
type configService struct {
	db *gorm.DB
}

// NewConfigService 创建参数配置服务实例
func NewConfigService(db *gorm.DB) ConfigService {
	return &configService{db: db}
}

// ==================== 服务方法实现 ====================

func (s *configService) Create(ctx context.Context, req *requests.ConfigCreateRequest) error {
	// 检查配置键是否已存在
	var existConfig models.Config
	if err := s.db.WithContext(ctx).Where("config_key = ?", req.ConfigKey).First(&existConfig).Error; err == nil {
		return fmt.Errorf("配置键已存在")
	}

	config := models.Config{
		ConfigName:  req.ConfigName,
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigType:  req.ConfigType,
		IsSystem:    models.ConfigIsSystem(req.IsSystem),
		Remark:      toStringPtrStr(req.Remark),
	}

	if err := s.db.WithContext(ctx).Create(&config).Error; err != nil {
		return fmt.Errorf("创建参数配置失败: %w", err)
	}

	return nil
}

func (s *configService) Update(ctx context.Context, req *requests.ConfigUpdateRequest) error {
	var config models.Config
	if err := s.db.WithContext(ctx).First(&config, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("参数配置不存在")
		}
		return fmt.Errorf("查询参数配置失败: %w", err)
	}

	// 系统内置参数不能修改键名
	if config.IsSystem == models.ConfigIsSystemYes && req.ConfigName != config.ConfigName {
		return fmt.Errorf("系统内置参数不能修改")
	}

	// 验证请求加密开关的值（T-QUICK-01, T-QUICK-05 mitigation）
	if config.ConfigKey == "sys.request.encryption.enabled" {
		lowerValue := strings.ToLower(req.ConfigValue)
		if lowerValue != "true" && lowerValue != "false" && req.ConfigValue != "1" && req.ConfigValue != "0" {
			return fmt.Errorf("请求加密开关的值只能是 true 或 false")
		}
	}

	updates := map[string]interface{}{
		"config_name":  req.ConfigName,
		"config_value": req.ConfigValue,
		"config_type":  req.ConfigType,
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if err := s.db.WithContext(ctx).Model(&config).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新参数配置失败: %w", err)
	}

	// 如果更新的是请求加密开关，刷新缓存
	if config.ConfigKey == "sys.request.encryption.enabled" {
		// 这里需要导入 middleware 包来调用 RefreshEncryptionConfigCache
		// 为了避免循环依赖，我们在路由层处理缓存刷新
	}

	return nil
}

func (s *configService) Delete(ctx context.Context, id string) error {
	var config models.Config
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("参数配置不存在")
		}
		return fmt.Errorf("查询参数配置失败: %w", err)
	}

	// 系统内置参数不能删除
	if config.IsSystem == models.ConfigIsSystemYes {
		return fmt.Errorf("系统内置参数不能删除")
	}

	if err := s.db.WithContext(ctx).Delete(&config).Error; err != nil {
		return fmt.Errorf("删除参数配置失败: %w", err)
	}

	return nil
}

func (s *configService) BatchDelete(ctx context.Context, ids []string) error {
	// 检查是否有系统内置参数
	var systemCount int64
	if err := s.db.WithContext(ctx).Model(&models.Config{}).Where("id IN ? AND is_system = 1", ids).Count(&systemCount).Error; err == nil && systemCount > 0 {
		return fmt.Errorf("系统内置参数不能删除")
	}

	if err := s.db.WithContext(ctx).Where("id IN ? AND is_system = 0", ids).Delete(&models.Config{}).Error; err != nil {
		return fmt.Errorf("批量删除参数配置失败: %w", err)
	}

	return nil
}

func (s *configService) GetByID(ctx context.Context, id string) (*models.Config, error) {
	var config models.Config
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("参数配置不存在")
		}
		return nil, fmt.Errorf("查询参数配置失败: %w", err)
	}
	return &config, nil
}

func (s *configService) GetByKey(ctx context.Context, configKey string) (*models.Config, error) {
	var config models.Config
	err := s.db.WithContext(ctx).Where("config_key = ?", configKey).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("参数配置不存在")
		}
		return nil, fmt.Errorf("查询参数配置失败: %w", err)
	}
	return &config, nil
}

func (s *configService) List(ctx context.Context, params requests.ConfigListParams) (*PageResult, error) {
	var total int64
	var list []models.Config

	query := s.db.WithContext(ctx).Model(&models.Config{})

	if params.ConfigName != nil && *params.ConfigName != "" {
		query = query.Where("config_name LIKE ?", "%"+*params.ConfigName+"%")
	}
	if params.ConfigKey != nil && *params.ConfigKey != "" {
		query = query.Where("config_key LIKE ?", "%"+*params.ConfigKey+"%")
	}
	if params.ConfigType != nil && *params.ConfigType != "" {
		query = query.Where("config_type = ?", *params.ConfigType)
	}
	if params.BeginTime != nil && *params.BeginTime != "" {
		query = query.Where("created_at >= ?", *params.BeginTime)
	}
	if params.EndTime != nil && *params.EndTime != "" {
		query = query.Where("created_at <= ?", *params.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计参数配置总数失败: %w", err)
	}

	offset := (params.Current - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询参数配置列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

func (s *configService) RefreshCache(ctx context.Context) error {
	// TODO: 实现缓存刷新逻辑
	return nil
}

// toStringPtrStr 将 *string 转换为 string
func toStringPtrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
