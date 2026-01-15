package operations

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// WorkstationService 工位服务接口
type WorkstationService interface {
	Create(ctx context.Context, workstation *models.Workstation) error
	Update(ctx context.Context, workstation *models.Workstation) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Workstation, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	BatchUpdatePositions(ctx context.Context, items []PositionUpdateItem) error
}

type workstationService struct {
	db *gorm.DB
}

// PositionUpdateItem 位置更新项
type PositionUpdateItem struct {
	ID         string  `json:"id"`
	PositionX  int     `json:"positionX"`
	PositionY  int     `json:"positionY"`
	Rotation   *int    `json:"rotation,omitempty"`   // 旋转角度（可选）
}

// NewWorkstationService 创建工位服务实例
func NewWorkstationService(db *gorm.DB) WorkstationService {
	return &workstationService{
		db: db,
	}
}

// Create 创建工位
func (s *workstationService) Create(ctx context.Context, workstation *models.Workstation) error {
	return s.db.WithContext(ctx).Create(workstation).Error
}

// Update 更新工位
func (s *workstationService) Update(ctx context.Context, workstation *models.Workstation) error {
	return s.db.WithContext(ctx).Save(workstation).Error
}

// Delete 删除工位
func (s *workstationService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&models.Workstation{}, "id = ?", id).Error
}

// GetByID 根据ID获取工位
func (s *workstationService) GetByID(ctx context.Context, id string) (*models.Workstation, error) {
	var workstation models.Workstation
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&workstation).Error
	if err != nil {
		return nil, err
	}
	return &workstation, nil
}

// List 查询工位列表
func (s *workstationService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []models.Workstation

	query := s.db.WithContext(ctx).Model(&models.Workstation{})

	// 添加筛选条件
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("sys_workstation.workstation_name LIKE ?", "%"+name+"%")
	}
	if floorId, ok := params["floorId"].(string); ok && floorId != "" {
		query = query.Where("sys_workstation.floor_id = ?", floorId)
	}
	if status, ok := params["status"].(int); ok {
		query = query.Where("sys_workstation.status = ?", status)
	}
	if status, ok := params["status"].(float64); ok {
		query = query.Where("sys_workstation.status = ?", int(status))
	}
	if typeVal, ok := params["type"].(int); ok {
		query = query.Where("sys_workstation.workstation_type = ?", typeVal)
	}
	if typeVal, ok := params["type"].(float64); ok {
		query = query.Where("sys_workstation.workstation_type = ?", int(typeVal))
	}

	// 分页
	current := 1
	pageSize := 10
	if c, ok := params["current"].(int); ok {
		current = c
	}
	if c, ok := params["current"].(float64); ok {
		current = int(c)
	}
	if ps, ok := params["pageSize"].(int); ok {
		pageSize = ps
	}
	if ps, ok := params["pageSize"].(float64); ok {
		pageSize = int(ps)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (current - 1) * pageSize
	// JOIN ops_floors 表获取 floor_name（类型转换：varchar -> uuid）
	if err := query.
		Select("sys_workstation.*, ops_floors.name as floor_name, ops_floors.floor_no as floor_code").
		Joins("LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid").
		Offset(offset).
		Limit(pageSize).
		Order("sys_workstation.created_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// BatchDelete 批量删除工位
func (s *workstationService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&models.Workstation{}, "id IN ?", ids).Error
}

// BatchUpdatePositions 批量更新工位位置
func (s *workstationService) BatchUpdatePositions(ctx context.Context, items []PositionUpdateItem) error {
	if len(items) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			updates := map[string]interface{}{
				"position_x": item.PositionX,
				"position_y": item.PositionY,
			}
			// 如果提供了 rotation 值，则更新
			if item.Rotation != nil {
				updates["rotation"] = *item.Rotation
			}

			if err := tx.Model(&models.Workstation{}).
				Where("id = ?", item.ID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
