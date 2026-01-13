package operations

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// WorkstationService 工位服务接口
type WorkstationService interface {
	Create(ctx context.Context, workstation *operations.OpsWorkstation) error
	Update(ctx context.Context, workstation *operations.OpsWorkstation) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsWorkstation, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type workstationService struct {
	db *gorm.DB
}

// NewWorkstationService 创建工位服务实例
func NewWorkstationService(db *gorm.DB) WorkstationService {
	return &workstationService{db: db}
}

// Create 创建工位
func (s *workstationService) Create(ctx context.Context, workstation *operations.OpsWorkstation) error {
	// 验证楼层存在性
	if err := s.validateFloor(ctx, workstation.FloorID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(workstation).Error
}

// Update 更新工位
func (s *workstationService) Update(ctx context.Context, workstation *operations.OpsWorkstation) error {
	if workstation.FloorID != "" {
		if err := s.validateFloor(ctx, workstation.FloorID); err != nil {
			return err
		}
	}
	return s.db.WithContext(ctx).Save(workstation).Error
}

// Delete 删除工位
func (s *workstationService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsWorkstation{}, "id = ?", id).Error
}

// GetByID 根据ID获取工位
func (s *workstationService) GetByID(ctx context.Context, id string) (*operations.OpsWorkstation, error) {
	var workstation operations.OpsWorkstation
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&workstation).Error
	if err != nil {
		return nil, err
	}
	return &workstation, nil
}

// List 查询工位列表
func (s *workstationService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsWorkstation

	query := s.db.WithContext(ctx).Model(&operations.OpsWorkstation{})

	// 添加筛选条件
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if code, ok := params["code"].(string); ok && code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}
	if floorId, ok := params["floorId"].(string); ok && floorId != "" {
		query = query.Where("floor_id = ?", floorId)
	}
	if status, ok := params["status"].(int); ok {
		query = query.Where("status = ?", status)
	}
	if status, ok := params["status"].(float64); ok {
		query = query.Where("status = ?", int(status))
	}
	if typeVal, ok := params["type"].(int); ok {
		query = query.Where("type = ?", typeVal)
	}
	if typeVal, ok := params["type"].(float64); ok {
		query = query.Where("type = ?", int(typeVal))
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
	if err := query.Offset(offset).Limit(pageSize).Order("code ASC").Find(&list).Error; err != nil {
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
	return s.db.WithContext(ctx).Delete(&operations.OpsWorkstation{}, "id IN ?", ids).Error
}

// validateFloor 验证楼层存在性
func (s *workstationService) validateFloor(ctx context.Context, floorID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsFloor{}).Where("id = ?", floorID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("所属楼层不存在")
	}
	return nil
}
