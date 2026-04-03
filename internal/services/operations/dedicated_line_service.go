package operations

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
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
	List(ctx context.Context, req requests.DedicatedLineListRequest) (*PageResult, error)
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

// List 查询专线列表（类型安全版本）
func (s *dedicatedLineService) List(ctx context.Context, req requests.DedicatedLineListRequest) (*PageResult, error) {
	var total int64
	var list []operations.OpsDedicatedLine

	query := s.db.WithContext(ctx).Model(&operations.OpsDedicatedLine{})

	// 添加筛选条件 - 类型安全，使用相关字段进行筛选
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.LineType != "" {
		query = query.Where("line_type = ?", req.LineType)
	}
	if req.ISP != "" {
		query = query.Where("isp = ?", req.ISP)
	}
	if req.SourceRoomName != "" {
		query = query.Where("source_room_name LIKE ?", "%"+req.SourceRoomName+"%")
	}
	if req.DestRoomName != "" {
		query = query.Where("dest_room_name LIKE ?", "%"+req.DestRoomName+"%")
	}
	if req.SourceDeviceName != "" {
		query = query.Where("source_device_name LIKE ?", "%"+req.SourceDeviceName+"%")
	}
	if req.DestDeviceName != "" {
		query = query.Where("dest_device_name LIKE ?", "%"+req.DestDeviceName+"%")
	}
	if req.CarrierContactName != "" {
		query = query.Where("carrier_contact_name LIKE ?", "%"+req.CarrierContactName+"%")
	}
	if req.HasStatus() {
		query = query.Where("status = ?", req.GetStatus(0))
	}

	// 分页 - 使用请求结构体的方法
	offset := req.GetOffset()
	_, pageSize := req.GetPagination()
	current, _ := req.GetPagination()

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

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
