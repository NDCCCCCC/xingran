package services

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// DutyPoolService 值班池管理服务
type DutyPoolService struct {
	db *gorm.DB
}

// NewDutyPoolService 创建值班池管理服务
func NewDutyPoolService(db *gorm.DB) *DutyPoolService {
	return &DutyPoolService{db: db}
}

// CreateDutyPool 创建值班池
func (s *DutyPoolService) CreateDutyPool(ctx context.Context, req *DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error) {
	var pool *models.DutyPool

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查值班池名称是否已存在
		var count int64
		if err := tx.Model(&models.DutyPool{}).Where("pool_name = ?", req.PoolName).Count(&count).Error; err != nil {
			return fmt.Errorf("检查值班池名称失败: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("值班池名称已存在")
		}

		// 创建值班池
		newPool := &models.DutyPool{
			BaseModel:   models.BaseModel{CreatedBy: creatorID},
			PoolName:    req.PoolName,
			DeptID:      req.DeptID,
			Description: req.Description,
			Status:      models.DutyPoolStatusEnabled,
			DailyCount:  req.DailyCount,
		}

		if err := tx.Create(newPool).Error; err != nil {
			return fmt.Errorf("创建值班池失败: %w", err)
		}

		// 添加成员 - 使用批量查询验证用户
		if len(req.MemberIDs) > 0 {
			// 一次性查询所有用户是否存在
			var existingUsers []models.User
			if err := tx.Where("id IN ?", req.MemberIDs).Find(&existingUsers).Error; err != nil {
				return fmt.Errorf("验证用户失败: %w", err)
			}

			// 构建已存在用户的ID集合
			existingUserIDs := make(map[string]bool)
			for _, user := range existingUsers {
				existingUserIDs[user.ID] = true
			}

			// 验证所有用户都存在
			for _, userID := range req.MemberIDs {
				if !existingUserIDs[userID] {
					return fmt.Errorf("用户不存在: %s", userID)
				}
			}

			// 创建成员记录
			var members []models.DutyPoolMember
			for i, userID := range req.MemberIDs {
				members = append(members, models.DutyPoolMember{
					PoolID:      newPool.ID,
					UserID:      userID,
					MemberOrder: i,
				})
			}

			if err := tx.Create(&members).Error; err != nil {
				return fmt.Errorf("添加值班池成员失败: %w", err)
			}
		}

		pool = newPool
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新加载完整的值班池信息
	if err := s.db.WithContext(ctx).Preload("Members.User").Preload("Dept").Where("id = ?", pool.ID).First(pool).Error; err != nil {
		return nil, fmt.Errorf("查询值班池失败: %w", err)
	}

	return pool, nil
}

// GetDutyPoolList 获取值班池列表
func (s *DutyPoolService) GetDutyPoolList(ctx context.Context, req *DutyPoolListRequest) ([]models.DutyPool, int64, error) {
	var pools []models.DutyPool
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DutyPool{})

	// 筛选条件
	if req.PoolName != nil && *req.PoolName != "" {
		query = query.Where("pool_name LIKE ?", "%"+*req.PoolName+"%")
	}
	if req.DeptID != nil && *req.DeptID != "" {
		query = query.Where("dept_id = ?", *req.DeptID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计值班池数量失败: %w", err)
	}

	// 分页查询
	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	offset := (req.Current - 1) * req.PageSize
	if err := query.Preload("Members.User").Preload("Dept").
		Order("created_at DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&pools).Error; err != nil {
		return nil, 0, fmt.Errorf("查询值班池列表失败: %w", err)
	}

	return pools, total, nil
}

// GetDutyPoolByID 根据ID获取值班池
func (s *DutyPoolService) GetDutyPoolByID(ctx context.Context, poolID string) (*models.DutyPool, error) {
	var pool models.DutyPool
	if err := s.db.WithContext(ctx).Preload("Members.User").Preload("Dept").
		Where("id = ?", poolID).First(&pool).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("值班池不存在")
		}
		return nil, fmt.Errorf("查询值班池失败: %w", err)
	}
	return &pool, nil
}

// UpdateDutyPool 更新值班池
func (s *DutyPoolService) UpdateDutyPool(ctx context.Context, req *DutyPoolUpdateRequest, updaterID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查值班池是否存在
		var pool models.DutyPool
		if err := tx.Where("id = ?", req.ID).First(&pool).Error; err != nil {
			return fmt.Errorf("值班池不存在")
		}

		// 检查名称是否与其他池重复
		var count int64
		if err := tx.Model(&models.DutyPool{}).Where("id != ? AND pool_name = ?", req.ID, req.PoolName).Count(&count).Error; err != nil {
			return fmt.Errorf("检查值班池名称失败: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("值班池名称已存在")
		}

		// 更新值班池基本信息
		updates := map[string]interface{}{
			"pool_name":   req.PoolName,
			"dept_id":     req.DeptID,
			"description": req.Description,
			"daily_count": req.DailyCount,
			"updated_by":  updaterID,
		}

		// 只有当请求中明确指定status时才更新
		if req.Status != nil {
			updates["status"] = *req.Status
		}

		if err := tx.Model(&pool).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新值班池失败: %w", err)
		}

		// 更新成员：先删除旧成员，再添加新成员
		if err := tx.Where("pool_id = ?", req.ID).Delete(&models.DutyPoolMember{}).Error; err != nil {
			return fmt.Errorf("删除旧成员失败: %w", err)
		}

		if len(req.MemberIDs) > 0 {
			var members []models.DutyPoolMember
			for i, userID := range req.MemberIDs {
				members = append(members, models.DutyPoolMember{
					PoolID:      req.ID,
					UserID:      userID,
					MemberOrder: i,
				})
			}
			if err := tx.Create(&members).Error; err != nil {
				return fmt.Errorf("添加值班池成员失败: %w", err)
			}
		}

		return nil
	})
}

// DeleteDutyPool 删除值班池
func (s *DutyPoolService) DeleteDutyPool(ctx context.Context, poolID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否有相关排班记录
		var count int64
		if err := tx.Model(&models.DutySchedule{}).Where("pool_id = ?", poolID).Count(&count).Error; err != nil {
			return fmt.Errorf("检查排班记录失败: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("该值班池存在排班记录，无法删除")
		}

		// 删除值班池成员
		if err := tx.Where("pool_id = ?", poolID).Delete(&models.DutyPoolMember{}).Error; err != nil {
			return fmt.Errorf("删除值班池成员失败: %w", err)
		}

		// 删除值班池
		if err := tx.Delete(&models.DutyPool{}, "id = ?", poolID).Error; err != nil {
			return fmt.Errorf("删除值班池失败: %w", err)
		}

		return nil
	})
}
