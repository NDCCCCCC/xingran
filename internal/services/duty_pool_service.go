package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
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

// DutyPoolStatistics 值班池统计结果。启停状态由 models.DutyPoolStatus 定义。
type DutyPoolStatistics struct {
	Total        int64 `json:"total"`
	Enabled      int64 `json:"enabled"`      // DutyPoolStatusEnabled
	Disabled     int64 `json:"disabled"`     // DutyPoolStatusDisabled
	TotalMembers int64 `json:"totalMembers"` // 所有非软删除池的成员总数
}

// GetDutyPoolStatistics 统计值班池总数/启停数及跨池成员总数。
// 用条件聚合(SUM CASE)避免「按当前页 list 计算统计」的错误(原前端用当前页 ~10 条算,
// 多页时统计严重偏小)。totalMembers 用子查询限定非软删除池,避免硬编码表名。
func (s *DutyPoolService) GetDutyPoolStatistics(ctx context.Context) (*DutyPoolStatistics, error) {
	var result DutyPoolStatistics
	err := s.db.WithContext(ctx).
		Model(&models.DutyPool{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS enabled", int(models.DutyPoolStatusEnabled)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS disabled", int(models.DutyPoolStatusDisabled)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计值班池失败: %w", err)
	}

	// 成员总数:仅统计非软删除池的成员(子查询自动套用 DutyPool 软删除 scope)
	if err := s.db.WithContext(ctx).
		Model(&models.DutyPoolMember{}).
		Where("pool_id IN (?)", s.db.Model(&models.DutyPool{}).Select("id")).
		Count(&result.TotalMembers).Error; err != nil {
		return nil, fmt.Errorf("统计值班池成员失败: %w", err)
	}

	return &result, nil
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

// dutyPoolAllowedSortFields 值班池列表可排序字段白名单（对应 sys_duty_pool 表列名）。
var dutyPoolAllowedSortFields = map[string]string{
	"poolName":  "pool_name",
	"deptId":    "dept_id",
	"status":    "status",
	"createdAt": "created_at",
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
	// 用户排序（白名单）优先，无 OrderByColumn 时保留 created_at DESC 默认
	query = base.ApplySort(query, req.BaseListRequest, dutyPoolAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Preload("Members.User").Preload("Dept").
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
