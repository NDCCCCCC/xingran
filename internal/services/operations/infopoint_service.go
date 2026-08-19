package operations

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
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
	// Statistics 信息点统计(专用 COUNT 聚合,不依赖分页列表)。
	Statistics(ctx context.Context) (*InfoPointStatisticsResult, error)
	// SearchInfoPointOptions 信息点下拉数据源(LIKE 模糊 + 工位/类型/状态筛选,LIMIT 50)。
	SearchInfoPointOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error)
}

// InfoPointStatisticsResult 信息点统计结果(status: 0=正常 1=故障 2=停用)。
type InfoPointStatisticsResult struct {
	Total    int64 `json:"total"`
	Normal   int64 `json:"normal"`   // operations.InfoPointStatusNormal
	Fault    int64 `json:"fault"`    // operations.InfoPointStatusFault
	Disabled int64 `json:"disabled"` // operations.InfoPointStatusDisabled
}

// Statistics 统计信息点(按 status 聚合,排除软删除)。
func (s *infoPointService) Statistics(ctx context.Context) (*InfoPointStatisticsResult, error) {
	var result InfoPointStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&operations.OpsInfoPoint{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS normal", int(operations.InfoPointStatusNormal)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS fault", int(operations.InfoPointStatusFault)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS disabled", int(operations.InfoPointStatusDisabled)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
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

// infoPointAllowedSortFields 信息点可排序字段白名单。
// 因 List 有 LEFT JOIN workstation/floors/buildings,字段值带表别名。
var infoPointAllowedSortFields = map[string]string{
	"name":          "ops_info_points.name",
	"infoPointType": "ops_info_points.info_point_type",
	"status":        "ops_info_points.status",
	"createdAt":     "ops_info_points.created_at",
	"workstationId": "ops_info_points.workstation_id",
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
// 通过 LEFT JOIN 工位/楼层/楼宇 动态填充 workstation_name/building_name/floor_name/building_id
// (这些字段是 JOIN 虚拟字段，见 OpsInfoPoint model 的 gorm:"->;-:migration" tag，非物理列)
func (s *infoPointService) GetByID(ctx context.Context, id string) (*operations.OpsInfoPoint, error) {
	var infoPoint operations.OpsInfoPoint
	err := s.db.WithContext(ctx).
		Select("ops_info_points.*, ops_floors.name as floor_name, ops_buildings.name as building_name, ops_buildings.id as building_id, sys_workstation.workstation_name").
		Joins("LEFT JOIN sys_workstation ON CAST(sys_workstation.id AS TEXT) = ops_info_points.workstation_id").
		Joins("LEFT JOIN ops_floors ON CAST(ops_floors.id AS TEXT) = sys_workstation.floor_id").
		Joins("LEFT JOIN ops_buildings ON CAST(ops_buildings.id AS TEXT) = sys_workstation.building_id").
		Where("ops_info_points.id = ?", id).
		First(&infoPoint).Error
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
				JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id
				JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id
				JOIN sys_dept d ON CAST(d.id AS TEXT) = b.org_id
				WHERE CAST(w.id AS TEXT) = ops_info_points.workstation_id
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
	// 用户排序(白名单,带表别名)优先,无 OrderByColumn 时保留原默认
	query = base.ApplySort(query, req.BaseListRequest, infoPointAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("ops_info_points.created_at DESC")
	}
	if err := query.
		Select("ops_info_points.*, ops_floors.name as floor_name, ops_buildings.name as building_name, ops_buildings.id as building_id, sys_workstation.workstation_name").
		Joins("LEFT JOIN sys_workstation ON CAST(sys_workstation.id AS TEXT) = ops_info_points.workstation_id").
		Joins("LEFT JOIN ops_floors ON CAST(ops_floors.id AS TEXT) = sys_workstation.floor_id").
		Joins("LEFT JOIN ops_buildings ON CAST(ops_buildings.id AS TEXT) = sys_workstation.building_id").
		Offset(offset).
		Limit(pageSize).
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

// SearchInfoPointOptions 信息点下拉数据源(name LIKE 模糊 + workstationId/infoPointType/status 筛选,LIMIT 50)。
// 与 List 同款 WHERE 语义;只 SELECT id+name 两列,无 JOIN 无分页。
func (s *infoPointService) SearchInfoPointOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	var result []DropdownOption

	query := s.db.WithContext(ctx).Table("ops_info_points").
		Select("ops_info_points.id AS value, ops_info_points.name AS label").
		Limit(DropdownMaxRows)

	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("ops_info_points.name LIKE ?", "%"+name+"%")
	}
	// 兼容前端传 workstationId 或旧字段 workId
	if workstationId := extractStringParam(params, "workstationId"); workstationId != "" {
		query = query.Where("ops_info_points.workstation_id = ?", workstationId)
	} else if workId := extractStringParam(params, "workId"); workId != "" {
		query = query.Where("ops_info_points.workstation_id = ?", workId)
	}
	if infoPointType := extractStringParam(params, "infoPointType"); infoPointType != "" {
		query = query.Where("ops_info_points.info_point_type = ?", infoPointType)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("ops_info_points.status = ?", status)
	}

	if err := query.Order("ops_info_points.name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}
