// Package workorder 提供工单管理的模块化服务
// 将原本的 workorder_service.go (1,813行) 拆分为多个职责单一的模块
package workorder

import (
	"gorm.io/gorm"
)

// WorkOrderService 工单服务（主服务）
// 提供统一的访问入口，内部委托给各个专门的服务
type WorkOrderService struct {
	Base       *BaseService
	Comment    *CommentService
	Category   *CategoryService
	Rating     *RatingService
	Assignment *AssignmentService
	Statistics *StatisticsService
	Periodic   *PeriodicService
	Config     *ConfigService
}

// NewWorkOrderService 创建工单服务
func NewWorkOrderService(db *gorm.DB) *WorkOrderService {
	return &WorkOrderService{
		Base:       NewBaseService(db),
		Comment:    NewCommentService(db),
		Category:   NewCategoryService(db),
		Rating:     NewRatingService(db),
		Assignment: NewAssignmentService(db),
		Statistics: NewStatisticsService(db),
		Periodic:   NewPeriodicService(db),
		Config:     NewConfigService(db),
	}
}
