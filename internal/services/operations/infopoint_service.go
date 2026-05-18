package operations

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
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
	List(ctx context.Context, req requests.InfoPointListRequest) (*PageResult, error)
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

// List 查询信息点列表（类型安全版本）
func (s *infoPointService) List(ctx context.Context, req requests.InfoPointListRequest) (*PageResult, error) {
	var total int64
	var list []operations.OpsInfoPoint

	query := s.db.WithContext(ctx).Model(&operations.OpsInfoPoint{})

	// 添加筛选条件 - 类型安全，无需类型断言
	if req.Name != "" {
		query = query.Where("ops_info_points.name LIKE ?", "%"+req.Name+"%")
	}
	// 优先使用新字段 workstationId，兼容旧字段 workId
	workstationID := req.WorkstationID
	if workstationID == "" {
		workstationID = req.WorkID
	}
	if workstationID != "" {
		query = query.Where("ops_info_points.workstation_id = ?", workstationID)
	}
	// 优先使用新字段 infoPointType，兼容旧字段 pointType
	infoPointType := req.InfoPointType
	if infoPointType == "" {
		infoPointType = req.PointType
	}
	if infoPointType != "" {
		query = query.Where("ops_info_points.info_point_type = ?", infoPointType)
	}
	if req.HasStatus() {
		query = query.Where("ops_info_points.status = ?", req.GetStatus(0))
	}
	// 通过关联工位、楼层、楼宇的 orgId 筛选部门（包含子部门）
	// 信息点 → 工位 → 楼层 → 楼宇 → 部门
	if req.OrgID != "" {
		// 使用 EXISTS 子查询避免与现有 JOIN 冲突
		// 查询该部门及其所有子部门：ancestors 包含该部门ID，或 ID 等于该部门ID
		query = query.Where(`
			EXISTS (
				SELECT 1 FROM sys_workstation w
				JOIN ops_floors f ON f.id = w.floor_id::uuid
				JOIN ops_buildings b ON b.id = f.building_id::uuid
				JOIN sys_dept d ON d.id::text = b.org_id
				WHERE w.id::text = ops_info_points.workstation_id
				AND (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?)
				AND w.deleted_at IS NULL
				AND f.deleted_at IS NULL
				AND b.deleted_at IS NULL
			)
		`, req.OrgID, "%,"+req.OrgID, req.OrgID)
	}

	// 分页 - 使用请求结构体的方法
	offset := req.GetOffset()
	_, pageSize := req.GetPagination()
	current, _ := req.GetPagination()

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// JOIN 楼宇和楼层表获取关联信息（需要类型转换：floor_id/building_id是varchar，而floors/buildings的id是UUID）
	// 注意：device_name和port_name是ops_info_points表的冗余字段，通过ops_info_points.*已经包含
	if err := query.
		Select("ops_info_points.*, ops_floors.name as floor_name, ops_buildings.name as building_name, ops_buildings.id as building_id, sys_workstation.workstation_name").
		Joins("LEFT JOIN sys_workstation ON sys_workstation.id::text = ops_info_points.workstation_id").
		Joins("LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = sys_workstation.building_id::uuid").
		Offset(offset).
		Limit(pageSize).
		Order("ops_info_points.created_at DESC").
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

// BatchDelete 批量删除信息点
func (s *infoPointService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsInfoPoint{}, "id IN ?", ids).Error
}
