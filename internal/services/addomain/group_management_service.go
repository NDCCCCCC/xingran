package addomain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// GroupManagementService AD组管理服务接口
type GroupManagementService interface {
	// CreateGroupForDept 为部门创建AD组（cxhub-{dept}命名）
	CreateGroupForDept(ctx context.Context, deptID, configID, parentOUDN string) (*models.ADGroup, error)

	// DeleteGroup 删除AD组
	DeleteGroup(ctx context.Context, groupID string) error

	// AddMembers 批量添加组成员
	AddMembers(ctx context.Context, groupID string, userIDs []string) (*MemberChangeResult, error)

	// RemoveMembers 批量移除组成员
	RemoveMembers(ctx context.Context, groupID string, userIDs []string) (*MemberChangeResult, error)

	// BulkCreateGroupsForDepts 批量为多个部门创建组
	BulkCreateGroupsForDepts(ctx context.Context, deptIDs []string, configID, parentOUDN string) (*BulkCreateResult, error)
}

// groupManagementService 服务实现
type groupManagementService struct {
	db   *gorm.DB
	pool AccountPool // Phase 38 Wave 1 DI 脚手架（38-02 将用于 FailoverClient 闭包改造）
}

// NewGroupManagementService 创建服务实例
// Phase 38 Wave 1: 注入 AccountPool 字段（38-02 将用于 FailoverClient 闭包改造）。
func NewGroupManagementService(db *gorm.DB, pool AccountPool) GroupManagementService {
	return &groupManagementService{db: db, pool: pool}
}

// MemberChangeResult 成员变更结果
type MemberChangeResult struct {
	GroupID      string `json:"groupId"`
	GroupName    string `json:"groupName"`
	AddedCount   int    `json:"addedCount"`
	RemovedCount int    `json:"removedCount"`
	FailedCount  int    `json:"failedCount"`
}

// BulkCreateResult 批量创建结果
type BulkCreateResult struct {
	TotalCount   int      `json:"totalCount"`
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedDepts  []string `json:"failedDepts,omitempty"`
}

// CreateGroupForDept 为部门创建AD组
func (s *groupManagementService) CreateGroupForDept(ctx context.Context, deptID, configID, parentOUDN string) (*models.ADGroup, error) {
	// 1. 获取部门信息
	var dept models.Department
	if err := s.db.WithContext(ctx).Where("id = ?", deptID).First(&dept).Error; err != nil {
		return nil, fmt.Errorf("部门不存在")
	}

	// 2. 获取AD配置
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在")
	}

	// 3. 构造组DN和组名
	groupName := fmt.Sprintf("cxhub-%s", strings.TrimSuffix(dept.DeptName, "部"))
	if parentOUDN == "" {
		// 使用配置的member_ou_dn，如果为空则使用baseDN
		parentOUDN = config.MemberOUDN
		if parentOUDN == "" {
			parentOUDN = config.BaseDN
		}
	}
	groupDN := fmt.Sprintf("CN=%s,%s", groupName, parentOUDN)

	// 4. 连接LDAP并创建组（Phase 38 Wave 1: 改走 FailoverClient 账号池故障切换）
	fc := NewFailoverClient(s.pool, &config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		// 检查组是否已存在
		if err := client.CreateGroup(groupDN, groupName, dept.DeptName+"部门组", 0); err != nil {
			// 判断是否为"已存在"错误
			if strings.Contains(err.Error(), "68") || strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("AD组已存在: %s", groupName)
			}
			return fmt.Errorf("创建AD组失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return nil, err
	}

	applogger.Infof("[组管理] 创建AD组成功: %s (%s)", groupName, groupDN)

	// 5. 创建本地数据库记录
	now := time.Now()
	group := &models.ADGroup{
		ADConfigID:  configID,
		GroupDN:     groupDN,
		GroupName:   groupName,
		Description: dept.DeptName + "部门组",
		GroupScope:  models.ADGroupScopeGlobal,
		GroupType:   models.ADGroupTypeSecurity,
		OUN:         parentOUDN,
		MemberCount: 0,
		LastSyncAt:  &now,
	}

	if err := s.db.WithContext(ctx).Create(group).Error; err != nil {
		applogger.Errorf("[组管理] 创建本地组记录失败: %v", err)
		// 组已在AD创建，但本地记录失败，记录错误但不返回失败
	}

	return group, nil
}

// DeleteGroup 删除AD组
func (s *groupManagementService) DeleteGroup(ctx context.Context, groupID string) error {
	// 1. 获取组信息
	var group models.ADGroup
	if err := s.db.WithContext(ctx).Where("id = ?", groupID).First(&group).Error; err != nil {
		return fmt.Errorf("AD组不存在")
	}

	// 2. 检查是否有成员
	if group.MemberCount > 0 {
		return fmt.Errorf("组中有成员，无法删除。请先移除所有成员")
	}

	// 3. 检查是否有OU映射关系
	var mappingCount int64
	s.db.WithContext(ctx).Model(&models.OUGroupMapping{}).
		Where("ad_group_id = ? AND deleted_at IS NULL", groupID).
		Count(&mappingCount)
	if mappingCount > 0 {
		return fmt.Errorf("存在OU映射关系，无法删除。请先删除映射")
	}

	// 4. 连接LDAP并删除组
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", group.ADConfigID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	fc := NewFailoverClient(s.pool, &config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		if err := client.DeleteGroup(group.GroupDN); err != nil {
			return fmt.Errorf("删除AD组失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	applogger.Infof("[组管理] 删除AD组成功: %s", group.GroupName)

	// 5. 软删除本地记录
	if err := s.db.WithContext(ctx).Delete(&group).Error; err != nil {
		applogger.Errorf("[组管理] 删除本地组记录失败: %v", err)
	}

	return nil
}

// AddMembers 批量添加组成员
func (s *groupManagementService) AddMembers(ctx context.Context, groupID string, userIDs []string) (*MemberChangeResult, error) {
	result := &MemberChangeResult{GroupID: groupID}

	// 1. 获取组信息
	var group models.ADGroup
	if err := s.db.WithContext(ctx).Where("id = ?", groupID).First(&group).Error; err != nil {
		return nil, fmt.Errorf("AD组不存在")
	}
	result.GroupName = group.GroupName

	// 2. 获取用户信息
	var users []models.User
	if err := s.db.WithContext(ctx).Where("id IN ? AND ad_dn IS NOT NULL AND ad_dn != ''", userIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败")
	}

	if len(users) == 0 {
		return result, fmt.Errorf("没有有效的AD用户")
	}

	// 3. 连接LDAP
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", group.ADConfigID).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在")
	}


	// 4. 批量添加成员（Phase 38 Wave 1: 改走 FailoverClient 账号池故障切换）
	userDNs := make([]string, 0, len(users))
	for _, user := range users {
		if user.AdDn != nil {
			userDNs = append(userDNs, *user.AdDn)
		}
	}

	fc := NewFailoverClient(s.pool, &config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		if err := client.AddGroupMembers(group.GroupDN, userDNs); err != nil {
			return fmt.Errorf("添加组成员失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return nil, err
	}

	result.AddedCount = len(userDNs)
	result.FailedCount = len(userIDs) - len(userDNs)

	// 5. 更新本地成员数
	group.MemberCount += result.AddedCount
	s.db.WithContext(ctx).Model(&group).Update("member_count", group.MemberCount)

	applogger.Infof("[组管理] 添加组成员成功: %s, 添加=%d, 跳过=%d", group.GroupName, result.AddedCount, result.FailedCount)

	return result, nil
}

// RemoveMembers 批量移除组成员
func (s *groupManagementService) RemoveMembers(ctx context.Context, groupID string, userIDs []string) (*MemberChangeResult, error) {
	result := &MemberChangeResult{GroupID: groupID}

	// 1. 获取组信息
	var group models.ADGroup
	if err := s.db.WithContext(ctx).Where("id = ?", groupID).First(&group).Error; err != nil {
		return nil, fmt.Errorf("AD组不存在")
	}
	result.GroupName = group.GroupName

	// 2. 获取用户信息
	var users []models.User
	if err := s.db.WithContext(ctx).Where("id IN ? AND ad_dn IS NOT NULL AND ad_dn != ''", userIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败")
	}

	if len(users) == 0 {
		return result, fmt.Errorf("没有有效的AD用户")
	}

	// 3. 连接LDAP
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", group.ADConfigID).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在")
	}


	// 4. 批量移除成员（Phase 38 Wave 1: 改走 FailoverClient 账号池故障切换）
	userDNs := make([]string, 0, len(users))
	for _, user := range users {
		if user.AdDn != nil {
			userDNs = append(userDNs, *user.AdDn)
		}
	}

	fc := NewFailoverClient(s.pool, &config)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		if err := client.RemoveGroupMembers(group.GroupDN, userDNs); err != nil {
			return fmt.Errorf("移除组成员失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return nil, err
	}

	result.RemovedCount = len(userDNs)
	result.FailedCount = len(userIDs) - len(userDNs)

	// 5. 更新本地成员数
	group.MemberCount -= result.RemovedCount
	if group.MemberCount < 0 {
		group.MemberCount = 0
	}
	s.db.WithContext(ctx).Model(&group).Update("member_count", group.MemberCount)

	applogger.Infof("[组管理] 移除组成员成功: %s, 移除=%d, 跳过=%d", group.GroupName, result.RemovedCount, result.FailedCount)

	return result, nil
}

// BulkCreateGroupsForDepts 批量为多个部门创建组
func (s *groupManagementService) BulkCreateGroupsForDepts(ctx context.Context, deptIDs []string, configID, parentOUDN string) (*BulkCreateResult, error) {
	result := &BulkCreateResult{
		TotalCount:  len(deptIDs),
		FailedDepts: []string{},
	}

	for _, deptID := range deptIDs {
		group, err := s.CreateGroupForDept(ctx, deptID, configID, parentOUDN)
		if err != nil {
			result.FailedCount++
			// 获取部门名用于错误报告
			var dept models.Department
			s.db.WithContext(ctx).Where("id = ?", deptID).Pluck("dept_name", &dept.DeptName)
			result.FailedDepts = append(result.FailedDepts, fmt.Sprintf("%s: %s", dept.DeptName, err.Error()))
		} else if group != nil {
			result.SuccessCount++
		}
	}

	applogger.Infof("[组管理] 批量创建组完成: 总数=%d, 成功=%d, 失败=%d",
		result.TotalCount, result.SuccessCount, result.FailedCount)

	return result, nil
}
