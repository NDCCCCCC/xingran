package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// AuthCredentialService 授权凭证服务
type AuthCredentialService struct {
	db             *gorm.DB
	passwordCipher addomain.PasswordCipher // 密码加密器
}

// NewAuthCredentialService 创建授权凭证服务
func NewAuthCredentialService(db *gorm.DB, passwordCipher addomain.PasswordCipher) *AuthCredentialService {
	return &AuthCredentialService{
		db:             db,
		passwordCipher: passwordCipher,
	}
}

// CredentialStatistics 授权凭证统计结果。
type CredentialStatistics struct {
	Total  int64 `json:"total"`
	SSH    int64 `json:"ssh"`    // protocol_type = 'ssh'
	Telnet int64 `json:"telnet"` // protocol_type = 'telnet'
}

// GetStatistics 统计授权凭证总数及 SSH/Telnet 数。用条件聚合避免加载全量行进内存。
func (s *AuthCredentialService) GetStatistics(ctx context.Context) (*CredentialStatistics, error) {
	var result CredentialStatistics
	err := s.db.WithContext(ctx).Model(&models.AuthCredential{}).
		Select(
			"COUNT(*) AS total",
			"SUM(CASE WHEN protocol_type = 'ssh' THEN 1 ELSE 0 END) AS ssh",
			"SUM(CASE WHEN protocol_type = 'telnet' THEN 1 ELSE 0 END) AS telnet",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计授权凭证失败: %w", err)
	}
	return &result, nil
}

// ListRequest 列表请求
type ListCredentialRequest struct {
	base.BaseListRequest
	CredentialName *string
	ProtocolType   *models.ProtocolType // 改为协议类型过滤
}

// credentialAllowedSortFields 授权凭证可排序字段白名单(对应 sys_auth_credential 表列名)。
var credentialAllowedSortFields = map[string]string{
	"credentialName": "credential_name",
	"protocolType":   "protocol_type",
	"username":       "username",
	"createdAt":      "created_at",
}

// List 获取授权凭证列表
func (s *AuthCredentialService) List(ctx context.Context, req *ListCredentialRequest) ([]models.AuthCredential, int64, error) {
	var credentials []models.AuthCredential
	var total int64

	query := s.db.Model(&models.AuthCredential{})

	if req.CredentialName != nil && *req.CredentialName != "" {
		query = query.Where("credential_name LIKE ?", "%"+*req.CredentialName+"%")
	}
	if req.ProtocolType != nil {
		query = query.Where("protocol_type = ?", *req.ProtocolType)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询凭证总数失败: %w", err)
	}

	// 分页查询;用户排序(白名单)优先,无 OrderByColumn 时保留原默认 created_at DESC
	offset := (req.Current - 1) * req.PageSize
	query = base.ApplySort(query, req.BaseListRequest, credentialAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(req.PageSize).Find(&credentials).Error; err != nil {
		return nil, 0, fmt.Errorf("查询凭证列表失败: %w", err)
	}

	// 隐藏密码
	for i := range credentials {
		credentials[i].Password = ""
		credentials[i].EnablePassword = ""
	}

	return credentials, total, nil
}

// CreateRequest 创建请求
type CreateCredentialRequest struct {
	CredentialName  string
	ProtocolType    models.ProtocolType // SSH 或 Telnet
	Username        string
	Password        string
	EnablePassword  string
	SNMPCommunities []string // 多个 SNMP Community
	SNMPVersion     models.SNMPVersion
	Description     string
	IsDefault       bool
	CreatedBy       string
}

// Create 创建授权凭证
func (s *AuthCredentialService) Create(ctx context.Context, req *CreateCredentialRequest) (*models.AuthCredential, error) {
	// 检查凭证名称是否已存在
	var count int64
	if err := s.db.Model(&models.AuthCredential{}).Where("credential_name = ?", req.CredentialName).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查凭证名称失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("凭证名称已存在")
	}

	// 验证凭证配置（新建凭证时密码必填）
	if err := s.validateCredentialConfig(req.ProtocolType, req.Username, req.Password, req.SNMPCommunities, true); err != nil {
		return nil, err
	}

	// 如果设置为默认凭证，取消其他默认凭证
	if req.IsDefault {
		s.db.Model(&models.AuthCredential{}).Where("is_default = ?", true).Update("is_default", false)
	}

	// 使用 SM4 加密密码
	if s.passwordCipher == nil {
		return nil, fmt.Errorf("SM4 加密器未初始化，无法加密密码")
	}

	encryptedPassword, err := s.passwordCipher.Encrypt(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	encryptedEnablePassword, err := s.passwordCipher.Encrypt(req.EnablePassword)
	if err != nil {
		return nil, fmt.Errorf("特权密码加密失败: %w", err)
	}

	credential := models.AuthCredential{
		CredentialName:  req.CredentialName,
		ProtocolType:    req.ProtocolType,
		Username:        req.Username,
		Password:        encryptedPassword,
		EnablePassword:  encryptedEnablePassword,
		SNMPCommunities: req.SNMPCommunities,
		SNMPVersion:     req.SNMPVersion,
		Description:     req.Description,
		IsDefault:       req.IsDefault,
		BaseModel:       models.BaseModel{CreatedBy: req.CreatedBy},
	}

	if err := s.db.Create(&credential).Error; err != nil {
		return nil, fmt.Errorf("创建凭证失败: %w", err)
	}

	// 清除密码后返回
	credential.Password = ""
	credential.EnablePassword = ""

	return &credential, nil
}

// GetByID 根据ID获取凭证
func (s *AuthCredentialService) GetByID(ctx context.Context, id string) (*models.AuthCredential, error) {
	var credential models.AuthCredential
	if err := s.db.Where("id = ?", id).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("查询凭证失败: %w", err)
	}

	// 隐藏密码
	credential.Password = ""
	credential.EnablePassword = ""

	return &credential, nil
}

// GetByIDWithPassword 根据ID获取凭证（包含密码，用于连接设备）
// 注意：此方法已废弃，请使用 GetDecryptedCredential 代替
func (s *AuthCredentialService) GetByIDWithPassword(ctx context.Context, id string) (*models.AuthCredential, error) {
	return s.GetDecryptedCredential(ctx, id)
}

// UpdateRequest 更新请求
type UpdateCredentialRequest struct {
	ID              string
	CredentialName  string
	ProtocolType    models.ProtocolType // SSH 或 Telnet
	Username        string
	Password        string
	EnablePassword  string
	SNMPCommunities []string // 多个 SNMP Community
	SNMPVersion     models.SNMPVersion
	Description     string
	IsDefault       bool
	UpdatedBy       string
}

// Update 更新授权凭证
func (s *AuthCredentialService) Update(ctx context.Context, req *UpdateCredentialRequest) error {
	var credential models.AuthCredential
	if err := s.db.Where("id = ?", req.ID).First(&credential).Error; err != nil {
		return fmt.Errorf("凭证不存在: %w", err)
	}

	// 检查凭证名称是否被其他凭证使用
	var count int64
	if err := s.db.Model(&models.AuthCredential{}).
		Where("credential_name = ? AND id != ?", req.CredentialName, req.ID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查凭证名称失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("凭证名称已存在")
	}

	// 验证凭证配置（更新凭证时用户名必填，密码可选）
	if err := s.validateCredentialConfig(req.ProtocolType, req.Username, req.Password, req.SNMPCommunities, false); err != nil {
		return err
	}

	// 如果设置为默认凭证，取消其他默认凭证
	if req.IsDefault && !credential.IsDefault {
		s.db.Model(&models.AuthCredential{}).Where("is_default = ?", true).Update("is_default", false)
	}

	// 更新字段
	credential.CredentialName = req.CredentialName
	credential.ProtocolType = req.ProtocolType
	credential.Username = req.Username
	credential.SNMPCommunities = req.SNMPCommunities
	credential.SNMPVersion = req.SNMPVersion
	credential.Description = req.Description
	credential.IsDefault = req.IsDefault
	credential.UpdatedBy = req.UpdatedBy

	// 如果提供了新密码，则加密并更新密码
	if req.Password != "" {
		if s.passwordCipher == nil {
			return fmt.Errorf("SM4 加密器未初始化，无法加密密码")
		}
		encrypted, err := s.passwordCipher.Encrypt(req.Password)
		if err != nil {
			return fmt.Errorf("密码加密失败: %w", err)
		}
		credential.Password = encrypted
	}
	if req.EnablePassword != "" {
		if s.passwordCipher == nil {
			return fmt.Errorf("SM4 加密器未初始化，无法加密特权密码")
		}
		encrypted, err := s.passwordCipher.Encrypt(req.EnablePassword)
		if err != nil {
			return fmt.Errorf("特权密码加密失败: %w", err)
		}
		credential.EnablePassword = encrypted
	}

	if err := s.db.Save(&credential).Error; err != nil {
		return fmt.Errorf("更新凭证失败: %w", err)
	}

	return nil
}

// Delete 删除授权凭证
func (s *AuthCredentialService) Delete(ctx context.Context, id string) error {
	// 检查是否有设备正在使用此凭证
	var count int64
	if err := s.db.Model(&models.NetworkDevice{}).Where("credential_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查凭证使用情况失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("有 %d 个设备正在使用此凭证，无法删除", count)
	}

	if err := s.db.Where("id = ?", id).Delete(&models.AuthCredential{}).Error; err != nil {
		return fmt.Errorf("删除凭证失败: %w", err)
	}

	return nil
}

// BatchDelete 批量删除凭证
func (s *AuthCredentialService) BatchDelete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// GetDefaultCredential 获取默认凭证
func (s *AuthCredentialService) GetDefaultCredential(ctx context.Context) (*models.AuthCredential, error) {
	var credential models.AuthCredential
	if err := s.db.Where("is_default = ?", true).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("未找到默认凭证")
	}

	// 隐藏密码
	credential.Password = ""
	credential.EnablePassword = ""

	return &credential, nil
}

// SetDefaultCredential 设置默认凭证
func (s *AuthCredentialService) SetDefaultCredential(ctx context.Context, id string, updatedBy string) error {
	// 取消所有默认凭证
	if err := s.db.Model(&models.AuthCredential{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return fmt.Errorf("更新凭证失败: %w", err)
	}

	// 设置新的默认凭证
	if err := s.db.Model(&models.AuthCredential{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_default": true,
		"updated_by": updatedBy,
	}).Error; err != nil {
		return fmt.Errorf("设置默认凭证失败: %w", err)
	}

	return nil
}

// GetDevicesByCredential 获取使用指定凭证的设备列表
func (s *AuthCredentialService) GetDevicesByCredential(ctx context.Context, credentialID string) ([]models.NetworkDevice, error) {
	var devices []models.NetworkDevice
	if err := s.db.Where("credential_id = ?", credentialID).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}
	return devices, nil
}

// validateCredentialConfig 验证凭证配置是否完整
// isNewCredential: 是否为新凭证（新建时密码必填，更新时可为空）
func (s *AuthCredentialService) validateCredentialConfig(_ models.ProtocolType, username, password string, _ []string, isNewCredential bool) error {
	// SSH/Telnet 配置验证
	// 新建凭证时：用户名和密码都必须提供
	// 更新凭证时：至少需要用户名或密码其一（允许保留原密码）
	if isNewCredential {
		if username == "" {
			return fmt.Errorf("请输入用户名")
		}
		if password == "" {
			return fmt.Errorf("请输入密码")
		}
	} else {
		// 更新时，用户名必须提供，密码可选
		if username == "" {
			return fmt.Errorf("请输入用户名")
		}
		// 密码为空表示不修改，这是允许的
	}

	// SNMP 配置验证 - 至少需要一个 community（允许为空用于仅SSH凭证）
	// 注意：有些场景下凭证可能只用于 SSH 连接，不需要 SNMP
	return nil
}

// ValidateCredential 验证凭证配置是否完整（保留兼容性）
func (s *AuthCredentialService) ValidateCredential(credential *models.AuthCredential) error {
	return s.validateCredentialConfig(credential.ProtocolType, credential.Username, credential.Password, credential.SNMPCommunities, true)
}

// GetDecryptedCredential 获取解密后的凭证（用于设备连接）
// 严格模式：解密失败则拒绝使用（不支持明文密码）
func (s *AuthCredentialService) GetDecryptedCredential(ctx context.Context, id string) (*models.AuthCredential, error) {
	var credential models.AuthCredential
	if err := s.db.Where("id = ?", id).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("查询凭证失败: %w", err)
	}

	// 解密密码（严格模式，解密失败则拒绝）
	if s.passwordCipher != nil && credential.Password != "" {
		decrypted, err := s.passwordCipher.Decrypt(credential.Password)
		if err != nil {
			return nil, fmt.Errorf("密码解密失败，请重新设置凭证密码")
		}
		credential.Password = decrypted
	}
	if s.passwordCipher != nil && credential.EnablePassword != "" {
		decrypted, err := s.passwordCipher.Decrypt(credential.EnablePassword)
		if err != nil {
			return nil, fmt.Errorf("特权密码解密失败，请重新设置凭证密码")
		}
		credential.EnablePassword = decrypted
	}

	return &credential, nil
}
