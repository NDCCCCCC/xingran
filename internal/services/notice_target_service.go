package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ==================== 目标用户相关 ====================

// GetTargetUsers 获取通知的目标用户列表
func (s *NoticeService) GetTargetUsers(ctx context.Context, notice *models.Notice) ([]string, error) {
	var userIDs []string

	// 处理 nil Targets 的情况
	if notice.Targets == nil {
		notice.Targets = []models.NoticeTarget{}
	}

	switch notice.TargetType {
	case models.TargetAll:
		// 限制最大返回数量，避免大数据量影响性能
		const maxUsers = 10000
		query := s.db.WithContext(ctx).Model(&models.User{}).Limit(maxUsers)
		if err := query.Pluck("id", &userIDs).Error; err != nil {
			return nil, fmt.Errorf("获取用户列表失败: %w", err)
		}

	case models.TargetDept:
		var deptIDs []string
		for _, t := range notice.Targets {
			if t.TargetType == "dept" {
				deptIDs = append(deptIDs, t.TargetID)
			}
		}
		if len(deptIDs) > 0 {
			// 添加限制避免部门用户过多
			const maxUsers = 10000
			query := s.db.WithContext(ctx).Model(&models.User{}).Where("dept_id IN ?", deptIDs).Limit(maxUsers)
			if err := query.Pluck("id", &userIDs).Error; err != nil {
				return nil, fmt.Errorf("获取部门用户失败: %w", err)
			}
		}

	case models.TargetRole:
		roleIDs := getTargetIDsByType(notice.Targets, "role")
		if len(roleIDs) > 0 {
			// 添加限制避免角色用户过多
			const maxUsers = 10000
			query := s.db.WithContext(ctx).Table("sys_user_role").
				Where("role_id IN ?", roleIDs).
				Distinct("user_id").
				Limit(maxUsers).
				Pluck("user_id", &userIDs)
			if err := query.Error; err != nil {
				return nil, fmt.Errorf("获取角色用户失败: %w", err)
			}
		}

	case models.TargetUser:
		userIDs = getTargetIDsByType(notice.Targets, "user")

	default:
		// 未知的 TargetType，返回空列表
		return []string{}, nil
	}

	return unique(userIDs), nil
}

// buildTargets 构建接收范围
func (s *NoticeService) buildTargets(noticeID string, req *CreateNoticeRequest) []models.NoticeTarget {
	var targets []models.NoticeTarget

	switch req.TargetType {
	case models.TargetDept:
		for _, deptID := range req.TargetDepts {
			targets = append(targets, models.NoticeTarget{
				NoticeID:   noticeID,
				TargetType: "dept",
				TargetID:   deptID,
			})
			// 包含子部门
			childDepts := s.getChildDeptIDs(deptID)
			for _, childID := range childDepts {
				targets = append(targets, models.NoticeTarget{
					NoticeID:   noticeID,
					TargetType: "dept",
					TargetID:   childID,
				})
			}
		}
	case models.TargetRole:
		for _, roleID := range req.TargetRoles {
			targets = append(targets, models.NoticeTarget{
				NoticeID:   noticeID,
				TargetType: "role",
				TargetID:   roleID,
			})
		}
	case models.TargetUser:
		for _, userID := range req.TargetUsers {
			targets = append(targets, models.NoticeTarget{
				NoticeID:   noticeID,
				TargetType: "user",
				TargetID:   userID,
			})
		}
	}

	return targets
}

// getChildDeptIDs 递归获取子部门ID
func (s *NoticeService) getChildDeptIDs(deptID string) []string {
	var deptIDs []string
	s.db.Raw(`
		WITH RECURSIVE dept_tree AS (
			SELECT id FROM sys_department WHERE id = ?
			UNION ALL
			SELECT d.id FROM sys_department d
			INNER JOIN dept_tree dt ON d.parent_id = dt.id
		)
		SELECT id FROM dept_tree WHERE id != ?
	`, deptID, deptID).Scan(&deptIDs)
	return deptIDs
}

// getTargetIDsByType 根据类型获取目标ID列表
func getTargetIDsByType(targets []models.NoticeTarget, targetType string) []string {
	var ids []string
	for _, t := range targets {
		if t.TargetType == targetType {
			ids = append(ids, t.TargetID)
		}
	}
	return ids
}

// unique 去重
func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if !keys[entry] {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// GetNoticeStatistics 获取阅读统计
func (s *NoticeService) GetNoticeStatistics(ctx context.Context, noticeID string) (*models.NoticeStatistics, error) {
	var stats models.NoticeStatistics

	// 获取通知详情
	var notice models.Notice
	if err := s.db.WithContext(ctx).Where("id = ?", noticeID).First(&notice).Error; err != nil {
		return nil, fmt.Errorf("查询通知失败: %w", err)
	}

	// 获取目标用户列表
	targetUserIDs, err := s.GetTargetUsers(ctx, &notice)
	if err != nil {
		return nil, fmt.Errorf("获取目标用户失败: %w", err)
	}

	totalTargets := len(targetUserIDs)

	// 获取已读数量
	var readCount int64
	if totalTargets > 0 {
		s.db.WithContext(ctx).Table("sys_notice_read").
			Where("notice_id = ? AND user_id IN ?", noticeID, targetUserIDs).
			Count(&readCount)
	}

	stats.ReadCount = int(readCount)
	stats.TotalTargets = totalTargets
	stats.UnreadCount = totalTargets - int(readCount)
	if totalTargets > 0 {
		stats.ReadRate = float64(readCount) / float64(totalTargets) * 100
	} else {
		stats.ReadRate = 0
	}

	return &stats, nil
}
