// Package query 提供通用的查询构建和分页辅助函数
package query

import (
	"fmt"

	"gorm.io/gorm"
)

// Condition 查询条件
type Condition struct {
	Field    string      // 字段名
	Operator string      // 操作符: "=", "!=", ">", ">=", "<", "<=", "LIKE", "IN", "NOT IN"
	Value    interface{} // 值
}

// Direction 排序方向
type Direction string

const (
	// ASC 升序
	ASC Direction = "ASC"
	// DESC 降序
	DESC Direction = "DESC"
)

// OrderBy 排序条件
type OrderBy struct {
	Field     string    // 字段名
	Direction Direction // 排序方向
}

// QueryBuilder 查询构建器
type QueryBuilder struct {
	db             *gorm.DB
	conditions     []Condition
	orConditions   [][]Condition // OR 条件组
	orderBy        []OrderBy
	preloads       []string // 预加载关联
	preloadsWith   []preloadConfig
	excludeDeleted bool
}

type preloadConfig struct {
	field    string
	callback func(*gorm.DB) *gorm.DB
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(db *gorm.DB) *QueryBuilder {
	return &QueryBuilder{
		db:             db,
		conditions:     make([]Condition, 0),
		orConditions:   make([][]Condition, 0),
		orderBy:        make([]OrderBy, 0),
		preloads:       make([]string, 0),
		preloadsWith:   make([]preloadConfig, 0),
		excludeDeleted: true,
	}
}

// Where 添加查询条件
func (qb *QueryBuilder) Where(field string, operator string, value interface{}) *QueryBuilder {
	if value != nil && value != "" {
		qb.conditions = append(qb.conditions, Condition{
			Field:    field,
			Operator: operator,
			Value:    value,
		})
	}
	return qb
}

// WhereEqual 添加等于条件
func (qb *QueryBuilder) WhereEqual(field string, value interface{}) *QueryBuilder {
	return qb.Where(field, "=", value)
}

// WhereNotEqual 添加不等于条件
func (qb *QueryBuilder) WhereNotEqual(field string, value interface{}) *QueryBuilder {
	return qb.Where(field, "!=", value)
}

// WhereLike 添加模糊匹配条件
func (qb *QueryBuilder) WhereLike(field string, value string) *QueryBuilder {
	if value != "" {
		qb.conditions = append(qb.conditions, Condition{
			Field:    field,
			Operator: "LIKE",
			Value:    "%" + value + "%",
		})
	}
	return qb
}

// WhereIn 添加 IN 条件
func (qb *QueryBuilder) WhereIn(field string, values []interface{}) *QueryBuilder {
	if len(values) > 0 {
		qb.conditions = append(qb.conditions, Condition{
			Field:    field,
			Operator: "IN",
			Value:    values,
		})
	}
	return qb
}

// WhereNotIn 添加 NOT IN 条件
func (qb *QueryBuilder) WhereNotIn(field string, values []interface{}) *QueryBuilder {
	if len(values) > 0 {
		qb.conditions = append(qb.conditions, Condition{
			Field:    field,
			Operator: "NOT IN",
			Value:    values,
		})
	}
	return qb
}

// WhereGreaterThan 添加大于条件
func (qb *QueryBuilder) WhereGreaterThan(field string, value interface{}) *QueryBuilder {
	return qb.Where(field, ">", value)
}

// WhereGreaterThanOrEqual 添加大于等于条件
func (qb *QueryBuilder) WhereGreaterThanOrEqual(field string, value interface{}) *QueryBuilder {
	return qb.Where(field, ">=", value)
}

// WhereLessThan 添加小于条件
func (qb *QueryBuilder) WhereLessThan(field string, value interface{}) *QueryBuilder {
	return qb.Where(field, "<", value)
}

// WhereLessThanOrEqual 添加小于等于条件
func (qb *QueryBuilder) WhereLessThanOrEqual(field string, value interface{}) *QueryBuilder {
	return qb.Where(field, "<=", value)
}

// WhereOrNull 添加字段等于某值或为NULL的条件
func (qb *QueryBuilder) WhereOrNull(field string, value interface{}) *QueryBuilder {
	if value != nil && value != "" {
		qb.orConditions = append(qb.orConditions, []Condition{
			{Field: field, Operator: "=", Value: value},
			{Field: field, Operator: "=", Value: nil},
		})
	}
	return qb
}

// OrWhere 添加 OR 条件组
func (qb *QueryBuilder) OrWhere(conditions []Condition) *QueryBuilder {
	if len(conditions) > 0 {
		qb.orConditions = append(qb.orConditions, conditions)
	}
	return qb
}

// OrderByField 添加排序
func (qb *QueryBuilder) OrderByField(field string, direction Direction) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, OrderBy{
		Field:     field,
		Direction: direction,
	})
	return qb
}

// Preload 添加预加载关联
func (qb *QueryBuilder) Preload(field string) *QueryBuilder {
	qb.preloads = append(qb.preloads, field)
	return qb
}

// PreloadWith 添加带回调的预加载关联
func (qb *QueryBuilder) PreloadWith(field string, callback func(*gorm.DB) *gorm.DB) *QueryBuilder {
	qb.preloadsWith = append(qb.preloadsWith, preloadConfig{
		field:    field,
		callback: callback,
	})
	return qb
}

// ExcludeDeleted 设置是否排除已删除记录
func (qb *QueryBuilder) ExcludeDeleted(exclude bool) *QueryBuilder {
	qb.excludeDeleted = exclude
	return qb
}

// Build 构建查询
func (qb *QueryBuilder) Build(model interface{}) *gorm.DB {
	query := qb.db.Model(model)

	// 应用所有 WHERE 条件
	for _, cond := range qb.conditions {
		switch cond.Operator {
		case "=":
			query = query.Where(fmt.Sprintf("%s = ?", cond.Field), cond.Value)
		case "!=":
			query = query.Where(fmt.Sprintf("%s != ?", cond.Field), cond.Value)
		case ">":
			query = query.Where(fmt.Sprintf("%s > ?", cond.Field), cond.Value)
		case ">=":
			query = query.Where(fmt.Sprintf("%s >= ?", cond.Field), cond.Value)
		case "<":
			query = query.Where(fmt.Sprintf("%s < ?", cond.Field), cond.Value)
		case "<=":
			query = query.Where(fmt.Sprintf("%s <= ?", cond.Field), cond.Value)
		case "LIKE":
			query = query.Where(fmt.Sprintf("%s LIKE ?", cond.Field), cond.Value)
		case "IN":
			query = query.Where(fmt.Sprintf("%s IN ?", cond.Field), cond.Value)
		case "NOT IN":
			query = query.Where(fmt.Sprintf("%s NOT IN ?", cond.Field), cond.Value)
		}
	}

	// 应用 OR 条件组
	for _, orGroup := range qb.orConditions {
		orQuery := query.Session(&gorm.Session{})
		for i, cond := range orGroup {
			switch cond.Operator {
			case "=":
				if cond.Value == nil {
					if i == 0 {
						orQuery = orQuery.Where(fmt.Sprintf("%s IS NULL", cond.Field))
					} else {
						orQuery = orQuery.Or(fmt.Sprintf("%s IS NULL", cond.Field))
					}
				} else {
					if i == 0 {
						orQuery = orQuery.Where(fmt.Sprintf("%s = ?", cond.Field), cond.Value)
					} else {
						orQuery = orQuery.Or(fmt.Sprintf("%s = ?", cond.Field), cond.Value)
					}
				}
			}
		}
		query = query.Where(orQuery)
	}

	// 应用预加载
	for _, preload := range qb.preloads {
		query = query.Preload(preload)
	}

	for _, preload := range qb.preloadsWith {
		query = query.Preload(preload.field, preload.callback)
	}

	// 应用排序
	for _, order := range qb.orderBy {
		query = query.Order(fmt.Sprintf("%s %s", order.Field, order.Direction))
	}

	return query
}

// Paginate 分页查询
func Paginate(query *gorm.DB, current, pageSize int) *gorm.DB {
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (current - 1) * pageSize
	return query.Offset(offset).Limit(pageSize)
}

// CountAndQuery 执行查询并返回总数和结果
func CountAndQuery(query *gorm.DB, dest interface{}, current, pageSize int) (int64, error) {
	var total int64

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 应用分页
	query = Paginate(query, current, pageSize)

	// 执行查询
	if err := query.Find(dest).Error; err != nil {
		return 0, fmt.Errorf("查询数据失败: %w", err)
	}

	return total, nil
}
