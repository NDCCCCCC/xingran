package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ==================== 用户查询上下文 ====================

// UserNoticeQueryContext 用户通知查询上下文
// 封装用户信息、角色信息和基础查询
type UserNoticeQueryContext struct {
	User    models.User
	RoleIDs []string
	Query   *gorm.DB
}

// buildUserVisibleQuery 构建用户可见通知的基础查询
// 封装了获取用户信息、角色信息以及构建权限过滤查询的公共逻辑
func (s *NoticeService) buildUserVisibleQuery(ctx context.Context, userID string) (*UserNoticeQueryContext, error) {
	// 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 获取用户角色
	var userRoles []models.UserRole
	s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles)
	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// 构建权限过滤查询：已发布 + 正常状态 + 未被忽略 + 匹配用户部门/角色
	// （publish_status=发布态 E 簇，status=Notice 启停字段，两类语义不可互换）
	query := s.db.WithContext(ctx).Model(&models.Notice{}).
		Where("publish_status = ? AND status = ?", models.PublishStatusPublished, models.NoticeStatusNormal).
		Where("id NOT IN (SELECT notice_id FROM sys_notice_ignore WHERE user_id = ?)", userID).
		Where("(target_type = 0) OR (target_type = 3 AND id IN (SELECT notice_id FROM sys_notice_target WHERE target_type = 'user' AND target_id = ?)) OR "+
			"(target_type = 1 AND id IN (SELECT notice_id FROM sys_notice_target WHERE target_type = 'dept' AND target_id = ?)) OR "+
			"(target_type = 2 AND id IN (SELECT notice_id FROM sys_notice_target WHERE target_type = 'role' AND target_id IN (?)))",
			userID, user.DeptID, roleIDs)

	return &UserNoticeQueryContext{
		User:    user,
		RoleIDs: roleIDs,
		Query:   query,
	}, nil
}
