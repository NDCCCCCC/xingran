package operations

import (
	"context"
	"fmt"

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
	Normal   int64 `json:"normal"`   // operations.RoomDeviceStatusNormal
	Fault    int64 `json:"fault"`    // operations.RoomDeviceStatusFault
	Scrapped int64 `json:"scrapped"` // operations.RoomDeviceStatusScrapped
}

// Statistics 统计机房设备(按 status 聚合,排除软删除)。
func (s *roomDeviceService) Statistics(ctx context.Context) (*RoomDeviceStatisticsResult, error) {
	var result RoomDeviceStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&operations.OpsRoomDevice{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS normal", int(operations.RoomDeviceStatusNormal)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS fault", int(operations.RoomDeviceStatusFault)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS scrapped", int(operations.RoomDeviceStatusScrapped)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type roomDeviceService struct {
	// repo 承接六方法 CRUD 管道（Phase 91-04 repo 化，D-02 纯 struct 组合）
	repo *base.GORMRepository[operations.OpsRoomDevice]
	// db 保留给 Statistics / SearchRoomDeviceOptions / validateRoom（D-04 留 service）
	db *gorm.DB
}

// NewRoomDeviceService 创建机房设备服务实例
func NewRoomDeviceService(db *gorm.DB) RoomDeviceService {
	return &roomDeviceService{
		repo: base.NewGORMRepository[operations.OpsRoomDevice](db),
		db:   db,
	}
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

	err := s.repo.Create(ctx, device)
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

	return s.repo.Update(ctx, device)
}

// Delete 删除机房设备
func (s *roomDeviceService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// GetByID 根据ID获取机房设备（JOIN 管道经 repo 执行；repo 经 Tabler 断言生成
// ops_room_devices.id = ?，与迁移前表限定逐字一致）。
func (s *roomDeviceService) GetByID(ctx context.Context, id string) (*operations.OpsRoomDevice, error) {
	return s.repo.GetByID(ctx, id, s.joinScope())
}

// joinScope List/GetByID 共用的 JOIN + Select scope（逐字平移，含 CAST 写法）。
//
// 迁移前 List 的 Joins 在 filter 之前、Select 在 Count 之后；repo 化后 joinScope
// 前置——两 JOIN 均为按主键 N:1 LEFT JOIN 无行增殖，Count 结果不变；Count 时
// 自定义 Select 被临时替换为 count(*) 并在 Find 恢复（TestBase91 契约锁定）。
// GetByID 迁移前仅 JOIN server_rooms——共用本 scope 后多一个按主键的 buildings
// LEFT JOIN（无行增殖、不取 buildings 列），返回数据一致。
func (s *roomDeviceService) joinScope() base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		// 先 JOIN 机房表和楼宇表，便于后续筛选和显示机房名称
		// 注意：server_rooms.building_id 是 varchar，需要转换为 uuid 才能与 buildings.id 比较
		return db.
			Joins("LEFT JOIN ops_server_rooms ON CAST(ops_server_rooms.id AS TEXT) = ops_room_devices.room_id").
			Joins("LEFT JOIN ops_buildings ON CAST(ops_buildings.id AS TEXT) = ops_server_rooms.building_id").
			// 选择字段（包括机房名称）
			Select("ops_room_devices.*, ops_server_rooms.name as room_name")
	}
}

// filterScope List 的过滤条件（条件字符串与 ? 占位符逐字平移自迁移前实现）。
// orgId 引用 JOIN 的 ops_buildings 列——必须后于 joinScope 应用（P8 顺序敏感）。
func (s *roomDeviceService) filterScope(req requests.RoomDeviceListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.Name != "" {
			db = db.Where("ops_room_devices.name LIKE ?", "%"+req.Name+"%")
		}
		if req.DeviceType != "" {
			db = db.Where("ops_room_devices.device_type = ?", req.DeviceType)
		}
		// 机房ID筛选优先
		if req.RoomID != "" {
			db = db.Where("ops_room_devices.room_id = ?", req.RoomID)
		}
		// 部门筛选：通过楼宇表筛选（设备 → 机房 → 楼宇 → 部门）
		if req.OrgID != "" {
			db = db.Where("ops_buildings.org_id = ?", req.OrgID)
		}
		if req.IPAddress != "" {
			db = db.Where("ops_room_devices.ip_address = ?", req.IPAddress)
		}
		if req.HasStatus() {
			db = db.Where("ops_room_devices.status = ?", req.GetStatus(0))
		}
		return db
	}
}

// List 查询机房设备列表（CRUD 管道经 base.GORMRepository 执行）。
//
// P8 顺序敏感（T-91-04-02 缓解）：joinScope 必须位于 filterScope 之前——
// orgId filter 引用 joined 表 ops_buildings 的列，顺序颠倒会把 filter 打在
// 未 join 的表列上（SQL 错误或语义漂移）。repo 按变参顺序应用 scopes。
func (s *roomDeviceService) List(ctx context.Context, req requests.RoomDeviceListRequest) (*PageResult, error) {
	current, pageSize := req.GetPagination()
	return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize},
		s.joinScope(),
		s.filterScope(req),
		base.SortScope(req.BaseListRequest, roomDeviceAllowedSortFields, "ops_room_devices.created_at DESC"),
	)
}

// BatchDelete 批量删除机房设备
func (s *roomDeviceService) BatchDelete(ctx context.Context, ids []string) error {
	return s.repo.BatchDelete(ctx, ids)
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
