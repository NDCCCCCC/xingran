package system

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// APINotificationConfigService API通知配置服务接口
type APINotificationConfigService interface {
	Create(ctx context.Context, req *APINotificationConfigCreateRequest) error
	Update(ctx context.Context, req *APINotificationConfigUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.APINotificationConfig, error)
	List(ctx context.Context, params APINotificationConfigListParams) (*PageResult, error)
}

// apiNotificationConfigService API通知配置服务实现
type apiNotificationConfigService struct {
	db *gorm.DB
}

// NewAPINotificationConfigService 创建API通知配置服务实例
func NewAPINotificationConfigService(db *gorm.DB) APINotificationConfigService {
	return &apiNotificationConfigService{db: db}
}

// ==================== 请求/响应类型 ====================

// APINotificationConfigListParams API通知配置列表查询参数
type APINotificationConfigListParams struct {
	ConfigType *string
	Status     *int
	Current    int
	PageSize   int
}

// DefaultAPINotificationConfigListParams 默认列表参数
func DefaultAPINotificationConfigListParams() APINotificationConfigListParams {
	return APINotificationConfigListParams{
		Current:  1,
		PageSize: 10,
	}
}

// APINotificationConfigCreateRequest 创建API通知配置请求
type APINotificationConfigCreateRequest struct {
	ConfigName   string                 `json:"configName" binding:"required,min=2,max=100"`
	ConfigType   models.APIConfigType   `json:"configType" binding:"required,oneof=sms webhook push"`
	APIURL       string                 `json:"apiUrl" binding:"required,max=500"`
	APIMethod    string                 `json:"apiMethod" binding:"required,oneof=GET POST"`
	Headers      map[string]interface{} `json:"headers"`
	TemplateBody string                 `json:"templateBody"`
	AuthType     models.AuthType        `json:"authType" binding:"oneof=none basic bearer apikey"`
	AuthConfig   map[string]interface{} `json:"authConfig"`
	RetryCount   int                    `json:"retryCount" binding:"min=0,max=10"`
	Timeout      int                    `json:"timeout" binding:"min=1,max=300"`
	IsDefault    bool                   `json:"isDefault"`
	Status       int                    `json:"status" binding:"oneof=0 1"`
	Remark       string                 `json:"remark" binding:"max=500"`
}

// APINotificationConfigUpdateRequest 更新API通知配置请求
type APINotificationConfigUpdateRequest struct {
	ID           string                 `json:"-"`
	ConfigName   *string                `json:"configName" binding:"omitempty,min=2,max=100"`
	APIURL       *string                `json:"apiUrl" binding:"omitempty,max=500"`
	APIMethod    *string                `json:"apiMethod" binding:"omitempty,oneof=GET POST"`
	Headers      map[string]interface{} `json:"headers"`
	TemplateBody *string                `json:"templateBody"`
	AuthType     *models.AuthType       `json:"authType" binding:"omitempty,oneof=none basic bearer apikey"`
	AuthConfig   map[string]interface{} `json:"authConfig"`
	RetryCount   *int                   `json:"retryCount" binding:"omitempty,min=0,max=10"`
	Timeout      *int                   `json:"timeout" binding:"omitempty,min=1,max=300"`
	IsDefault    *bool                  `json:"isDefault"`
	Status       *int                   `json:"status" binding:"omitempty,oneof=0 1"`
	Remark       *string                `json:"remark" binding:"omitempty,max=500"`
}

// ==================== 服务方法实现 ====================

func (s *apiNotificationConfigService) Create(ctx context.Context, req *APINotificationConfigCreateRequest) error {
	config := &models.APINotificationConfig{
		ConfigName:   req.ConfigName,
		ConfigType:   req.ConfigType,
		APIURL:       req.APIURL,
		APIMethod:    req.APIMethod,
		Headers:      req.Headers,
		TemplateBody: req.TemplateBody,
		AuthType:     req.AuthType,
		AuthConfig:   req.AuthConfig,
		RetryCount:   req.RetryCount,
		Timeout:      req.Timeout,
		IsDefault:    req.IsDefault,
		Status:       req.Status,
		Remark:       req.Remark,
	}

	// 如果设置为默认，先取消同类型的其他默认配置
	if config.IsDefault {
		s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).
			Where("config_type = ? AND del_flag = 0", config.ConfigType).
			Update("is_default", false)
	}

	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("创建API通知配置失败: %w", err)
	}

	return nil
}

func (s *apiNotificationConfigService) Update(ctx context.Context, req *APINotificationConfigUpdateRequest) error {
	// 检查配置是否存在
	var existing models.APINotificationConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", req.ID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("API通知配置不存在")
		}
		return fmt.Errorf("查询API通知配置失败: %w", err)
	}

	// 构建更新数据
	updates := make(map[string]interface{})

	if req.ConfigName != nil {
		updates["config_name"] = *req.ConfigName
	}
	if req.APIURL != nil {
		updates["api_url"] = *req.APIURL
	}
	if req.APIMethod != nil {
		updates["api_method"] = *req.APIMethod
	}
	if req.Headers != nil {
		updates["headers"] = req.Headers
	}
	if req.TemplateBody != nil {
		updates["template_body"] = *req.TemplateBody
	}
	if req.AuthType != nil {
		updates["auth_type"] = *req.AuthType
	}
	if req.AuthConfig != nil {
		updates["auth_config"] = req.AuthConfig
	}
	if req.RetryCount != nil {
		updates["retry_count"] = *req.RetryCount
	}
	if req.Timeout != nil {
		updates["timeout"] = *req.Timeout
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
		// 如果设置为默认，先取消同类型的其他默认配置
		if *req.IsDefault {
			s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).
				Where("config_type = ? AND id != ? AND del_flag = 0", existing.ConfigType, req.ID).
				Update("is_default", false)
		}
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if err := s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新API通知配置失败: %w", err)
	}

	return nil
}

func (s *apiNotificationConfigService) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).Where("id = ?", id).Update("del_flag", 1)
	if result.Error != nil {
		return fmt.Errorf("删除API通知配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API通知配置不存在")
	}
	return nil
}

func (s *apiNotificationConfigService) GetByID(ctx context.Context, id string) (*models.APINotificationConfig, error) {
	var config models.APINotificationConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", id).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API通知配置不存在")
		}
		return nil, fmt.Errorf("查询API通知配置失败: %w", err)
	}

	return &config, nil
}

func (s *apiNotificationConfigService) List(ctx context.Context, params APINotificationConfigListParams) (*PageResult, error) {
	var total int64
	var configs []models.APINotificationConfig

	query := s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).Where("del_flag = 0")

	if params.ConfigType != nil && *params.ConfigType != "" {
		query = query.Where("config_type = ?", *params.ConfigType)
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计API配置数量失败: %w", err)
	}

	offset := (params.Current - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("查询API配置列表失败: %w", err)
	}

	return &PageResult{
		List:     configs,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}
