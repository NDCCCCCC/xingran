package gormutil

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// JoinType JOIN类型
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
)

// JoinConfig JOIN配置
type JoinConfig struct {
	Type    JoinType
	Table   string
	On      string
	Alias   string
	Args    []interface{}
	Selects []string
}

// JoinBuilder JOIN构建器
type JoinBuilder struct {
	db      *gorm.DB
	configs []JoinConfig
	alias   map[string]bool // 跟踪已使用的别名
}

// NewJoinBuilder 创建JOIN构建器
func NewJoinBuilder(db *gorm.DB) *JoinBuilder {
	return &JoinBuilder{
		db:    db,
		alias: make(map[string]bool),
	}
}

// NewJoinBuilderWithModel 从模型创建JOIN构建器
func NewJoinBuilderWithModel(db *gorm.DB, model interface{}) *JoinBuilder {
	return &JoinBuilder{
		db:    db.Model(model),
		alias: make(map[string]bool),
	}
}

// InnerJoin 添加内连接
func (b *JoinBuilder) InnerJoin(table, on string, args ...interface{}) *JoinBuilder {
	return b.addJoin(InnerJoin, table, on, "", args...)
}

// LeftJoin 添加左连接
func (b *JoinBuilder) LeftJoin(table, on string, args ...interface{}) *JoinBuilder {
	return b.addJoin(LeftJoin, table, on, "", args...)
}

// RightJoin 添加右连接
func (b *JoinBuilder) RightJoin(table, on string, args ...interface{}) *JoinBuilder {
	return b.addJoin(RightJoin, table, on, "", args...)
}

// LeftJoinWithAlias 添加带别名的左连接
func (b *JoinBuilder) LeftJoinWithAlias(table, alias, on string, args ...interface{}) *JoinBuilder {
	return b.addJoin(LeftJoin, table, on, alias, args...)
}

// addJoin 添加JOIN配置
func (b *JoinBuilder) addJoin(joinType JoinType, table, on, alias string, args ...interface{}) *JoinBuilder {
	config := JoinConfig{
		Type:  joinType,
		Table: table,
		On:    on,
		Alias: alias,
		Args:  args,
	}
	b.configs = append(b.configs, config)
	if alias != "" {
		b.alias[alias] = true
	}
	return b
}

// Select 添加SELECT字段
func (b *JoinBuilder) Select(selects ...string) *JoinBuilder {
	if len(b.configs) == 0 {
		// GORM的Select接受可变参数，直接展开
		args := make([]interface{}, len(selects))
		for i, s := range selects {
			args[i] = s
		}
		b.db = b.db.Select(args[0], args[1:]...)
		return b
	}
	// 添加到最后一个JOIN配置
	lastIdx := len(b.configs) - 1
	b.configs[lastIdx].Selects = append(b.configs[lastIdx].Selects, selects...)
	return b
}

// Where 添加WHERE条件
func (b *JoinBuilder) Where(query interface{}, args ...interface{}) *JoinBuilder {
	b.db = b.db.Where(query, args...)
	return b
}

// Or 添加OR条件
func (b *JoinBuilder) Or(query interface{}, args ...interface{}) *JoinBuilder {
	b.db = b.db.Or(query, args...)
	return b
}

// Order 添加排序
func (b *JoinBuilder) Order(value interface{}) *JoinBuilder {
	b.db = b.db.Order(value)
	return b
}

// Limit 添加限制
func (b *JoinBuilder) Limit(limit int) *JoinBuilder {
	b.db = b.db.Limit(limit)
	return b
}

// Offset 添加偏移
func (b *JoinBuilder) Offset(offset int) *JoinBuilder {
	b.db = b.db.Offset(offset)
	return b
}

// Count 统计数量
func (b *JoinBuilder) Count(count *int64) *JoinBuilder {
	b.db = b.db.Count(count)
	return b
}

// Build 构建并返回DB
func (b *JoinBuilder) Build() *gorm.DB {
	// 应用所有JOIN配置
	for _, config := range b.configs {
		joinClause := string(config.Type)
		tableExpr := config.Table
		if config.Alias != "" {
			tableExpr = fmt.Sprintf("%s AS %s", config.Table, config.Alias)
		}
		joinClause += " " + tableExpr + " ON " + config.On

		if len(config.Args) > 0 {
			b.db = b.db.Joins(joinClause, config.Args...)
		} else {
			b.db = b.db.Joins(joinClause)
		}
	}
	return b.db
}

// Scan 扫描结果到目标
func (b *JoinBuilder) Scan(dest interface{}) *gorm.DB {
	return b.Build().Scan(dest)
}

// Find 查询结果到目标
func (b *JoinBuilder) Find(dest interface{}) *gorm.DB {
	return b.Build().Find(dest)
}

// First 查询单条结果
func (b *JoinBuilder) First(dest interface{}) *gorm.DB {
	return b.Build().First(dest)
}

// Pluck 提取单个列
func (b *JoinBuilder) Pluck(column string, dest interface{}) *gorm.DB {
	return b.Build().Pluck(column, dest)
}

// GetDB 获取底层DB（不应用JOIN）
func (b *JoinBuilder) GetDB() *gorm.DB {
	return b.db
}

// Reset 重置构建器
func (b *JoinBuilder) Reset() *JoinBuilder {
	b.configs = make([]JoinConfig, 0)
	b.alias = make(map[string]bool)
	return b
}

// GetConfigs 获取所有JOIN配置
func (b *JoinBuilder) GetConfigs() []JoinConfig {
	return b.configs
}

// BuildOnClause 构建ON子句
func BuildOnClause(leftTable, leftField, rightTable, rightField string) string {
	return fmt.Sprintf("%s.%s = %s.%s", leftTable, leftField, rightTable, rightField)
}

// BuildJoinClause 构建完整的JOIN子句
func BuildJoinClause(joinType JoinType, table, on string) string {
	return fmt.Sprintf("%s %s ON %s", joinType, table, on)
}

// ParseSelectFields 解析SELECT字段，自动添加表前缀
func ParseSelectFields(table string, fields []string) []string {
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if !strings.Contains(field, ".") {
			result = append(result, fmt.Sprintf("%s.%s", table, field))
		} else {
			result = append(result, field)
		}
	}
	return result
}
