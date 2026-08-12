package system

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

// EmailConfigService 邮箱配置服务接口
type EmailConfigService interface {
	Create(ctx context.Context, req *EmailConfigCreateRequest) error
	Update(ctx context.Context, req *EmailConfigUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*EmailConfigDTO, error)
	List(ctx context.Context, params EmailConfigListParams) (*PageResult, error)
}

// emailConfigService 邮箱配置服务实现
type emailConfigService struct {
	db *gorm.DB
}

// NewEmailConfigService 创建邮箱配置服务实例
func NewEmailConfigService(db *gorm.DB) EmailConfigService {
	return &emailConfigService{db: db}
}

// ==================== 请求/响应类型 ====================

// EmailConfigDTO 邮箱配置数据传输对象
type EmailConfigDTO struct {
	ID          string `json:"id"`
	ConfigName  string `json:"configName"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromName    string `json:"fromName"`
	FromEmail   string `json:"fromEmail"`
	UseSSL      bool   `json:"useSsl"`
	UseSTARTTLS bool   `json:"useStartTls"`
	IsDefault   bool   `json:"isDefault"`
	Status      int    `json:"status"`
	Remark      string `json:"remark"`
	CreatedBy   string `json:"createdBy"`
	UpdatedBy   string `json:"updatedBy"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// EmailConfigListParams 邮箱配置列表查询参数
type EmailConfigListParams struct {
	Status   *int
	Current  int
	PageSize int
}

// DefaultEmailConfigListParams 默认列表参数
func DefaultEmailConfigListParams() EmailConfigListParams {
	return EmailConfigListParams{
		Current:  1,
		PageSize: 10,
	}
}

// EmailConfigCreateRequest 创建邮箱配置请求
type EmailConfigCreateRequest struct {
	ConfigName  string `json:"configName" binding:"required,min=2,max=100"`
	Host        string `json:"host" binding:"required,max=255"`
	Port        int    `json:"port" binding:"required,min=1,max=65535"`
	Username    string `json:"username" binding:"required,max=255"`
	Password    string `json:"password" binding:"required"`
	FromName    string `json:"fromName" binding:"max=100"`
	FromEmail   string `json:"fromEmail" binding:"email,max=255"`
	UseSSL      bool   `json:"useSsl"`
	UseSTARTTLS bool   `json:"useStartTls"`
	IsDefault   bool   `json:"isDefault"`
	Status      int    `json:"status" binding:"oneof=0 1"`
	Remark      string `json:"remark" binding:"max=500"`
}

// EmailConfigUpdateRequest 更新邮箱配置请求
type EmailConfigUpdateRequest struct {
	ID          string  `json:"-"`
	ConfigName  *string `json:"configName" binding:"omitempty,min=2,max=100"`
	Host        *string `json:"host" binding:"omitempty,max=255"`
	Port        *int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username    *string `json:"username" binding:"omitempty,max=255"`
	Password    *string `json:"password" binding:"omitempty"`
	FromName    *string `json:"fromName" binding:"omitempty,max=100"`
	FromEmail   *string `json:"fromEmail" binding:"omitempty,max=255"`
	UseSSL      *bool   `json:"useSsl"`
	UseSTARTTLS *bool   `json:"useStartTls"`
	IsDefault   *bool   `json:"isDefault"`
	Status      *int    `json:"status" binding:"omitempty,oneof=0 1"`
	Remark      *string `json:"remark" binding:"omitempty,max=500"`
}

// ==================== 服务方法实现 ====================

func (s *emailConfigService) Create(ctx context.Context, req *EmailConfigCreateRequest) error {
	// 检查是否已存在邮件配置
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("del_flag = 0").Count(&count).Error; err != nil {
		return fmt.Errorf("检查邮件配置失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("邮件配置已存在，系统只允许一条邮件配置。请先删除现有配置后再创建")
	}

	// 加密密码
	encryptedPassword, err := services.EncryptPassword(req.Password, "")
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	config := &models.EmailConfig{
		ConfigName:  req.ConfigName,
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		Password:    encryptedPassword,
		FromName:    req.FromName,
		FromEmail:   req.FromEmail,
		UseSSL:      req.UseSSL,
		UseSTARTTLS: req.UseSTARTTLS,
		IsDefault:   true,
		Status:      req.Status,
		Remark:      req.Remark,
	}

	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("创建邮箱配置失败: %w", err)
	}

	return nil
}

func (s *emailConfigService) Update(ctx context.Context, req *EmailConfigUpdateRequest) error {
	// 检查配置是否存在
	var existing models.EmailConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", req.ID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("邮箱配置不存在")
		}
		return fmt.Errorf("查询邮箱配置失败: %w", err)
	}

	// 构建更新数据
	updates := make(map[string]interface{})

	if req.ConfigName != nil {
		updates["config_name"] = *req.ConfigName
	}
	if req.Host != nil {
		updates["host"] = *req.Host
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Password != nil {
		encryptedPassword, err := services.EncryptPassword(*req.Password, "")
		if err != nil {
			return fmt.Errorf("密码加密失败: %w", err)
		}
		updates["password"] = encryptedPassword
	}
	if req.FromName != nil {
		updates["from_name"] = *req.FromName
	}
	if req.FromEmail != nil {
		updates["from_email"] = *req.FromEmail
	}
	if req.UseSSL != nil {
		updates["use_ssl"] = *req.UseSSL
	}
	if req.UseSTARTTLS != nil {
		updates["use_starttls"] = *req.UseSTARTTLS
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if err := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新邮箱配置失败: %w", err)
	}

	return nil
}

func (s *emailConfigService) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("id = ?", id).Update("del_flag", 1)
	if result.Error != nil {
		return fmt.Errorf("删除邮箱配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("邮箱配置不存在")
	}
	return nil
}

func (s *emailConfigService) GetByID(ctx context.Context, id string) (*EmailConfigDTO, error) {
	var config models.EmailConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", id).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("邮箱配置不存在")
		}
		return nil, fmt.Errorf("查询邮箱配置失败: %w", err)
	}

	return toEmailConfigDTO(&config), nil
}

func (s *emailConfigService) List(ctx context.Context, params EmailConfigListParams) (*PageResult, error) {
	var total int64
	var configs []models.EmailConfig

	query := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("del_flag = 0")

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计邮箱配置数量失败: %w", err)
	}

	offset := (params.Current - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("查询邮箱配置列表失败: %w", err)
	}

	// 转换为DTO
	dtos := make([]EmailConfigDTO, len(configs))
	for i, config := range configs {
		dtos[i] = *toEmailConfigDTO(&config)
	}

	return &PageResult{
		List:     dtos,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// ==================== 辅助函数 ====================

// toEmailConfigDTO 转换为DTO
func toEmailConfigDTO(config *models.EmailConfig) *EmailConfigDTO {
	return &EmailConfigDTO{
		ID:          config.ID,
		ConfigName:  config.ConfigName,
		Host:        config.Host,
		Port:        config.Port,
		Username:    config.Username,
		Password:    config.Password,
		FromName:    config.FromName,
		FromEmail:   config.FromEmail,
		UseSSL:      config.UseSSL,
		UseSTARTTLS: config.UseSTARTTLS,
		IsDefault:   config.IsDefault,
		Status:      config.Status,
		Remark:      config.Remark,
		CreatedBy:   config.CreatedBy,
		UpdatedBy:   config.UpdatedBy,
		CreatedAt:   config.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
