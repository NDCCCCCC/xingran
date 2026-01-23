package monitor

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// OperLogService 操作日志服务接口
type OperLogService interface {
	// List 查询操作日志列表
	List(ctx context.Context, params OperLogListParams) (*PageResult, error)
	// GetByID 获取操作日志详情
	GetByID(ctx context.Context, id string) (*models.OperLog, error)
	// Delete 删除操作日志
	Delete(ctx context.Context, id string) error
	// BatchDelete 批量删除操作日志
	BatchDelete(ctx context.Context, ids []string) error
	// Clean 清空操作日志
	Clean(ctx context.Context) error
}

// operLogService 操作日志服务实现
type operLogService struct {
	db *gorm.DB
}

// NewOperLogService 创建操作日志服务实例
func NewOperLogService(db *gorm.DB) OperLogService {
	return &operLogService{db: db}
}

// OperLogListParams 操作日志列表查询参数
type OperLogListParams struct {
	Title        *string
	BusinessType *int
	Status       *int
	OperName     *string
	BeginTime    *string
	EndTime      *string
	Current      int
	PageSize     int
}

// DefaultOperLogListParams 默认列表参数
func DefaultOperLogListParams() OperLogListParams {
	return OperLogListParams{
		Current:  1,
		PageSize: 10,
	}
}

// List 查询操作日志列表
func (s *operLogService) List(ctx context.Context, params OperLogListParams) (*PageResult, error) {
	var total int64
	var list []models.OperLog

	query := s.db.WithContext(ctx).Model(&models.OperLog{})

	// 标题模糊查询
	if params.Title != nil && *params.Title != "" {
		query = query.Where("title LIKE ?", "%"+*params.Title+"%")
	}

	// 业务类型查询
	if params.BusinessType != nil {
		query = query.Where("business_type = ?", *params.BusinessType)
	}

	// 状态查询
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 操作人员模糊查询
	if params.OperName != nil && *params.OperName != "" {
		query = query.Where("operator_name LIKE ?", "%"+*params.OperName+"%")
	}

	// 开始时间查询
	if params.BeginTime != nil && *params.BeginTime != "" {
		query = query.Where("oper_time >= ?", *params.BeginTime)
	}

	// 结束时间查询
	if params.EndTime != nil && *params.EndTime != "" {
		query = query.Where("oper_time <= ?", *params.EndTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询操作日志总数失败: %w", err)
	}

	// 查询分页数据
	offset := (params.Current - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("oper_time DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询操作日志列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetByID 获取操作日志详情
func (s *operLogService) GetByID(ctx context.Context, id string) (*models.OperLog, error) {
	var operLog models.OperLog
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&operLog).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("操作日志不存在")
		}
		return nil, fmt.Errorf("查询操作日志失败: %w", err)
	}
	return &operLog, nil
}

// Delete 删除操作日志
func (s *operLogService) Delete(ctx context.Context, id string) error {
	// 检查操作日志是否存在
	var operLog models.OperLog
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&operLog).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("操作日志不存在")
		}
		return fmt.Errorf("查询操作日志失败: %w", err)
	}

	// 删除操作日志
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.OperLog{}).Error; err != nil {
		return fmt.Errorf("删除操作日志失败: %w", err)
	}

	return nil
}

// BatchDelete 批量删除操作日志
func (s *operLogService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	// 批量删除操作日志
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.OperLog{}).Error; err != nil {
		return fmt.Errorf("批量删除操作日志失败: %w", err)
	}

	return nil
}

// Clean 清空操作日志
func (s *operLogService) Clean(ctx context.Context) error {
	if err := s.db.WithContext(ctx).Exec("DELETE FROM sys_oper_log").Error; err != nil {
		return fmt.Errorf("清空操作日志失败: %w", err)
	}
	return nil
}
