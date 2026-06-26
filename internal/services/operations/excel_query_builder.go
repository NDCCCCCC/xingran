package operations

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type QueryBuilder interface {
	BuildQuery(ctx context.Context, db *gorm.DB, config ExcelExportConfig, params map[string]interface{}) *gorm.DB
}

type QueryBuilderFactory interface {
	GetQueryBuilder(name string) (QueryBuilder, bool)
}

type queryBuilderFactoryImpl struct {
	db *gorm.DB
}

func NewQueryBuilderFactory(db *gorm.DB) QueryBuilderFactory {
	return &queryBuilderFactoryImpl{db: db}
}

func (f *queryBuilderFactoryImpl) GetQueryBuilder(name string) (QueryBuilder, bool) {
	switch name {
	case "department":
		return NewDepartmentQueryBuilder(), true
	case "workstation":
		return NewWorkstationQueryBuilder(), true
	default:
		return NewDefaultQueryBuilder(), true
	}
}

type DefaultQueryBuilder struct{}

func NewDefaultQueryBuilder() *DefaultQueryBuilder {
	return &DefaultQueryBuilder{}
}

func (b *DefaultQueryBuilder) BuildQuery(
	ctx context.Context,
	db *gorm.DB,
	config ExcelExportConfig,
	params map[string]interface{},
) *gorm.DB {
	query := db.Table(config.TableName).Where("deleted_at IS NULL")

	// 支持动态筛选条件
	for paramKey, paramValue := range params {
		if paramValue == nil || paramValue == "" {
			continue
		}

		// 查找配置的字段映射
		dbField, hasMapping := config.FilterMapping[paramKey]
		if !hasMapping {
			// 如果没有映射，尝试直接使用参数名作为字段名
			dbField = paramKey
		}

		// 根据值类型构建查询条件
		switch v := paramValue.(type) {
		case string:
			// 字符串使用 LIKE 查询
			query = query.Where(dbField+" LIKE ?", "%"+v+"%")
		case int, int64, float64:
			// 数字使用精确匹配
			query = query.Where(dbField+" = ?", v)
		case bool:
			query = query.Where(dbField+" = ?", v)
		case []interface{}:
			// 数组使用 IN 查询
			if len(v) > 0 {
				query = query.Where(dbField+" IN ?", v)
			}
		}
	}

	return query
}

type DepartmentQueryBuilder struct{}

func NewDepartmentQueryBuilder() *DepartmentQueryBuilder {
	return &DepartmentQueryBuilder{}
}

func (b *DepartmentQueryBuilder) BuildQuery(
	ctx context.Context,
	db *gorm.DB,
	config ExcelExportConfig,
	params map[string]interface{},
) *gorm.DB {
	query := db.Table(config.TableName).Where("deleted_at IS NULL")

	if name, ok := params["name"]; ok && name != "" {
		if nameStr, ok := name.(string); ok {
			query = query.Where("dept_name LIKE ?", "%"+nameStr+"%")
		}
	}
	if code, ok := params["code"]; ok && code != "" {
		if codeStr, ok := code.(string); ok {
			query = query.Where("dept_code LIKE ?", "%"+codeStr+"%")
		}
	}
	if status, ok := params["status"]; ok && status != nil {
		query = query.Where("status = ?", status)
	}

	return query
}

// WorkstationQueryBuilder 镜像生产 List 查询（workstationJoinClause），
// 用显式 LEFT JOIN + ::uuid/::text 转换一次性带出 楼层/楼宇/部门/人员 名称，
// 并用相关子查询取主设备的 序列号/名称/型号。
// 彻底绕开 resolveAssociations 的 uuid/text 类型坑（config 无 Join 时其提前返回）。
type WorkstationQueryBuilder struct{}

func NewWorkstationQueryBuilder() *WorkstationQueryBuilder {
	return &WorkstationQueryBuilder{}
}

func (b *WorkstationQueryBuilder) BuildQuery(
	ctx context.Context,
	db *gorm.DB,
	config ExcelExportConfig,
	params map[string]interface{},
) *gorm.DB {
	query := db.Table("sys_workstation").
		Select(`sys_workstation.id,
			sys_workstation.workstation_name,
			sys_workstation.workstation_type,
			sys_workstation.status,
			sys_workstation.location,
			sys_workstation.description,
			sys_workstation.dept_id,
			sys_workstation.user_id,
			sys_workstation.floor_id,
			sys_workstation.building_id,
			sys_workstation.created_at,
			sys_workstation.updated_at,
			ops_floors.name AS floor_name,
			ops_buildings.name AS building_name,
			sys_dept.dept_name AS dept_name,
			sys_user.nickname AS user_name,
			(SELECT device_serial FROM ops_workstation_device
				WHERE workstation_id = sys_workstation.id::uuid AND deleted_at IS NULL
				ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC
				LIMIT 1) AS device_serial,
			(SELECT device_name FROM ops_workstation_device
				WHERE workstation_id = sys_workstation.id::uuid AND deleted_at IS NULL
				ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC
				LIMIT 1) AS device_name,
			(SELECT device_model FROM ops_workstation_device
				WHERE workstation_id = sys_workstation.id::uuid AND deleted_at IS NULL
				ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC
				LIMIT 1) AS device_model`).
		Joins("LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = sys_workstation.building_id::uuid").
		Joins("LEFT JOIN sys_dept ON sys_dept.id::text = sys_workstation.dept_id").
		Joins("LEFT JOIN sys_user ON sys_user.id::text = sys_workstation.user_id").
		Where("sys_workstation.deleted_at IS NULL")

	// 复用 config.FilterMapping 应用筛选（逻辑与 DefaultQueryBuilder 等价，
	// 仅在字段无表别名前缀时补 sys_workstation.）
	for paramKey, paramValue := range params {
		if paramValue == nil || paramValue == "" {
			continue
		}

		dbField, hasMapping := config.FilterMapping[paramKey]
		if !hasMapping {
			dbField = paramKey
		}
		if !strings.Contains(dbField, ".") {
			dbField = "sys_workstation." + dbField
		}

		switch v := paramValue.(type) {
		case string:
			query = query.Where(dbField+" LIKE ?", "%"+v+"%")
		case int, int64, float64:
			query = query.Where(dbField+" = ?", v)
		case bool:
			query = query.Where(dbField+" = ?", v)
		case []interface{}:
			if len(v) > 0 {
				query = query.Where(dbField+" IN ?", v)
			}
		}
	}

	return query
}
