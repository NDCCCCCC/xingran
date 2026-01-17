package operations

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// RoomDeviceService 机房设备服务接口
type RoomDeviceService interface {
	Create(ctx context.Context, device *operations.OpsRoomDevice) error
	Update(ctx context.Context, device *operations.OpsRoomDevice) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsRoomDevice, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type roomDeviceService struct {
	db *gorm.DB
}

// NewRoomDeviceService 创建机房设备服务实例
func NewRoomDeviceService(db *gorm.DB) RoomDeviceService {
	return &roomDeviceService{db: db}
}

// Create 创建机房设备
func (s *roomDeviceService) Create(ctx context.Context, device *operations.OpsRoomDevice) error {
	// 验证机房存在性
	if err := s.validateRoom(ctx, device.RoomID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(device).Error
}

// Update 更新机房设备
func (s *roomDeviceService) Update(ctx context.Context, device *operations.OpsRoomDevice) error {
	// 验证机房存在性
	if err := s.validateRoom(ctx, device.RoomID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Save(device).Error
}

// Delete 删除机房设备
func (s *roomDeviceService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsRoomDevice{}, "id = ?", id).Error
}

// GetByID 根据ID获取机房设备
func (s *roomDeviceService) GetByID(ctx context.Context, id string) (*operations.OpsRoomDevice, error) {
	var device operations.OpsRoomDevice
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// List 查询机房设备列表
func (s *roomDeviceService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsRoomDevice

	query := s.db.WithContext(ctx).Model(&operations.OpsRoomDevice{})

	// 添加筛选条件
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if deviceType, ok := params["deviceType"].(string); ok && deviceType != "" {
		query = query.Where("device_type = ?", deviceType)
	}
	if roomId, ok := params["roomId"].(string); ok && roomId != "" {
		query = query.Where("room_id = ?", roomId)
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

// BatchDelete 批量删除机房设备
func (s *roomDeviceService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsRoomDevice{}, "id IN ?", ids).Error
}

// validateRoom 验证机房存在性
func (s *roomDeviceService) validateRoom(ctx context.Context, roomID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsServerRoom{}).Where("id = ?", roomID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("所属机房不存在")
	}
	return nil
}
