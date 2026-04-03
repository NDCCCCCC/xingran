package operations

import (
	"context"

	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// ServerRoomService 机房服务接口
type ServerRoomService interface {
	Create(ctx context.Context, room *operations.OpsServerRoom) error
	Update(ctx context.Context, room *operations.OpsServerRoom) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsServerRoom, error)
	List(ctx context.Context, req requests.ServerRoomListRequest) (*PageResult, error)
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

// List 查询机房列表（类型安全版本）
func (s *serverRoomService) List(ctx context.Context, req requests.ServerRoomListRequest) (*PageResult, error) {
	var total int64
	var list []operations.OpsServerRoom

	query := s.db.WithContext(ctx).Model(&operations.OpsServerRoom{})

	// 添加筛选条件 - 类型安全，无需类型断言
	// 注意：使用表前缀避免与 JOIN 的表产生列名歧义
	if req.Name != "" {
		query = query.Where("ops_server_rooms.name LIKE ?", "%"+req.Name+"%")
	}
	if req.BuildingID != "" {
		query = query.Where("ops_server_rooms.building_id = ?", req.BuildingID)
	}
	if req.FloorID != "" {
		query = query.Where("ops_server_rooms.floor_id = ?", req.FloorID)
	}
	if req.HasStatus() {
		query = query.Where("ops_server_rooms.status = ?", req.GetStatus(0))
	}

	// 分页 - 使用请求结构体的方法
	offset := req.GetOffset()
	_, pageSize := req.GetPagination()
	current, _ := req.GetPagination()

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// JOIN 关联表获取楼宇名称和楼层名称（需要类型转换：varchar -> uuid）
	if err := query.
		Select("ops_server_rooms.*, b.name as building_name, f.name as floor_name, f.floor_no").
		Joins("LEFT JOIN ops_buildings b ON b.id::text = ops_server_rooms.building_id").
		Joins("LEFT JOIN ops_floors f ON f.id::text = ops_server_rooms.floor_id").
		Offset(offset).
		Limit(pageSize).
		Order("ops_server_rooms.created_at DESC").
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
		return apperrors.FloorNotFound()
	}
	return nil
}
