package operations

import (
	"context"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

const (
	workstationTable      = "sys_workstation"
	floorTable            = "ops_floors"
	workstationJoinSelect = "sys_workstation.*, ops_floors.name as floor_name, ops_floors.floor_no as floor_code, ops_buildings.name as building_name, ops_buildings.id as building_id, sys_dept.dept_name as dept_name, sys_user.nickname as user_name, (SELECT device_serial FROM ops_workstation_device WHERE workstation_id = sys_workstation.id AND deleted_at IS NULL AND is_primary = true ORDER BY priority DESC, created_at ASC LIMIT 1) as primary_device_serial"
	// uuid 列(varchar 外键列对 uuid 主键)用标准 SQL CAST(... AS TEXT) 统一转 text 比较,
	// PG/SQLite 双方言行为一致(PG 专有 ::uuid/::text 在 SQLite 报 unrecognized token ':')。
	// 与 GetWorkstationDeptOptions / location_alias_service 的 CAST 范式一致。
	workstationJoinClause = "LEFT JOIN ops_floors ON CAST(ops_floors.id AS TEXT) = sys_workstation.floor_id LEFT JOIN ops_buildings ON CAST(ops_buildings.id AS TEXT) = ops_floors.building_id LEFT JOIN sys_dept ON CAST(sys_dept.id AS TEXT) = sys_workstation.dept_id LEFT JOIN sys_user ON CAST(sys_user.id AS TEXT) = sys_workstation.user_id"
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
	// Statistics 工位统计(专用 COUNT 聚合,不依赖分页列表;支持 orgId 部门筛选含子部门,替代前端 4 次 list 拼 total)。
	Statistics(ctx context.Context, params map[string]interface{}) (*WorkstationStatisticsResult, error)
	// GetWorkstationDeptOptions 工位编辑"所属部门"下拉数据源(orgId 子孙 + alias union)
	GetWorkstationDeptOptions(ctx context.Context, orgId string) ([]DeptOption, error)
	// SearchWorkstationOptions 工位下拉数据源(LIKE 模糊 + 多维筛选,LIMIT 50)。
	// 设计给前端 Select/AutoComplete 远程搜索。workstation 高频变更,不缓存。
	SearchWorkstationOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error)
}

// DeptOption 工位编辑"所属部门"下拉选项(union: orgId 子孙 + alias 映射)
type DeptOption struct {
	DeptID   string `json:"deptId"`   // 部门 ID (UUID text)
	DeptName string `json:"deptName"` // 部门名称 (短名,由前端 trimTitleToLastSegment 进一步收窄)
	IsAlias  bool   `json:"isAlias"`  // 是否为 alias 映射条目; true → 前端追加 [映射] 后缀
}

// WorkstationStatisticsResult 工位统计结果(status: 0=可用 1=占用 2=维护)。
type WorkstationStatisticsResult struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"` // models.WorkstationStatusAvailable
	Occupied  int64 `json:"occupied"`  // models.WorkstationStatusOccupied
	Maintain  int64 `json:"maintain"`  // models.WorkstationStatusMaintain
}

// Statistics 统计工位(按 status 聚合,排除软删除;支持 orgId 部门筛选含子部门,与 List orgId 过滤一致)。
func (s *workstationService) Statistics(ctx context.Context, params map[string]interface{}) (*WorkstationStatisticsResult, error) {
	var result WorkstationStatisticsResult
	query := s.db.WithContext(ctx).Model(&models.Workstation{})
	if orgId := extractStringParam(params, "orgId"); orgId != "" {
		query = query.Where("EXISTS (SELECT 1 FROM ops_floors f JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id JOIN sys_dept d ON CAST(d.id AS TEXT) = b.org_id WHERE CAST(f.id AS TEXT) = sys_workstation.floor_id AND (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?) AND b.deleted_at IS NULL)", orgId, "%,"+orgId, orgId)
	}
	err := query.
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS available", int(models.WorkstationStatusAvailable)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS occupied", int(models.WorkstationStatusOccupied)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS maintain", int(models.WorkstationStatusMaintain)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetWorkstationDeptOptions 工位编辑"所属部门"下拉数据源 (D-06 单 query union)
// union: orgId 子孙节点 (is_alias=false) + alias 命中节点 (is_alias=true)
func (s *workstationService) GetWorkstationDeptOptions(ctx context.Context, orgId string) ([]DeptOption, error) {
	if orgId == "" {
		return []DeptOption{}, nil
	}
	var result []DeptOption
	// 单 query union: sys_dept 子孙节点 (is_alias=false) + alias 命中节点 (is_alias=true)
	// 1) 子孙节点: ancestors LIKE '%,<orgId>,%' OR id = orgId, 排除外部机构本身
	// 2) alias 节点: scope='workstation' AND location_id=orgId, JOIN sys_dept 取 dept_name
	//
	// 关于 sys_dept.id (uuid) 与 alias.dept_id/location_id (varchar) 的列对列比较:
	// 用标准 SQL CAST(... AS TEXT) 而非 PG 专有 ::text。
	//   - PG: CAST(uuid AS TEXT) 把 uuid 转成 text,与 varchar 比较(text/varchar 同族有 =)
	//   - SQLite: uuid 列实际存为 TEXT,CAST 是 no-op,TEXT = TEXT 正常
	// 故 CAST 写法在 PG/SQLite 双 DB 行为一致(避免 PG 专有 ::text 在 SQLite 语法错误)。
	// 与 location_alias_service.go 的 aliasListJoinClause 处理方式一致。
	err := s.db.WithContext(ctx).Raw(`
		SELECT CAST(id AS TEXT) AS dept_id, dept_name AS dept_name, false AS is_alias
		FROM sys_dept
		WHERE deleted_at IS NULL
		  AND is_external_org = 0
		  AND ((',' || ancestors || ',') LIKE ('%,' || ? || ',%') OR CAST(id AS TEXT) = ?)
		UNION ALL
		SELECT a.dept_id, d.dept_name, true AS is_alias
		FROM sys_dept_location_alias a
		JOIN sys_dept d ON CAST(d.id AS TEXT) = a.dept_id
		WHERE a.deleted_at IS NULL
		  AND a.scope = 'workstation'
		  AND a.location_id = ?
	`, orgId, orgId, orgId).Scan(&result).Error
	return result, err
}

type workstationService struct {
	db *gorm.DB
}

// workstationAllowedSortFields 工位可排序字段白名单。
// 因 List 有 LEFT JOIN floors/buildings/dept/user,字段值带表别名。
// key 须与前端列 dataIndex 一致(useServerSort 按 sorter.field === dataIndex 匹配 sorterMeta)。
//
// 注:workstationNo 列在 legacy-2026-06-15/034_remove_workstation_code_and_capacity.sql
// 已删除,白名单不再映射 — 否则客户端发 orderByColumn=workstationNo 会触发
// SQLSTATE 42703 (column workstation_no does not exist)。
var workstationAllowedSortFields = map[string]string{
	"workstationName": "sys_workstation.workstation_name",
	"name":            "sys_workstation.workstation_name", // 与前端 dataIndex="name" 对齐
	"status":          "sys_workstation.status",
	"workstationType": "sys_workstation.workstation_type",
	"type":            "sys_workstation.workstation_type", // 与前端 dataIndex="type" 对齐
	"buildingName":    "ops_buildings.name",
	"floorName":       "ops_floors.name",
	"deptName":        "sys_dept.dept_name",
	"userName":        "sys_user.nickname",
	"createdAt":       "sys_workstation.created_at",
	"updatedAt":       "sys_workstation.updated_at",
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
//
// 新建工位默认 status=0（空闲）。如果用户在创建时同时指定了 user_id，则
// applyWorkstationOccupancyLink 会把 status 联动为 1（占用）。Maintain 状态
// 仅由管理员手动设置，新建路径不会触发（因为新工位入库前 status 来自表单，
// 通常是 Available）。
func (s *workstationService) Create(ctx context.Context, workstation *models.Workstation) error {
	// ✨ 联动 user_id ↔ status(占用/空闲)
	applyWorkstationOccupancyLink(workstation, workstation.Status)
	return s.db.WithContext(ctx).Create(workstation).Error
}

// Update 更新工位
//
// 实现说明(handler 反序列化的请求体只含表单字段,不含 createdAt/updatedAt):
// handler 把请求体反序列化到一个零值 models.Workstation,其 BaseModel.CreatedAt
// 是非指针 time.Time(见 internal/models/base.go),零值即 0001-01-01。若直接 Save,
// GORM 会用零值全量覆盖 created_at,导致工位创建时间被清零。
// 修复:Save 前先 First 查出现有记录,把审计字段 CreatedAt/CreatedBy 回填到入参,
// 再 Save。参考 internal/services/system/user_service.go 的 Update 模式。
func (s *workstationService) Update(ctx context.Context, workstation *models.Workstation) error {
	var existing models.Workstation
	if err := s.db.WithContext(ctx).Where("id = ?", workstation.ID).First(&existing).Error; err != nil {
		return err
	}
	// 保留创建时间和创建人,避免 Save 用零值覆盖(0001-01-01 bug 的根因)
	workstation.CreatedAt = existing.CreatedAt
	workstation.CreatedBy = existing.CreatedBy

	// ✨ 联动 user_id ↔ status(占用/空闲),Maintain(2) 保留维护语义
	applyWorkstationOccupancyLink(workstation, existing.Status)

	return s.db.WithContext(ctx).Save(workstation).Error
}

// applyWorkstationOccupancyLink 联动工位状态与所属人员。
//
// 规则:
//   - 如果 user_id 有值 → status = Occupied(1)
//   - 如果 user_id 为空 → status = Available(0)
//   - 如果当前 status = Maintain(2)，**保留维护状态不变**(运维流程强语义)
//
// 调用入口: Create + Update(在 Save 前)。Excel 导入路径单独在
// excel_service.prepareRecordsForUpsert 里覆盖 status(参见 n71)。
func applyWorkstationOccupancyLink(w *models.Workstation, currentStatus models.WorkstationStatus) {
	if currentStatus == models.WorkstationStatusMaintain {
		// 维护状态不受 user_id 联动影响
		return
	}
	if w.UserID != nil && *w.UserID != "" {
		w.Status = models.WorkstationStatusOccupied
	} else {
		w.Status = models.WorkstationStatusAvailable
	}
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
		query = query.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE CAST("+floorTable+".id AS TEXT) = sys_workstation.floor_id AND "+floorTable+".floor_no = ?)", floorCode)
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
		query = query.Where("EXISTS (SELECT 1 FROM ops_floors f JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id JOIN sys_dept d ON CAST(d.id AS TEXT) = b.org_id WHERE CAST(f.id AS TEXT) = sys_workstation.floor_id AND (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?) AND b.deleted_at IS NULL)", orgId, "%,"+orgId, orgId)
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

// SearchWorkstationOptions 工位下拉数据源(name LIKE 模糊 + floorId/floorCode/status/type/orgId 多维筛选,LIMIT 50)。
// 设计给前端 Select/AutoComplete 远程搜索(原 bug 位置: info-points/index.tsx 「所属工位」下拉)。
//
// 工位高频变更(占用/释放/调位),不缓存避免 staleness。
func (s *workstationService) SearchWorkstationOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	var result []DropdownOption

	query := s.db.WithContext(ctx).Table("sys_workstation").
		Select("sys_workstation.id AS value, sys_workstation.workstation_name AS label").
		Limit(DropdownMaxRows)

	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("sys_workstation.workstation_name LIKE ?", "%"+name+"%")
	}
	if floorId := extractStringParam(params, "floorId"); floorId != "" {
		query = query.Where("sys_workstation.floor_id = ?", floorId)
	}
	if floorCode := extractStringParam(params, "floorCode"); floorCode != "" {
		if !validateTableName(floorTable) {
			return nil, fmt.Errorf("invalid table name: %s", floorTable)
		}
		query = query.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE CAST("+floorTable+".id AS TEXT) = sys_workstation.floor_id AND "+floorTable+".floor_no = ?)", floorCode)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("sys_workstation.status = ?", status)
	}
	if typeVal := extractIntParam(params, "type", -1); typeVal >= 0 {
		query = query.Where("sys_workstation.workstation_type = ?", typeVal)
	}
	// orgId 部门筛选含子部门:与 List 同款 EXISTS 子查询,避免类型转换问题
	if orgId := extractStringParam(params, "orgId"); orgId != "" {
		query = query.Where("EXISTS (SELECT 1 FROM ops_floors f JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id JOIN sys_dept d ON CAST(d.id AS TEXT) = b.org_id WHERE CAST(f.id AS TEXT) = sys_workstation.floor_id AND (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?) AND b.deleted_at IS NULL)", orgId, "%,"+orgId, orgId)
	}

	if err := query.Order("sys_workstation.workstation_name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}
