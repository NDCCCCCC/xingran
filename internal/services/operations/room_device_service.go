package operations

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// RoomDeviceService 机房设备服务接口
type RoomDeviceService interface {
	Create(ctx context.Context, device *operations.OpsRoomDevice) error
	Update(ctx context.Context, device *operations.OpsRoomDevice) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsRoomDevice, error)
	List(ctx context.Context, req requests.RoomDeviceListRequest) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	// Statistics 机房设备统计(专用 COUNT 聚合,不依赖分页列表)。
	Statistics(ctx context.Context) (*RoomDeviceStatisticsResult, error)
	// SearchRoomDeviceOptions 机房设备下拉数据源(LIKE 模糊 + roomId/类型/状态筛选,LIMIT 50)。
	SearchRoomDeviceOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error)
}

// RoomDeviceStatisticsResult 机房设备统计结果(status: 0=正常 1=故障 2=报废)。
type RoomDeviceStatisticsResult struct {
	Total    int64 `json:"total"`
	Normal   int64 `json:"normal"`   // status = 0
	Fault    int64 `json:"fault"`    // status = 1
	Scrapped int64 `json:"scrapped"` // status = 2
}

// Statistics 统计机房设备(按 status 聚合,排除软删除)。
func (s *roomDeviceService) Statistics(ctx context.Context) (*RoomDeviceStatisticsResult, error) {
	var result RoomDeviceStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&operations.OpsRoomDevice{}).
		Select(
			"COUNT(*) AS total",
			"COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END), 0) AS normal",
			"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS fault",
			"COALESCE(SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END), 0) AS scrapped",
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type roomDeviceService struct {
	db *gorm.DB
}

// NewRoomDeviceService 创建机房设备服务实例
func NewRoomDeviceService(db *gorm.DB) RoomDeviceService {
	return &roomDeviceService{db: db}
}

// roomDeviceAllowedSortFields 机房设备可排序字段白名单。
// 因 List 有 LEFT JOIN server_rooms,字段值必须带表别名限定。
var roomDeviceAllowedSortFields = map[string]string{
	"name":       "ops_room_devices.name",
	"deviceType": "ops_room_devices.device_type",
	"ipAddress":  "ops_room_devices.ip_address",
	"status":     "ops_room_devices.status",
	"createdAt":  "ops_room_devices.created_at",
	"roomName":   "ops_server_rooms.name",
}

// Create 创建机房设备
func (s *roomDeviceService) Create(ctx context.Context, device *operations.OpsRoomDevice) error {
	// 验证机房存在性
	if err := s.validateRoom(ctx, device.RoomID); err != nil {
		return err
	}

	err := s.db.WithContext(ctx).Create(device).Error
	if err != nil && isDuplicateKeyError(err) {
		return apperrors.DeviceCodeAlreadyExists()
	}
	return err
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
	err := s.db.WithContext(ctx).
		Joins("LEFT JOIN ops_server_rooms ON ops_server_rooms.id = ops_room_devices.room_id::uuid").
		Select("ops_room_devices.*, ops_server_rooms.name as room_name").
		Where("ops_room_devices.id = ?", id).
		First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// List 查询机房设备列表（类型安全版本）
func (s *roomDeviceService) List(ctx context.Context, req requests.RoomDeviceListRequest) (*PageResult, error) {
	var total int64
	var list []operations.OpsRoomDevice

	query := s.db.WithContext(ctx).Model(&operations.OpsRoomDevice{})

	// 先 JOIN 机房表和楼宇表，便于后续筛选和显示机房名称
	// 注意：server_rooms.building_id 是 varchar，需要转换为 uuid 才能与 buildings.id 比较
	query = query.Joins("LEFT JOIN ops_server_rooms ON ops_server_rooms.id = ops_room_devices.room_id::uuid").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = ops_server_rooms.building_id::uuid")

	// 添加筛选条件 - 类型安全，无需类型断言
	if req.Name != "" {
		query = query.Where("ops_room_devices.name LIKE ?", "%"+req.Name+"%")
	}
	if req.DeviceType != "" {
		query = query.Where("ops_room_devices.device_type = ?", req.DeviceType)
	}
	// 机房ID筛选优先
	if req.RoomID != "" {
		query = query.Where("ops_room_devices.room_id = ?", req.RoomID)
	}
	// 部门筛选：通过楼宇表筛选（设备 → 机房 → 楼宇 → 部门）
	if req.OrgID != "" {
		query = query.Where("ops_buildings.org_id = ?", req.OrgID)
	}
	if req.IPAddress != "" {
		query = query.Where("ops_room_devices.ip_address = ?", req.IPAddress)
	}
	if req.HasStatus() {
		query = query.Where("ops_room_devices.status = ?", req.GetStatus(0))
	}

	// 分页 - 使用请求结构体的方法
	offset := req.GetOffset()
	_, pageSize := req.GetPagination()
	current, _ := req.GetPagination()

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 选择字段（包括机房名称）
	query = query.Select("ops_room_devices.*, ops_server_rooms.name as room_name")

	// 用户排序(白名单,带表别名)优先,无 OrderByColumn 时保留原默认
	query = base.ApplySort(query, req.BaseListRequest, roomDeviceAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("ops_room_devices.created_at DESC")
	}
	if err := query.Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
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

// SearchRoomDeviceOptions 机房设备下拉数据源(name LIKE 模糊 + roomId/deviceType/status 筛选,LIMIT 50)。
// 与 List 同款 WHERE 语义;只 SELECT id+name 两列,无需 JOIN server_rooms(orgId 不支持以简化 SQL)。
func (s *roomDeviceService) SearchRoomDeviceOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	var result []DropdownOption

	query := s.db.WithContext(ctx).Table("ops_room_devices").
		Select("ops_room_devices.id AS value, ops_room_devices.name AS label").
		Limit(DropdownMaxRows)

	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("ops_room_devices.name LIKE ?", "%"+name+"%")
	}
	if roomId := extractStringParam(params, "roomId"); roomId != "" {
		query = query.Where("ops_room_devices.room_id = ?", roomId)
	}
	if deviceType := extractStringParam(params, "deviceType"); deviceType != "" {
		query = query.Where("ops_room_devices.device_type = ?", deviceType)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("ops_room_devices.status = ?", status)
	}

	if err := query.Order("ops_room_devices.name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// validateRoom 验证机房存在性
func (s *roomDeviceService) validateRoom(ctx context.Context, roomID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsServerRoom{}).Where("id = ?", roomID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return apperrors.ServerRoomNotFound()
	}
	return nil
}
