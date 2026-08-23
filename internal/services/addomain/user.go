package addomain

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// UserService 用户服务
type UserService struct {
	db   *gorm.DB
	pool AccountPool // Phase 38 Wave 1 DI 脚手架（38-02 将用于 FailoverClient 闭包改造）
}

// NewUserService 创建用户服务
// Phase 38 Wave 1: 注入 AccountPool 字段（38-02 将用于 FailoverClient 闭包改造）。
func NewUserService(db *gorm.DB, pool AccountPool) *UserService {
	return &UserService{db: db, pool: pool}
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
		Where("username NOT LIKE ?", "$DUPLICATE-%").
		Where("username NOT LIKE ?", "%$") // 过滤计算机账号（以$结尾）

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

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		return client.UpdateUserAttribute(userDN, attrs)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
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

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		return client.EnableUser(userDN)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
		Update("is_enabled", true)

	return nil
}

// Disable 禁用用户
func (s *UserService) Disable(ctx context.Context, config *models.ADConfig, userDN string) error {

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		return client.DisableUser(userDN)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
		Update("is_enabled", false)

	return nil
}

// Move 移动用户到其他OU
func (s *UserService) Move(ctx context.Context, config *models.ADConfig, userDN, newOUDN string) error {

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		return client.MoveUser(userDN, newOUDN)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
		Update("ou_dn", newOUDN)

	return nil
}

// GetUserIds 获取用户ID列表（用于全选功能）
func (s *UserService) GetUserIds(ctx context.Context, req *UserListRequest) ([]string, error) {
	var userIds []string

	query := s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", req.ConfigID).
		Where("username NOT LIKE ?", "$DUPLICATE-%").
		Where("username NOT LIKE ?", "%$") // 过滤计算机账号（以$结尾）

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

	err := query.Pluck("id", &userIds).Error
	if err != nil {
		return nil, fmt.Errorf("查询用户ID列表失败: %w", err)
	}

	return userIds, nil
}
