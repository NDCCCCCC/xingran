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

// DepartmentQueryBuilder 部门导出查询构建器
//
// 用 PostgreSQL 递归 CTE 一次性算每个部门的完整父级链路(dept_path_cte),
// 对同名部门(例: 多个 "个人营销业务销售部" 挂在不同市中心支公司下)
// 在导出表里也能通过 parent_path 列区分.
//
// recursive CTE 模式参照项目内三处先例:
//   - pkg/permission/service.go:184              (菜单祖先扩展)
//   - internal/services/notice_target_service.go:119 (部门树下钻)
//   - internal/services/system/menu_service.go:147 (后代批量删除)
//
// 模式: base case (parent_id IS NULL) UNION ALL recursive case (parent + child),
//       再用 DISTINCT ON (id) ORDER BY depth DESC 取每个部门的最深一级链(即 root → … → 当前).
type DepartmentQueryBuilder struct{}

func NewDepartmentQueryBuilder() *DepartmentQueryBuilder {
	return &DepartmentQueryBuilder{}
}

// deptPathCTE 完整的递归 CTE,导出所有部门的"根 → … → 当前"链路.
//
// " → " (U+2192 空格) 用作分隔符,Excel 中显示比 "/" 更清晰,
// 也避免与 Phase 39 location alias 的 "/" 分隔符混淆.
//
// 跨方言设计 (PostgreSQL 生产 + SQLite 单测共用):
//   - PG ≥ 9.1 与 SQLite ≥ 3.8.3 均支持 WITH RECURSIVE + UNION ALL
//   - 取"每部门最深链"用相关子查询 ORDER BY depth DESC LIMIT 1,
//     避免 PostgreSQL 专有 DISTINCT ON,保证 SQLite 单测可跑通同一段 SQL.
//
// 性能: PG 11+ 会对相关子查询做 semi-join 优化,大表耗时与 LATERAL JOIN 相近.
//
// 递归方向必须正确 (Phase 工位部门映射 P3 / 单测复盘):
//   - base case: 根部门的 path_text 是自身名
//   - recursive step: 对父级已经在 CTE 的子部门,把"自身名"追加到父级已积累的链尾,
//     而不是把父名拼在前面 — 后者会把链路倒着算,且会双倍根名.
//     正确写法: path_text = dpc.path_text || ' → ' || c.dept_name
//               depth     = dpc.depth + 1
const deptPathCTE = `
WITH RECURSIVE dept_path_cte AS (
    -- base case: 根部门(parent_id 为 NULL),path 起点就是自身名
    SELECT id, parent_id, dept_name AS path_text, 0 AS depth
    FROM sys_dept
    WHERE parent_id IS NULL

    UNION ALL

    -- recursive case: c 是 sys_dept 中 parent_id 指向 CTE 已有行的子部门
    -- 把 c 自己名字追加到父级已积累的链尾, depth+1
    SELECT c.id, c.parent_id,
           dpc.path_text || ' → ' || c.dept_name,
           dpc.depth + 1
    FROM sys_dept c
    JOIN dept_path_cte dpc ON dpc.id = c.parent_id
)`

func (b *DepartmentQueryBuilder) BuildQuery(
	ctx context.Context,
	db *gorm.DB,
	config ExcelExportConfig,
	params map[string]interface{},
) *gorm.DB {
	// 主查询: sys_dept 所有列 + 每个部门的"最深深度链路字符串"作 parent_path
	// 相关子查询 ORDER BY depth DESC LIMIT 1 = 取该部门在 CTE 中最深的一行,
	// 即"根 → … → 当前"完整链. 跨方言 (PG/SQLite) 兼容.
	sql := deptPathCTE + `
SELECT d.*,
       (SELECT path_text
        FROM dept_path_cte
        WHERE id = d.id
        ORDER BY depth DESC
        LIMIT 1) AS parent_path
FROM sys_dept d
WHERE d.deleted_at IS NULL`

	args := make([]interface{}, 0, 3)

	// 动态筛选: 与原 DefaultQueryBuilder 同样语义(LIKE / IN),所有改动
	// 只发生在 WHERE 子句和 args slice 上,SQL 主体保持静态(便于参数化与缓存).
	if name, ok := params["name"]; ok && name != "" {
		if nameStr, ok := name.(string); ok {
			sql += " AND d.dept_name LIKE ?"
			args = append(args, "%"+nameStr+"%")
		}
	}
	if code, ok := params["code"]; ok && code != "" {
		if codeStr, ok := code.(string); ok {
			sql += " AND d.dept_code LIKE ?"
			args = append(args, "%"+codeStr+"%")
		}
	}
	if status, ok := params["status"]; ok && status != nil {
		sql += " AND d.status = ?"
		args = append(args, status)
	}

	sql += " ORDER BY d.dept_code"

	// db.Raw() 返回 *gorm.DB,可被下游 .Find(&data) 正常消费.
	// 与项目内三处 recursive CTE 先例 (pkg/permission/service.go:184 等)
	// 风格一致 — SQL 内联, 不引 clause.With / .With().
	return db.WithContext(ctx).Raw(sql, args...)
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
				WHERE workstation_id = sys_workstation.id::text AND deleted_at IS NULL
				ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC
				LIMIT 1) AS device_serial,
			(SELECT device_name FROM ops_workstation_device
				WHERE workstation_id = sys_workstation.id::text AND deleted_at IS NULL
				ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC
				LIMIT 1) AS device_name,
			(SELECT device_model FROM ops_workstation_device
				WHERE workstation_id = sys_workstation.id::text AND deleted_at IS NULL
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
