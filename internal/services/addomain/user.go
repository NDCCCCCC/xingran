package addomain

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// UserService 用户服务
type UserService struct {
	db *gorm.DB
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// adUserAllowedSortFields AD用户可排序字段白名单(对应 sys_ad_user 表列名)。
var adUserAllowedSortFields = map[string]string{
	"username":    "username",
	"displayName": "display_name",
	"email":       "email",
	"phone":       "phone",
	"ouDn":        "ou_dn",
	"isEnabled":   "is_enabled",
}

// UserListRequest 用户列表请求
type UserListRequest struct {
	base.BaseListRequest
	ConfigID  string  `json:"configId" binding:"required"`
	OUDN      *string `json:"ouDn,omitempty"`
	Username  *string `json:"username,omitempty"`
	IsEnabled *bool   `json:"isEnabled,omitempty"`
}

// GetList 获取用户列表
func (s *UserService) GetList(ctx context.Context, req *UserListRequest) ([]models.ADUser, int64, error) {
	var users []models.ADUser
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", req.ConfigID).
		Where("username NOT LIKE ?", "$DUPLICATE-%")

	if req.OUDN != nil {
		// 选择父OU时包含所有子OU: ou_dn = '选中的OU' OR ou_dn LIKE '%,选中的OU'
		query = query.Where("ou_dn = ? OR ou_dn LIKE ?", *req.OUDN, "%,"+*req.OUDN)
	}

	if req.Username != nil {
		query = query.Where("username LIKE ?", "%"+*req.Username+"%")
	}

	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	offset := (req.Current - 1) * req.PageSize
	query = base.ApplySort(query, req.BaseListRequest, adUserAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("username ASC")
	}
	err := query.Offset(offset).Limit(req.PageSize).Find(&users).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询列表失败: %w", err)
	}

	return users, total, nil
}

// GetByDN 根据DN获取用户
func (s *UserService) GetByDN(ctx context.Context, configID, userDN string) (*models.ADUser, error) {
	var user models.ADUser
	err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND user_dn = ? AND deleted_at IS NULL", configID, userDN).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	return &user, nil
}

// GetByID 根据数据库ID获取用户
func (s *UserService) GetByID(ctx context.Context, userID string) (*models.ADUser, error) {
	var user models.ADUser
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	return &user, nil
}

// UserUpdateRequest 用户更新请求
type UserUpdateRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Mobile      *string `json:"mobile,omitempty"`
	Title       *string `json:"title,omitempty"`
	Department  *string `json:"department,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Update 更新用户
func (s *UserService) Update(ctx context.Context, config *models.ADConfig, userDN string, req *UserUpdateRequest) error {
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer client.Close()

	// 构建更新属性
	attrs := make(map[string]string)
	if req.DisplayName != nil {
		attrs["displayName"] = *req.DisplayName
	}
	if req.Email != nil {
		attrs["mail"] = *req.Email
	}
	if req.Phone != nil {
		attrs["telephoneNumber"] = *req.Phone
	}
	if req.Mobile != nil {
		attrs["mobile"] = *req.Mobile
	}
	if req.Title != nil {
		attrs["title"] = *req.Title
	}
	if req.Department != nil {
		attrs["department"] = *req.Department
	}
	if req.Description != nil {
		attrs["description"] = *req.Description
	}

	if err := client.UpdateUserAttribute(userDN, attrs); err != nil {
		return err
	}

	// 更新本地缓存
	updateData := make(map[string]interface{})
	if req.DisplayName != nil {
		updateData["display_name"] = *req.DisplayName
	}
	if req.Email != nil {
		updateData["email"] = *req.Email
	}
	if req.Phone != nil {
		updateData["phone"] = *req.Phone
	}
	if req.Mobile != nil {
		updateData["mobile"] = *req.Mobile
	}
	if req.Title != nil {
		updateData["title"] = *req.Title
	}
	if req.Department != nil {
		updateData["department"] = *req.Department
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
		Updates(updateData)

	return nil
}

// Enable 启用用户
func (s *UserService) Enable(ctx context.Context, config *models.ADConfig, userDN string) error {
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.EnableUser(userDN); err != nil {
		return err
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
		Update("is_enabled", true)

	return nil
}

// Disable 禁用用户
func (s *UserService) Disable(ctx context.Context, config *models.ADConfig, userDN string) error {
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.DisableUser(userDN); err != nil {
		return err
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
		Update("is_enabled", false)

	return nil
}

// Move 移动用户到其他OU
func (s *UserService) Move(ctx context.Context, config *models.ADConfig, userDN, newOUDN string) error {
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.MoveUser(userDN, newOUDN); err != nil {
		return err
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
		Update("ou_dn", newOUDN)

	return nil
}
