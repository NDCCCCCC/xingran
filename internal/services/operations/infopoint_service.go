package operations

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// InfoPointService 信息点服务接口
type InfoPointService interface {
	Create(ctx context.Context, infoPoint *operations.OpsInfoPoint) error
	Update(ctx context.Context, infoPoint *operations.OpsInfoPoint) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsInfoPoint, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type infoPointService struct {
	db *gorm.DB
}

// NewInfoPointService 创建信息点服务实例
func NewInfoPointService(db *gorm.DB) InfoPointService {
	return &infoPointService{
		db: db,
	}
}

// Create 创建信息点
func (s *infoPointService) Create(ctx context.Context, infoPoint *operations.OpsInfoPoint) error {
	// 填充冗余字段
	if err := s.populateRedundantFields(ctx, infoPoint); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(infoPoint).Error
}

// Update 更新信息点
func (s *infoPointService) Update(ctx context.Context, infoPoint *operations.OpsInfoPoint) error {
	// 填充冗余字段
	if err := s.populateRedundantFields(ctx, infoPoint); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Save(infoPoint).Error
}

// populateRedundantFields 填充冗余字段（设备名称、端口名称、工位名称）
func (s *infoPointService) populateRedundantFields(ctx context.Context, infoPoint *operations.OpsInfoPoint) error {
	// 填充工位名称
	if infoPoint.WorkstationID != "" && infoPoint.WorkstationName == nil {
		var workstation models.Workstation
		if err := s.db.WithContext(ctx).Select("workstation_name").Where("id = ?", infoPoint.WorkstationID).First(&workstation).Error; err == nil {
			infoPoint.WorkstationName = &workstation.WorkstationName
		}
	}

	// 填充设备名称
	if infoPoint.DeviceID != nil && *infoPoint.DeviceID != "" && infoPoint.DeviceName == nil {
		var device models.NetworkDevice
		if err := s.db.WithContext(ctx).Select("device_name").Where("id = ?", *infoPoint.DeviceID).First(&device).Error; err == nil {
			infoPoint.DeviceName = &device.DeviceName
		}
	}

	// 填充端口名称
	if infoPoint.PortID != nil && *infoPoint.PortID != "" && infoPoint.PortName == nil {
		var portStatus models.DevicePortStatus
		if err := s.db.WithContext(ctx).Select("interface_name").Where("id = ?", *infoPoint.PortID).First(&portStatus).Error; err == nil {
			infoPoint.PortName = &portStatus.InterfaceName
		}
	}

	return nil
}

// Delete 删除信息点
func (s *infoPointService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsInfoPoint{}, "id = ?", id).Error
}

// GetByID 根据ID获取信息点
func (s *infoPointService) GetByID(ctx context.Context, id string) (*operations.OpsInfoPoint, error) {
	var infoPoint operations.OpsInfoPoint
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&infoPoint).Error
	if err != nil {
		return nil, err
	}
	return &infoPoint, nil
}

// List 查询信息点列表
func (s *infoPointService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsInfoPoint

	query := s.db.WithContext(ctx).Model(&operations.OpsInfoPoint{})

	// 添加筛选条件
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if workstationId, ok := params["workstationId"].(string); ok && workstationId != "" {
		query = query.Where("workstation_id = ?", workstationId)
	}
	if deviceId, ok := params["deviceId"].(string); ok && deviceId != "" {
		query = query.Where("device_id = ?", deviceId)
	}
	if infoPointType, ok := params["infoPointType"].(string); ok && infoPointType != "" {
		query = query.Where("info_point_type = ?", infoPointType)
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

// BatchDelete 批量删除信息点
func (s *infoPointService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsInfoPoint{}, "id IN ?", ids).Error
}
