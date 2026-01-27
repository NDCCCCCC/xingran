package operations

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BatchUpsert 批量插入或更新
// 使用GORM的Clause实现PostgreSQL的ON CONFLICT DO UPDATE功能
type BatchUpsert struct {
	db     *gorm.DB
	config ExcelConfig
}

// NewBatchUpsert 创建批量Upsert实例
func NewBatchUpsert(db *gorm.DB, config ExcelConfig) *BatchUpsert {
	return &BatchUpsert{db: db, config: config}
}

// Upsert 执行批量插入或更新
// 返回：影响行数、错误
func (b *BatchUpsert) Upsert(
	ctx context.Context,
	records []map[string]interface{},
) (int, int, error) {
	if len(records) == 0 {
		return 0, 0, nil
	}

	// 构建冲突处理列
	conflictColumns := b.buildConflictColumns()

	// 构建需要更新的列
	updateColumns := b.buildUpdateColumns()

	// 执行批量Upsert
	result := b.db.WithContext(ctx).
		Table(b.config.TableName).
		Clauses(clause.OnConflict{
			Columns:   conflictColumns,
			DoUpdates: clause.AssignmentColumns(updateColumns),
		}).
		Create(&records)

	if result.Error != nil {
		return 0, 0, fmt.Errorf("批量Upsert失败: %w", result.Error)
	}

	return int(result.RowsAffected), 0, nil
}

// buildConflictColumns 构建冲突检测列
// 优先级：UpsertKey > UniqueKeys > ID
func (b *BatchUpsert) buildConflictColumns() []clause.Column {
	var columns []clause.Column

	// 优先使用配置的UpsertKey字段
	for _, col := range b.config.Columns {
		if col.UpsertKey {
			columns = append(columns, clause.Column{Name: col.Field})
		}
	}

	// 其次使用配置的UniqueKeys
	if len(columns) == 0 && len(b.config.UniqueKeys) > 0 {
		for _, key := range b.config.UniqueKeys {
			columns = append(columns, clause.Column{Name: key})
		}
	}

	// 最后使用ID（默认回退）
	if len(columns) == 0 {
		columns = []clause.Column{{Name: "id"}}
	}

	return columns
}

// buildUpdateColumns 构建需要更新的列
// 排除：ID字段、创建时间、冲突检测列
func (b *BatchUpsert) buildUpdateColumns() []string {
	updateColumns := make([]string, 0)
	conflictFields := b.collectConflictFields()

	for _, col := range b.config.Columns {
		// 跳过ID、创建时间和冲突检测列
		if b.shouldSkipFromUpdate(col.Field, conflictFields) {
			continue
		}
		updateColumns = append(updateColumns, col.Field)
	}

	return updateColumns
}

// collectConflictFields 收集所有冲突检测字段
func (b *BatchUpsert) collectConflictFields() map[string]bool {
	conflictFields := make(map[string]bool)

	// 收集UpsertKey字段
	for _, col := range b.config.Columns {
		if col.UpsertKey {
			conflictFields[col.Field] = true
		}
	}

	// 收集UniqueKeys字段
	for _, key := range b.config.UniqueKeys {
		conflictFields[key] = true
	}

	return conflictFields
}

// shouldSkipFromUpdate 判断字段是否应该从更新中排除
func (b *BatchUpsert) shouldSkipFromUpdate(field string, conflictFields map[string]bool) bool {
	// 排除ID和创建时间字段
	if field == "id" || field == "createdAt" || field == "created_at" {
		return true
	}

	// 排除冲突检测字段
	return conflictFields[field]
}
