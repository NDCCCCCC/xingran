package operations

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
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
	// Statistics 专线统计(专用 COUNT 聚合,不依赖分页列表)。
	Statistics(ctx context.Context) (*DedicatedLineStatisticsResult, error)
	// SearchDedicatedLineOptions 专线下拉数据源(LIKE 模糊 + 类型/ISP/状态筛选,LIMIT 50)。
	SearchDedicatedLineOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error)
}

// DedicatedLineStatisticsResult 专线统计结果(status: 0=正常 1=故障 2=停用)。
type DedicatedLineStatisticsResult struct {
	Total    int64 `json:"total"`
	Normal   int64 `json:"normal"`   // operations.LineStatusNormal
	Fault    int64 `json:"fault"`    // operations.LineStatusFault
	Disabled int64 `json:"disabled"` // operations.LineStatusDisabled
}

// Statistics 统计专线(按 status 聚合,排除软删除)。
func (s *dedicatedLineService) Statistics(ctx context.Context) (*DedicatedLineStatisticsResult, error) {
	var result DedicatedLineStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&operations.OpsDedicatedLine{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS normal", int(operations.LineStatusNormal)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS fault", int(operations.LineStatusFault)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS disabled", int(operations.LineStatusDisabled)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type dedicatedLineService struct {
	// repo 承接六方法 CRUD 管道（Phase 91-04 repo 化，D-02 纯 struct 组合）
	repo *base.GORMRepository[operations.OpsDedicatedLine]
	// db 保留给 populateRoomNames 回填 / Statistics / SearchDedicatedLineOptions
	// （D-04 留 service 的直用场景）
	db *gorm.DB
}

// NewDedicatedLineService 创建专线服务实例
func NewDedicatedLineService(db *gorm.DB) DedicatedLineService {
	return &dedicatedLineService{
		repo: base.NewGORMRepository[operations.OpsDedicatedLine](db),
		db:   db,
	}
}

// dedicatedLineAllowedSortFields 专线可排序字段白名单(对应 ops_dedicated_lines 表列名)。
var dedicatedLineAllowedSortFields = map[string]string{
	"name":               "name",
	"lineType":           "line_type",
	"bandwidth":          "bandwidth",
	"isp":                "isp",
	"sourceDeviceName":   "source_device_name",
	"destDeviceName":     "dest_device_name",
	"carrierContactName": "carrier_contact_name",
	"status":             "status",
	"createdAt":          "created_at",
}

// Create 创建专线
func (s *dedicatedLineService) Create(ctx context.Context, line *operations.OpsDedicatedLine) error {
	if err := s.populateRoomNames(ctx, line); err != nil {
		return err
	}

	return s.repo.Create(ctx, line)
}

// Update 更新专线
func (s *dedicatedLineService) Update(ctx context.Context, line *operations.OpsDedicatedLine) error {
	if err := s.populateRoomNames(ctx, line); err != nil {
		return err
	}

	return s.repo.Update(ctx, line)
}

// Delete 删除专线
func (s *dedicatedLineService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// GetByID 根据ID获取专线
func (s *dedicatedLineService) GetByID(ctx context.Context, id string) (*operations.OpsDedicatedLine, error) {
	return s.repo.GetByID(ctx, id)
}

// filterScope List 管道的全部 11 个 filter（条件字符串、else-if 分支结构与
// ? 占位符逐字平移自迁移前实现）。
func (s *dedicatedLineService) filterScope(req requests.DedicatedLineListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.Name != "" {
			db = db.Where("name LIKE ?", "%"+req.Name+"%")
		}
		if req.LineType != "" {
			db = db.Where("line_type = ?", req.LineType)
		}
		if req.ISP != "" {
			db = db.Where("isp = ?", req.ISP)
		}
		// 机房ID筛选优先（更精确）
		if req.SourceRoomId != "" {
			db = db.Where("source_room_id = ?", req.SourceRoomId)
		} else if req.SourceRoomName != "" {
			db = db.Where("source_room_name LIKE ?", "%"+req.SourceRoomName+"%")
		}
		if req.DestRoomId != "" {
			db = db.Where("dest_room_id = ?", req.DestRoomId)
		} else if req.DestRoomName != "" {
			db = db.Where("dest_room_name LIKE ?", "%"+req.DestRoomName+"%")
		}
		if req.SourceDeviceName != "" {
			db = db.Where("source_device_name LIKE ?", "%"+req.SourceDeviceName+"%")
		}
		if req.DestDeviceName != "" {
			db = db.Where("dest_device_name LIKE ?", "%"+req.DestDeviceName+"%")
		}
		if req.CarrierContactName != "" {
			db = db.Where("carrier_contact_name LIKE ?", "%"+req.CarrierContactName+"%")
		}
		if req.HasStatus() {
			db = db.Where("status = ?", req.GetStatus(0))
		}
		return db
	}
}

// List 查询专线列表（CRUD 管道经 base.GORMRepository 执行；B 型排他默认排序）。
func (s *dedicatedLineService) List(ctx context.Context, req requests.DedicatedLineListRequest) (*PageResult, error) {
	current, pageSize := req.GetPagination()
	return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize},
		s.filterScope(req),
		base.SortScope(req.BaseListRequest, dedicatedLineAllowedSortFields, "created_at DESC"),
	)
}

// BatchDelete 批量删除专线
func (s *dedicatedLineService) BatchDelete(ctx context.Context, ids []string) error {
	return s.repo.BatchDelete(ctx, ids)
}

// SearchDedicatedLineOptions 专线下拉数据源(name LIKE 模糊 + lineType/ISP/sourceRoomId/destRoomId/status 筛选,LIMIT 50)。
// 与 List 同款 WHERE 语义;只 SELECT id+name 两列。
func (s *dedicatedLineService) SearchDedicatedLineOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	var result []DropdownOption

	query := s.db.WithContext(ctx).Table("ops_dedicated_lines").
		Select("id AS value, name AS label").
		Limit(DropdownMaxRows)

	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if lineType := extractStringParam(params, "lineType"); lineType != "" {
		query = query.Where("line_type = ?", lineType)
	}
	if isp := extractStringParam(params, "isp"); isp != "" {
		query = query.Where("isp = ?", isp)
	}
	if sourceRoomId := extractStringParam(params, "sourceRoomId"); sourceRoomId != "" {
		query = query.Where("source_room_id = ?", sourceRoomId)
	}
	if destRoomId := extractStringParam(params, "destRoomId"); destRoomId != "" {
		query = query.Where("dest_room_id = ?", destRoomId)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}
