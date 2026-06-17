package operations

import (
	"context"
	"fmt"
	"strings"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
	"gorm.io/gorm"
)

const (
	workstationTable      = "sys_workstation"
	floorTable            = "ops_floors"
	workstationJoinSelect = "sys_workstation.*, ops_floors.name as floor_name, ops_floors.floor_no as floor_code, ops_buildings.name as building_name, ops_buildings.id as building_id, sys_dept.dept_name as dept_name, sys_user.nickname as user_name"
	workstationJoinClause = "LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid LEFT JOIN ops_buildings ON ops_buildings.id = ops_floors.building_id::uuid LEFT JOIN sys_dept ON sys_dept.id::text = sys_workstation.dept_id LEFT JOIN sys_user ON sys_user.id::text = sys_workstation.user_id"
)

// validateTableName 验证表名是否在白名单中，防止 SQL 注入
func validateTableName(tableName string) bool {
	allowedTables := map[string]bool{
		"ops_buildings": true,
		"sys_buildings": true,
		"ops_floors":    true,
		"sys_floors":    true,
	}
	return allowedTables[tableName]
}

// WorkstationService 工位服务接口
type WorkstationService interface {
	Create(ctx context.Context, workstation *models.Workstation) error
	Update(ctx context.Context, workstation *models.Workstation) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Workstation, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	BatchUpdatePositions(ctx context.Context, items []PositionUpdateItem) error
}

type workstationService struct {
	db *gorm.DB
}

// workstationAllowedSortFields 工位可排序字段白名单。
// 因 List 有 LEFT JOIN floors,字段值带表别名。
var workstationAllowedSortFields = map[string]string{
	"workstationName": "sys_workstation.workstation_name",
	"workstationNo":   "sys_workstation.workstation_no",
	"status":          "sys_workstation.status",
	"workstationType": "sys_workstation.workstation_type",
	"createdAt":       "sys_workstation.created_at",
}

// PositionUpdateItem 位置更新项
type PositionUpdateItem struct {
	ID        string `json:"id"`
	PositionX int    `json:"positionX"`
	PositionY int    `json:"positionY"`
	Rotation  *int   `json:"rotation,omitempty"` // 旋转角度（可选）
	Width     *int   `json:"width,omitempty"`    // 工位宽度（毫米，可选）
	Depth     *int   `json:"depth,omitempty"`    // 工位深度（毫米，可选）
	DeskType  *int   `json:"deskType,omitempty"` // 桌型（可选）：0=一字型, 1=L型
}

// NewWorkstationService 创建工位服务实例
func NewWorkstationService(db *gorm.DB) WorkstationService {
	return &workstationService{
		db: db,
	}
}

// Create 创建工位
func (s *workstationService) Create(ctx context.Context, workstation *models.Workstation) error {
	return s.db.WithContext(ctx).Create(workstation).Error
}

// Update 更新工位
func (s *workstationService) Update(ctx context.Context, workstation *models.Workstation) error {
	return s.db.WithContext(ctx).Save(workstation).Error
}

func (s *workstationService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Table(workstationTable).Where("id = ?", id).Delete(&models.Workstation{}).Error
}

// GetByID 根据ID获取工位
func (s *workstationService) GetByID(ctx context.Context, id string) (*models.Workstation, error) {
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Select(workstationJoinSelect).
		Joins(workstationJoinClause).
		Where("sys_workstation.id = ?", id).
		First(&workstation).Error
	if err != nil {
		return nil, err
	}
	return &workstation, nil
}

// List 查询工位列表
func (s *workstationService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []models.Workstation

	query := s.db.WithContext(ctx).Model(&models.Workstation{})

	// 添加筛选条件
	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("sys_workstation.workstation_name LIKE ?", "%"+name+"%")
	}
	if floorId := extractStringParam(params, "floorId"); floorId != "" {
		query = query.Where("sys_workstation.floor_id = ?", floorId)
	}
	if floorCode := extractStringParam(params, "floorCode"); floorCode != "" {
		// 验证表名是否在白名单中，防止 SQL 注入
		if !validateTableName(floorTable) {
			return nil, fmt.Errorf("invalid table name: %s", floorTable)
		}
		query = query.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE "+floorTable+".id = sys_workstation.floor_id::uuid AND "+floorTable+".floor_no = ?)", floorCode)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("sys_workstation.status = ?", status)
	}
	if typeVal := extractIntParam(params, "type", -1); typeVal >= 0 {
		query = query.Where("sys_workstation.workstation_type = ?", typeVal)
	}
	// 通过关联楼宇的 orgId 筛选部门（包含子部门）
	if orgId := extractStringParam(params, "orgId"); orgId != "" {
		// 工位 -> 楼层 -> 楼宇，通过楼宇的 org_id 筛选
		// 支持查询该部门及其所有子部门的工位
		// 使用 EXISTS 子查询避免与现有 JOIN 冲突
		// 注意：需要类型转换
		// - ops_floors.building_id 是 varchar，ops_buildings.id 是 uuid
		// - ops_buildings.org_id 是 varchar，sys_dept.id 是 uuid
		// - 将两边都转为 text 进行比较，避免类型不匹配
		// 查询该部门及其所有子部门：ancestors 包含该部门ID，或 ID 等于该部门ID
		query = query.Where("EXISTS (SELECT 1 FROM ops_floors f JOIN ops_buildings b ON b.id = f.building_id::uuid JOIN sys_dept d ON d.id::text = b.org_id WHERE f.id::text = sys_workstation.floor_id AND (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?) AND b.deleted_at IS NULL)", orgId, "%,"+orgId, orgId)
	}

	// 分页
	pagination := extractPagination(params)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := calculateOffset(pagination)
	// JOIN ops_floors 表获取 floor_name（类型转换：varchar -> uuid）
	// 用户排序(白名单,带表别名);无 OrderByColumn 时保留原默认
	sortReq := extractSortRequest(params)
	query = base.ApplySort(query, sortReq, workstationAllowedSortFields)
	if sortReq.OrderByColumn == "" {
		query = query.Order("sys_workstation.created_at DESC")
	}
	if err := query.
		Select(workstationJoinSelect).
		Joins(workstationJoinClause).
		Offset(offset).
		Limit(pagination.PageSize).
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  pagination.Current,
		PageSize: pagination.PageSize,
	}, nil
}

func (s *workstationService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Table(workstationTable).Where("id IN ?", ids).Delete(&models.Workstation{}).Error
}

// BatchUpdatePositions 批量更新工位位置和尺寸
// 遵循 Go 最佳实践：使用 CASE WHEN 批量更新，避免 N+1 问题
func (s *workstationService) BatchUpdatePositions(ctx context.Context, items []PositionUpdateItem) error {
	if len(items) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// F-19: 重构为 GORM 占位符参数化查询,彻底消除 SQL 注入。
		// 原代码用 fmt.Sprintf 直接拼接 item.ID (UUID 字符串若含 ' 即可注入)
		// 与 PositionX/Y 等整数字段。本实现按 SQL 中 ? 的出现顺序顺序追加 args。

		ids := make([]string, 0, len(items))
		args := make([]interface{}, 0, len(items)*12)

		// 必填: position_x / position_y 的 CASE WHEN
		var positionXClause, positionYClause strings.Builder
		positionXClause.WriteString("position_x = CASE id ")
		positionYClause.WriteString("position_y = CASE id ")
		for _, item := range items {
			ids = append(ids, item.ID)
			positionXClause.WriteString("WHEN ? THEN ? ")
		}
		positionXClause.WriteString("END")
		for range items {
			positionYClause.WriteString("WHEN ? THEN ? ")
		}
		positionYClause.WriteString("END")

		// 按 SQL 模板中 ? 的出现顺序追加 args
		for _, item := range items {
			args = append(args, item.ID, item.PositionX)
		}
		for _, item := range items {
			args = append(args, item.ID, item.PositionY)
		}

		setClauses := []string{positionXClause.String(), positionYClause.String()}

		// 可选字段:rotation / width / depth / desk_type
		// 提取生成 CASE 子句和 args 的公共逻辑
		appendOptionalClause := func(col string, valuer func(PositionUpdateItem) (int, bool)) {
			var b strings.Builder
			b.WriteString(col + " = CASE id ")
			added := 0
			for _, item := range items {
				if v, ok := valuer(item); ok {
					_ = v
					b.WriteString("WHEN ? THEN ? ")
					added++
				}
			}
			if added == 0 {
				return
			}
			b.WriteString("ELSE " + col + " END")
			setClauses = append(setClauses, b.String())
			for _, item := range items {
				if v, ok := valuer(item); ok {
					args = append(args, item.ID, v)
				}
			}
		}
		appendOptionalClause("rotation", func(it PositionUpdateItem) (int, bool) {
			if it.Rotation == nil {
				return 0, false
			}
			return *it.Rotation, true
		})
		appendOptionalClause("width", func(it PositionUpdateItem) (int, bool) {
			if it.Width == nil {
				return 0, false
			}
			return *it.Width, true
		})
		appendOptionalClause("depth", func(it PositionUpdateItem) (int, bool) {
			if it.Depth == nil {
				return 0, false
			}
			return *it.Depth, true
		})
		appendOptionalClause("desk_type", func(it PositionUpdateItem) (int, bool) {
			if it.DeskType == nil {
				return 0, false
			}
			return *it.DeskType, true
		})

		// WHERE id IN (?) — GORM 接受 []string 参数展开
		sql := fmt.Sprintf("UPDATE sys_workstation SET %s WHERE id IN ?", strings.Join(setClauses, ", "))
		args = append(args, ids)

		if err := tx.Exec(sql, args...).Error; err != nil {
			return fmt.Errorf("批量更新工位位置失败: %w", err)
		}

		return nil
	})
}
