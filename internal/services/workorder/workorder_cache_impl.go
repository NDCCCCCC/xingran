package workorder

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"gorm.io/gorm"
)

// WorkOrderCacheService 工单缓存服务接口
type WorkOrderCacheService interface {
	// 基础服务方法
	GetList(ctx context.Context, req *ListRequest) ([]models.WorkOrder, int64, error)
	GetStatusStatistics(ctx context.Context) (*WorkOrderStatusStatistics, error)
	GetByID(ctx context.Context, id string) (*models.WorkOrder, error)
	Create(ctx context.Context, req *CreateRequest, submitterID string) (*models.WorkOrder, error)
	Update(ctx context.Context, req *UpdateRequest, operatorID string) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error

	// 待办工单
	GetMyPending(ctx context.Context, req *GetMyPendingRequest, userID string) ([]models.WorkOrder, int64, error)

	// 统计服务方法
	GetStatistics(ctx context.Context) (*Statistics, error)

	// 子服务访问（用于未缓存的方法）
	Assignment() *AssignmentService
	Comment() *CommentService
	Category() *CategoryService
	Periodic() *PeriodicService
	Config() *ConfigService

	// 缓存失效方法
	InvalidateWorkOrderCache(ctx context.Context, workOrderID string) error
	InvalidateMyPendingCache(ctx context.Context, userID string) error
	InvalidateStatisticsCache(ctx context.Context) error
	InvalidateAllWorkOrderCache(ctx context.Context) error
}

// workOrderCacheServiceImpl 工单缓存服务实现
type workOrderCacheServiceImpl struct {
	db         *gorm.DB
	base       *WorkOrderService
	statistics *StatisticsService
	cache      systemServices.CacheProvider
	config     *services.CacheConfigService
}

// NewWorkOrderServiceWithCache 创建带缓存的工单服务
func NewWorkOrderServiceWithCache(
	db *gorm.DB,
	cache systemServices.CacheProvider,
	config *services.CacheConfigService,
) WorkOrderCacheService {
	return &workOrderCacheServiceImpl{
		db:         db,
		base:       NewWorkOrderService(db),
		statistics: NewStatisticsService(db),
		cache:      cache,
		config:     config,
	}
}

// getExpiration 获取缓存过期时间
func (s *workOrderCacheServiceImpl) getExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if s.config != nil {
		return s.config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}

// ==================== 基础服务方法（部分缓存） ====================

// GetList 获取工单列表（不缓存，参数多变）
func (s *workOrderCacheServiceImpl) GetList(ctx context.Context, req *ListRequest) ([]models.WorkOrder, int64, error) {
	return s.base.Base.GetList(ctx, req)
}

// GetStatusStatistics 统计工单各状态计数(供统计卡片),委托给基础服务。不缓存。
func (s *workOrderCacheServiceImpl) GetStatusStatistics(ctx context.Context) (*WorkOrderStatusStatistics, error) {
	return s.base.Base.GetStatusStatistics(ctx)
}

// GetByID 获取工单详情（不缓存，查询频率低）
func (s *workOrderCacheServiceImpl) GetByID(ctx context.Context, id string) (*models.WorkOrder, error) {
	return s.base.Base.GetByID(ctx, id)
}

// Create 创建工单（带缓存失效）
func (s *workOrderCacheServiceImpl) Create(ctx context.Context, req *CreateRequest, submitterID string) (*models.WorkOrder, error) {
	workOrder, err := s.base.Base.Create(ctx, req, submitterID)
	if err != nil {
		return nil, err
	}
	// 清除统计缓存
	_ = s.InvalidateStatisticsCache(ctx)
	// 如果指定了处理人，清除其待办缓存
	if workOrder.AssigneeID != nil && *workOrder.AssigneeID != "" {
		_ = s.InvalidateMyPendingCache(ctx, *workOrder.AssigneeID)
	}
	return workOrder, nil
}

// Update 更新工单（带缓存失效）
func (s *workOrderCacheServiceImpl) Update(ctx context.Context, req *UpdateRequest, operatorID string) error {
	// 先获取工单以便获取处理人信息
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 继续执行，可能工单已被删除
			_ = err
		}
	}

	oldAssigneeID := workOrder.AssigneeID

	if err := s.base.Base.Update(ctx, req, operatorID); err != nil {
		return err
	}

	// 清除该工单的缓存
	_ = s.InvalidateWorkOrderCache(ctx, req.ID)

	// 如果处理人变更，清除旧处理人的待办缓存
	if oldAssigneeID != nil && req.AssigneeID != nil && *oldAssigneeID != *req.AssigneeID {
		_ = s.InvalidateMyPendingCache(ctx, *oldAssigneeID)
	}
	// 清除新处理人的待办缓存
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		_ = s.InvalidateMyPendingCache(ctx, *req.AssigneeID)
	}
	// 清除统计缓存
	_ = s.InvalidateStatisticsCache(ctx)

	return nil
}

// Delete 删除工单（带缓存失效）
func (s *workOrderCacheServiceImpl) Delete(ctx context.Context, id string) error {
	// 先获取工单以便获取处理人信息
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = err // 工单不存在，继续执行
		}
	}

	assigneeID := workOrder.AssigneeID

	if err := s.base.Base.Delete(ctx, id); err != nil {
		return err
	}

	// 清除缓存
	_ = s.InvalidateWorkOrderCache(ctx, id)
	if assigneeID != nil && *assigneeID != "" {
		_ = s.InvalidateMyPendingCache(ctx, *assigneeID)
	}
	_ = s.InvalidateStatisticsCache(ctx)

	return nil
}

// BatchDelete 批量删除工单（带缓存失效）
func (s *workOrderCacheServiceImpl) BatchDelete(ctx context.Context, ids []string) error {
	// 先获取所有工单以便获取处理人信息
	var workOrders []models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&workOrders).Error; err != nil {
		return err
	}

	if err := s.base.Base.BatchDelete(ctx, ids); err != nil {
		return err
	}

	// 清除所有工单缓存
	for _, id := range ids {
		_ = s.InvalidateWorkOrderCache(ctx, id)
	}

	// 清除所有涉及的处理人的待办缓存
	assigneeMap := make(map[string]bool)
	for _, wo := range workOrders {
		if wo.AssigneeID != nil && *wo.AssigneeID != "" {
			assigneeMap[*wo.AssigneeID] = true
		}
	}
	for assigneeID := range assigneeMap {
		_ = s.InvalidateMyPendingCache(ctx, assigneeID)
	}

	// 清除统计缓存
	_ = s.InvalidateStatisticsCache(ctx)

	return nil
}

// ==================== 待办工单（带缓存） ====================

// GetMyPending 获取当前用户的待办工单（带缓存）
func (s *workOrderCacheServiceImpl) GetMyPending(ctx context.Context, req *GetMyPendingRequest, userID string) ([]models.WorkOrder, int64, error) {
	cacheKey := fmt.Sprintf("workorder:my_pending:%s", userID)
	var result struct {
		List  []models.WorkOrder
		Total int64
	}

	expiration := s.getExpiration("cache.workorder.my_pending", 2*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		list, total, err := s.base.Base.GetMyPending(ctx, req, userID)
		if err != nil {
			return nil, err
		}
		return struct {
			List  []models.WorkOrder
			Total int64
		}{
			List:  list,
			Total: total,
		}, nil
	})

	if err != nil {
		return nil, 0, err
	}
	return result.List, result.Total, nil
}

// ==================== 统计服务（带缓存） ====================

// GetStatistics 获取工单统计数据（带缓存）
func (s *workOrderCacheServiceImpl) GetStatistics(ctx context.Context) (*Statistics, error) {
	cacheKey := "workorder:statistics"
	var result Statistics

	expiration := s.getExpiration("cache.workorder.statistics", 5*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.statistics.Get(ctx)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ==================== 缓存失效方法 ====================

// InvalidateWorkOrderCache 失效工单缓存
func (s *workOrderCacheServiceImpl) InvalidateWorkOrderCache(ctx context.Context, workOrderID string) error {
	keys := []string{fmt.Sprintf("workorder:detail:%s", workOrderID)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "WORKORDER")
	return nil
}

// InvalidateMyPendingCache 失效待办工单缓存
func (s *workOrderCacheServiceImpl) InvalidateMyPendingCache(ctx context.Context, userID string) error {
	keys := []string{fmt.Sprintf("workorder:my_pending:%s", userID)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "WORKORDER")
	return nil
}

// InvalidateStatisticsCache 失效统计缓存
func (s *workOrderCacheServiceImpl) InvalidateStatisticsCache(ctx context.Context) error {
	keys := []string{"workorder:statistics"}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "WORKORDER")
	return nil
}

// InvalidateAllWorkOrderCache 失效所有工单缓存
func (s *workOrderCacheServiceImpl) InvalidateAllWorkOrderCache(ctx context.Context) error {
	systemServices.InvalidateCacheByPattern(ctx, s.cache, []string{"workorder:*"}, "WORKORDER")
	return nil
}

// ==================== 子服务访问 ====================

func (s *workOrderCacheServiceImpl) Assignment() *AssignmentService {
	return s.base.Assignment
}

func (s *workOrderCacheServiceImpl) Comment() *CommentService {
	return s.base.Comment
}

func (s *workOrderCacheServiceImpl) Category() *CategoryService {
	return s.base.Category
}

func (s *workOrderCacheServiceImpl) Periodic() *PeriodicService {
	return s.base.Periodic
}

func (s *workOrderCacheServiceImpl) Config() *ConfigService {
	return s.base.Config
}
