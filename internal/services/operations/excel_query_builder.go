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

	if name, ok := params["name"]; ok && name != "" {
		if nameStr, ok := name.(string); ok {
			if fieldName, exists := config.FilterMapping["name"]; exists {
				query = query.Where(fieldName+" LIKE ?", "%"+nameStr+"%")
			} else {
				query = query.Where("name LIKE ?", "%"+nameStr+"%")
			}
		}
	}
	if code, ok := params["code"]; ok && code != "" {
		if codeStr, ok := code.(string); ok {
			if fieldName, exists := config.FilterMapping["code"]; exists {
				query = query.Where(fieldName+" LIKE ?", "%"+codeStr+"%")
			} else {
				query = query.Where("code LIKE ?", "%"+codeStr+"%")
			}
		}
	}
	if status, ok := params["status"]; ok && status != nil {
		query = query.Where("status = ?", status)
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
