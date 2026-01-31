package operations

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

const (
	workstationTable        = "sys_workstation"
	floorTable             = "ops_floors"
	workstationJoinSelect  = "sys_workstation.*, ops_floors.name as floor_name, ops_floors.floor_no as floor_code"
	workstationJoinClause  = "LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid"
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
	ID        string `json:"id"`
	PositionX int    `json:"positionX"`
	PositionY int    `json:"positionY"`
	Rotation  *int   `json:"rotation,omitempty"` // 旋转角度（可选）
	Width     *int   `json:"width,omitempty"`     // 工位宽度（毫米，可选）
	Depth     *int   `json:"depth,omitempty"`     // 工位深度（毫米，可选）
	DeskType  *int   `json:"deskType,omitempty"`  // 桌型（可选）：0=一字型, 1=L型
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

func (s *workstationService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Table(workstationTable).Where("id = ?", id).Delete(&models.Workstation{}).Error
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
	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("sys_workstation.workstation_name LIKE ?", "%"+name+"%")
	}
	if floorId := extractStringParam(params, "floorId"); floorId != "" {
		query = query.Where("sys_workstation.floor_id = ?", floorId)
	}
	if floorCode := extractStringParam(params, "floorCode"); floorCode != "" {
		query = query.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE "+floorTable+".id = sys_workstation.floor_id::uuid AND "+floorTable+".floor_no = ?)", floorCode)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("sys_workstation.status = ?", status)
	}
	if typeVal := extractIntParam(params, "type", -1); typeVal >= 0 {
		query = query.Where("sys_workstation.workstation_type = ?", typeVal)
	}

	// 分页
	pagination := extractPagination(params)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := calculateOffset(pagination)
	// JOIN ops_floors 表获取 floor_name（类型转换：varchar -> uuid）
	if err := query.
		Select(workstationJoinSelect).
		Joins(workstationJoinClause).
		Offset(offset).
		Limit(pagination.PageSize).
		Order("sys_workstation.created_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  pagination.Current,
		PageSize: pagination.PageSize,
	}, nil
}

func (s *workstationService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Table(workstationTable).Where("id IN ?", ids).Delete(&models.Workstation{}).Error
}

// BatchUpdatePositions 批量更新工位位置和尺寸
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

			if item.Rotation != nil {
				updates["rotation"] = *item.Rotation
			}
			if item.Width != nil {
				updates["width"] = *item.Width
			}
			if item.Depth != nil {
				updates["depth"] = *item.Depth
			}
			if item.DeskType != nil {
				updates["desk_type"] = *item.DeskType
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
