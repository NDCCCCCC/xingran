package addomain

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ConfigService AD配置管理服务
type ConfigService struct {
	db   *gorm.DB
	pool AccountPool // Phase 38 Wave 1 DI 脚手架（38-02 将用于 TestConnection PickFirstConnect 改造）
}

// NewConfigService 创建配置服务
// Phase 38 Wave 1: 注入 AccountPool 字段（38-02 将用于 TestConnection PickFirstConnect 改造）。
func NewConfigService(db *gorm.DB, pool AccountPool) *ConfigService {
	return &ConfigService{db: db, pool: pool}
}

// configAllowedSortFields AD配置可排序字段白名单(对应 sys_ad_config 表列名)。
var configAllowedSortFields = map[string]string{
	"configName":   "config_name",
	"serverAddress": "server_address",
	"domainName":   "domain_name",
	"status":       "status",
	"createdAt":    "created_at",
}

// ListRequest 配置列表请求
type ListRequest struct {
	base.BaseListRequest
	Status *int `json:"status,omitempty"`
}

// GetList 获取AD配置列表
func (s *ConfigService) GetList(ctx context.Context, req *ListRequest) ([]models.ADConfig, int64, error) {
	var configs []models.ADConfig
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADConfig{})

	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询AD配置总数失败: %w", err)
	}

	offset := (req.Current - 1) * req.PageSize
	query = base.ApplySort(query, req.BaseListRequest, configAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(req.PageSize).Find(&configs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询AD配置列表失败: %w", err)
	}

	return configs, total, nil
}

// GetByID 根据ID获取AD配置
func (s *ConfigService) GetByID(ctx context.Context, id string) (*models.ADConfig, error) {
	var config models.ADConfig
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("AD配置不存在")
		}
		return nil, fmt.Errorf("查询AD配置失败: %w", err)
	}
	return &config, nil
}

// CreateRequest 创建AD配置请求
type CreateRequest struct {
	ConfigName    string `json:"configName" binding:"required,max=100"`
	ServerAddress string `json:"serverAddress" binding:"required,max=255"`
	ServerPort    int    `json:"serverPort" binding:"required,min=1,max=65535"`
	DomainName    string `json:"domainName" binding:"required,max=255"`
	BaseDN        string `json:"baseDn" binding:"required,max=500"`
	UseSSL        bool   `json:"useSsl"`
	UseTLS        bool   `json:"useTls"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncInterval  int    `json:"syncInterval" binding:"min=60"`
	MemberOUDN    string `json:"memberOuDn,omitempty"` // 本部部门分组OU DN
}

// Create 创建AD配置
func (s *ConfigService) Create(ctx context.Context, req *CreateRequest, creatorID string) (*models.ADConfig, error) {
	// 保存配置
	config := &models.ADConfig{
		BaseModel: models.BaseModel{
			CreatedBy: creatorID,
			Version:   0,
		},
		ConfigName:    req.ConfigName,
		ServerAddress: req.ServerAddress,
		ServerPort:    req.ServerPort,
		DomainName:    req.DomainName,
		BaseDN:        req.BaseDN,
		UseSSL:        req.UseSSL,
		UseTLS:        req.UseTLS,
		SyncEnabled:   req.SyncEnabled,
		SyncInterval:  req.SyncInterval,
		MemberOUDN:    req.MemberOUDN,
		Status:        models.ADConfigStatusEnabled,
	}

	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return nil, fmt.Errorf("保存AD配置失败: %w", err)
	}

	return config, nil
}

// UpdateRequest 更新AD配置请求
type UpdateRequest struct {
	ID            string `json:"-"`
	ConfigName    string `json:"configName" binding:"required,max=100"`
	ServerAddress string `json:"serverAddress" binding:"required,max=255"`
	ServerPort    int    `json:"serverPort" binding:"required,min=1,max=65535"`
	DomainName    string `json:"domainName" binding:"required,max=255"`
	BaseDN        string `json:"baseDn" binding:"required,max=500"`
	UseSSL        bool   `json:"useSsl"`
	UseTLS        bool   `json:"useTls"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncInterval  int    `json:"syncInterval" binding:"min=60"`
	MemberOUDN    string `json:"memberOuDn,omitempty"` // 本部部门分组OU DN
	Status        *int   `json:"status,omitempty"`
}

// Update 更新AD配置
func (s *ConfigService) Update(ctx context.Context, req *UpdateRequest, updaterID string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	// 构建更新数据
	updates := map[string]interface{}{
		"config_name":    req.ConfigName,
		"server_address": req.ServerAddress,
		"server_port":    req.ServerPort,
		"domain_name":    req.DomainName,
		"base_dn":        req.BaseDN,
		"use_ssl":        req.UseSSL,
		"use_tls":        req.UseTLS,
		"sync_enabled":   req.SyncEnabled,
		"sync_interval":  req.SyncInterval,
		"member_ou_dn":   req.MemberOUDN,
		"updated_by":     updaterID,
	}

	// 更新状态
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// 更新版本号
	updates["version"] = config.Version + 1

	if err := s.db.WithContext(ctx).Model(&config).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新AD配置失败: %w", err)
	}

	return nil
}

// Delete 删除AD配置
func (s *ConfigService) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.ADConfig{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("删除AD配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("AD配置不存在")
	}
	return nil
}

// TestConnection 测试AD连接
func (s *ConfigService) TestConnection(ctx context.Context, config *models.ADConfig) error {
	applogger.Infof("[ADTest] 开始测试AD连接: configID=%s, server=%s:%d, baseDN=%s", config.ID, config.ServerAddress, config.ServerPort, config.BaseDN)

	// Phase 38: 改走 FailoverClient.PickFirstConnect（SP-4）
	// 只建连接 + bind 成功即返回，不做后续 operation；
	// 唯一允许闭包外用 client 的场景（caller 负责 defer client.Close()）
	fc := NewFailoverClient(s.pool, config)
	client, _, err := fc.PickFirstConnect(ctx)
	if err != nil {
		applogger.Errorf("[ADTest] LDAP连接失败: %v", err)
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 连接测试失败：账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return fmt.Errorf("AD 连接测试失败（账号池无可用账号或全部 bind 失败）: %w", err)
	}
	defer client.Close()

	applogger.Infof("[ADTest] LDAP连接测试成功")
	return nil
}
