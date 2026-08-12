package system

import (
	"context"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// OnEncryptionConfigChanged 当 sys.request.encryption.enabled 配置变更时调用的回调。
// 由 internal/core.Init() 在启动时注入 pkg/middleware.RefreshEncryptionConfigCache,
// 避免 services/system 直接 import pkg/middleware (会形成
// system → middleware → core → system 循环依赖)。
//
// 若调用方未注入(测试场景或独立构建),保持 nil,Update 静默跳过缓存失效。
var OnEncryptionConfigChanged func()

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
	// Statistics 参数配置统计(专用 COUNT 聚合,不依赖分页列表,不受 MaxPageSize=100 钳制)。
	Statistics(ctx context.Context) (*ConfigStatisticsResult, error)
}

// ConfigStatisticsResult 参数配置统计结果(config_type: Y/N 两类)。
type ConfigStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`   // config_type = 'Y'
	Inactive int64 `json:"inactive"` // config_type = 'N'
}

// configService 参数配置服务实现
type configService struct {
	db *gorm.DB
}

// NewConfigService 创建参数配置服务实例
func NewConfigService(db *gorm.DB) ConfigService {
	return &configService{db: db}
}

// configAllowedSortFields 参数配置可排序字段白名单。
// 值为 DB 列名(纯列名,无 JOIN)。
var configAllowedSortFields = map[string]string{
	"configName":  "config_name",
	"configKey":   "config_key",
	"configType":  "config_type",
	"createdAt":   "created_at",
}

// Statistics 统计参数配置(按 config_type 聚合,排除软删除)。
// 不依赖分页列表,避免「用 pageSize:1000 拉全量再 .length」被 MaxPageSize=100 钳制。
func (s *configService) Statistics(ctx context.Context) (*ConfigStatisticsResult, error) {
	var result ConfigStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&models.Config{}).
		Select(
			"COUNT(*) AS total",
			"COALESCE(SUM(CASE WHEN config_type = 'Y' THEN 1 ELSE 0 END), 0) AS active",
			"COALESCE(SUM(CASE WHEN config_type = 'N' THEN 1 ELSE 0 END), 0) AS inactive",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计参数配置失败: %w", err)
	}
	return &result, nil
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

	// 系统内置参数不能修改键名 (F-17 fix: 校验 ConfigKey 而非 ConfigName,
	// 原代码写反且 req 没有 ConfigKey 字段使保护不可达。
	// - req.ConfigKey 为空: 客户端未声明键名,跳过校验 (向后兼容)
	// - req.ConfigKey 与 DB 一致: 显式声明键名相同,允许更新
	// - req.ConfigKey 与 DB 不同: 拒绝)
	if config.IsSystem == models.ConfigIsSystemYes && req.ConfigKey != "" && req.ConfigKey != config.ConfigKey {
		return fmt.Errorf("系统内置参数不能修改键名 (expected=%q, got=%q)", config.ConfigKey, req.ConfigKey)
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

	// P1 fix: 修改请求加密开关后立即失效缓存,无需重启进程即可生效。
	// 通过包级回调变量 OnEncryptionConfigChanged 反转依赖,
	// 由 core.Init() 注入 middleware.RefreshEncryptionConfigCache。
	if config.ConfigKey == "sys.request.encryption.enabled" && OnEncryptionConfigChanged != nil {
		OnEncryptionConfigChanged()
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
	query = base.ApplySort(query, params.BaseListRequest, configAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
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
