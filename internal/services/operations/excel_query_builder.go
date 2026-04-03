package operations

import (
	"context"

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
