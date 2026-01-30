package addomain

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// GroupService 用户组服务
type GroupService struct {
	db *gorm.DB
}

// NewGroupService 创建用户组服务
func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{db: db}
}

// GroupListRequest 用户组列表请求
type GroupListRequest struct {
	ConfigID  string  `json:"configId" binding:"required"`
	OUDN      *string `json:"ouDn,omitempty"`
	GroupName *string `json:"groupName,omitempty"`
	Current   int     `json:"current"`
	PageSize  int     `json:"pageSize"`
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
	err := query.Offset(offset).Limit(req.PageSize).Order("group_name ASC").Find(&groups).Error
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
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.AddGroupMember(groupDN, userDN); err != nil {
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
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.RemoveGroupMember(groupDN, userDN); err != nil {
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
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	// 构建更新属性
	attrs := make(map[string]string)
	if req.GroupName != nil {
		attrs["cn"] = *req.GroupName
	}
	if req.Description != nil {
		attrs["description"] = *req.Description
	}

	if err := client.UpdateGroupAttribute(groupDN, attrs); err != nil {
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
