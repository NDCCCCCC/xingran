package operations

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// ServerRoomService 机房服务接口
type ServerRoomService interface {
	Create(ctx context.Context, room *operations.OpsServerRoom) error
	Update(ctx context.Context, room *operations.OpsServerRoom) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsServerRoom, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type serverRoomService struct {
	db *gorm.DB
}

// NewServerRoomService 创建机房服务实例
func NewServerRoomService(db *gorm.DB) ServerRoomService {
	return &serverRoomService{
		db: db,
	}
}

// Create 创建机房
func (s *serverRoomService) Create(ctx context.Context, room *operations.OpsServerRoom) error {
	// 验证楼层存在性
	if err := s.validateFloor(ctx, room.FloorID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(room).Error
}

// Update 更新机房
func (s *serverRoomService) Update(ctx context.Context, room *operations.OpsServerRoom) error {
	// 验证楼层存在性
	if err := s.validateFloor(ctx, room.FloorID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Save(room).Error
}

// Delete 删除机房
func (s *serverRoomService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsServerRoom{}, "id = ?", id).Error
}

// GetByID 根据ID获取机房
func (s *serverRoomService) GetByID(ctx context.Context, id string) (*operations.OpsServerRoom, error) {
	var room operations.OpsServerRoom
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// List 查询机房列表
func (s *serverRoomService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsServerRoom

	query := s.db.WithContext(ctx).Model(&operations.OpsServerRoom{})

	// 添加筛选条件
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
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

// BatchDelete 批量删除机房
func (s *serverRoomService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsServerRoom{}, "id IN ?", ids).Error
}

// validateFloor 验证楼层存在性
func (s *serverRoomService) validateFloor(ctx context.Context, floorID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsFloor{}).Where("id = ?", floorID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("所属楼层不存在")
	}
	return nil
}
