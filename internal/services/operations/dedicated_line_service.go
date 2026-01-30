package operations

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// populateRoomNames 填充冗余的机房名称字段
func (s *dedicatedLineService) populateRoomNames(ctx context.Context, line *operations.OpsDedicatedLine) error {
	if line.SourceRoomID != nil && *line.SourceRoomID != "" && line.SourceRoomName == nil {
		var room operations.OpsServerRoom
		if err := s.db.WithContext(ctx).Select("name").Where("id = ?", *line.SourceRoomID).First(&room).Error; err == nil {
			line.SourceRoomName = &room.Name
		}
	}

	if line.DestRoomID != nil && *line.DestRoomID != "" && line.DestRoomName == nil {
		var room operations.OpsServerRoom
		if err := s.db.WithContext(ctx).Select("name").Where("id = ?", *line.DestRoomID).First(&room).Error; err == nil {
			line.DestRoomName = &room.Name
		}
	}

	return nil
}

// DedicatedLineService 专线服务接口
type DedicatedLineService interface {
	Create(ctx context.Context, line *operations.OpsDedicatedLine) error
	Update(ctx context.Context, line *operations.OpsDedicatedLine) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsDedicatedLine, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type dedicatedLineService struct {
	db *gorm.DB
}

// NewDedicatedLineService 创建专线服务实例
func NewDedicatedLineService(db *gorm.DB) DedicatedLineService {
	return &dedicatedLineService{db: db}
}

// Create 创建专线
func (s *dedicatedLineService) Create(ctx context.Context, line *operations.OpsDedicatedLine) error {
	if err := s.populateRoomNames(ctx, line); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(line).Error
}

// Update 更新专线
func (s *dedicatedLineService) Update(ctx context.Context, line *operations.OpsDedicatedLine) error {
	if err := s.populateRoomNames(ctx, line); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Save(line).Error
}

// Delete 删除专线
func (s *dedicatedLineService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsDedicatedLine{}, "id = ?", id).Error
}

// GetByID 根据ID获取专线
func (s *dedicatedLineService) GetByID(ctx context.Context, id string) (*operations.OpsDedicatedLine, error) {
	var line operations.OpsDedicatedLine
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&line).Error
	if err != nil {
		return nil, err
	}
	return &line, nil
}

// List 查询专线列表
func (s *dedicatedLineService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsDedicatedLine

	query := s.db.WithContext(ctx).Model(&operations.OpsDedicatedLine{})

	if lineType, ok := params["lineType"].(string); ok && lineType != "" {
		query = query.Where("line_type = ?", lineType)
	}
	if isp, ok := params["isp"].(string); ok && isp != "" {
		query = query.Where("isp LIKE ?", "%"+isp+"%")
	}
	if status, ok := params["status"].(int); ok {
		query = query.Where("status = ?", status)
	}
	if status, ok := params["status"].(float64); ok {
		query = query.Where("status = ?", int(status))
	}

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
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// BatchDelete 批量删除专线
func (s *dedicatedLineService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsDedicatedLine{}, "id IN ?", ids).Error
}
