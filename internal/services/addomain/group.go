package addomain

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// GroupService 用户组服务
type GroupService struct {
	db   *gorm.DB
	pool AccountPool // Phase 38 Wave 1 DI 脚手架（38-02 将用于 FailoverClient 闭包改造）
}

// NewGroupService 创建用户组服务
// Phase 38 Wave 1: 注入 AccountPool 字段（38-02 将用于 FailoverClient 闭包改造）。
func NewGroupService(db *gorm.DB, pool AccountPool) *GroupService {
	return &GroupService{db: db, pool: pool}
}

// adGroupAllowedSortFields AD用户组可排序字段白名单(对应 sys_ad_group 表列名)。
var adGroupAllowedSortFields = map[string]string{
	"groupName":    "group_name",
	"memberCount":  "member_count",
	"ouDn":         "ou_dn",
}

// GroupListRequest 用户组列表请求
type GroupListRequest struct {
	base.BaseListRequest
	ConfigID  string  `json:"configId" binding:"required"`
	OUDN      *string `json:"ouDn,omitempty"`
	GroupName *string `json:"groupName,omitempty"`
}

// GetList 获取用户组列表
func (s *GroupService) GetList(ctx context.Context, req *GroupListRequest) ([]models.ADGroup, int64, error) {
	var groups []models.ADGroup
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", req.ConfigID)

	if req.OUDN != nil && *req.OUDN != "" {
		// 选择父OU时包含所有子OU: ou_dn = '选中的OU' OR ou_dn LIKE '%,选中的OU'
		query = query.Where("ou_dn = ? OR ou_dn LIKE ?", *req.OUDN, "%,"+*req.OUDN)
	}

	if req.GroupName != nil {
		query = query.Where("group_name LIKE ?", "%"+*req.GroupName+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	offset := (req.Current - 1) * req.PageSize
	query = base.ApplySort(query, req.BaseListRequest, adGroupAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("group_name ASC")
	}
	err := query.Offset(offset).Limit(req.PageSize).Find(&groups).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询列表失败: %w", err)
	}

	return groups, total, nil
}

// GetByDN 根据DN获取用户组
func (s *GroupService) GetByDN(ctx context.Context, configID, groupDN string) (*models.ADGroup, error) {
	var group models.ADGroup
	err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND group_dn = ? AND deleted_at IS NULL", configID, groupDN).
		First(&group).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户组不存在")
		}
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	return &group, nil
}

// GetMembers 获取用户组成员
func (s *GroupService) GetMembers(ctx context.Context, configID, groupDN string, current, pageSize int) ([]models.ADUser, int64, error) {
	var members []models.ADGroupMember
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADGroupMember{}).
		Where("ad_config_id = ? AND group_dn = ?", configID, groupDN)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询成员总数失败: %w", err)
	}

	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&members).Error; err != nil {
		return nil, 0, fmt.Errorf("查询成员失败: %w", err)
	}

	// 根据用户DN获取用户详情
	var userDNs []string
	for _, m := range members {
		userDNs = append(userDNs, m.UserDN)
	}

	var users []models.ADUser
	if len(userDNs) > 0 {
		if err := s.db.WithContext(ctx).
			Where("ad_config_id = ? AND user_dn IN ? AND deleted_at IS NULL", configID, userDNs).
			Find(&users).Error; err != nil {
			return nil, 0, fmt.Errorf("查询用户详情失败: %w", err)
		}
	}

	return users, total, nil
}

// AddMember 添加用户组成员
func (s *GroupService) AddMember(ctx context.Context, config *models.ADConfig, groupDN, userDN string) error {

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		return client.AddGroupMember(groupDN, userDN)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	// 更新本地成员关系
	member := &models.ADGroupMember{
		ADConfigID: config.ID,
		GroupDN:    groupDN,
		UserDN:     userDN,
	}
	s.db.WithContext(ctx).Create(member)

	// 更新成员数
	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		UpdateColumn("member_count", gorm.Expr("member_count + 1"))

	return nil
}

// RemoveMember 移除用户组成员
func (s *GroupService) RemoveMember(ctx context.Context, config *models.ADConfig, groupDN, userDN string) error {

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		return client.RemoveGroupMember(groupDN, userDN)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	// 删除本地成员关系
	s.db.WithContext(ctx).Where("ad_config_id = ? AND group_dn = ? AND user_dn = ?", config.ID, groupDN, userDN).
		Delete(&models.ADGroupMember{})

	// 更新成员数
	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		UpdateColumn("member_count", gorm.Expr("member_count - 1"))

	return nil
}

// GroupUpdateRequest 用户组更新请求
type GroupUpdateRequest struct {
	GroupName   *string `json:"groupName,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Update 更新用户组
func (s *GroupService) Update(ctx context.Context, config *models.ADConfig, groupDN string, req *GroupUpdateRequest) error {

	// 构建更新属性
	attrs := make(map[string]string)
	if req.GroupName != nil {
		attrs["cn"] = *req.GroupName
	}
	if req.Description != nil {
		attrs["description"] = *req.Description
	}

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		return client.UpdateGroupAttribute(groupDN, attrs)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	// 更新本地缓存
	updateData := make(map[string]interface{})
	if req.GroupName != nil {
		updateData["group_name"] = *req.GroupName
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}

	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		Updates(updateData)

	return nil
}
