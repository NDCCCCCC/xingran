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
// 返回：插入行数、更新行数、错误
func (b *BatchUpsert) Upsert(
	ctx context.Context,
	records []map[string]interface{},
) (int, int, error) {
	if len(records) == 0 {
		return 0, 0, nil
	}

	// 如果启用了部分更新，使用部分更新逻辑
	if b.config.PartialUpdate {
		return b.partialUpsert(ctx, records)
	}

	// 否则使用标准的批量Upsert
	return b.standardUpsert(ctx, records)
}

// standardUpsert 标准批量Upsert（更新所有字段）
func (b *BatchUpsert) standardUpsert(
	ctx context.Context,
	records []map[string]interface{},
) (int, int, error) {
	// 构建冲突处理列
	conflictColumns := b.buildConflictColumns()
	// 获取需要更新的列
	updateColumns := b.buildUpdateColumns()

	// 执行批量Upsert（显式指定更新列）
	// 注意：使用 Table() + map 切片时，UpdateAll: true 不生效，必须显式指定列
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

// partialUpsert 部分更新的Upsert（只更新有值的字段）
func (b *BatchUpsert) partialUpsert(
	ctx context.Context,
	records []map[string]interface{},
) (int, int, error) {
	// 获取冲突检测字段
	conflictColumns := b.buildConflictColumns()
	if len(conflictColumns) == 0 {
		return 0, 0, fmt.Errorf("未配置冲突检测列")
	}

	// 提取所有冲突键值
	var conflictKeys []string
	keyField := conflictColumns[0].Name // 通常只有一个冲突键

	// 验证字段名安全性（防止 SQL 注入）
	if !isValidFieldName(keyField) {
		return 0, 0, fmt.Errorf("无效的字段名: %s", keyField)
	}

	for _, record := range records {
		if val, ok := record[keyField]; ok && val != nil {
			if strVal, ok := val.(string); ok && strVal != "" {
				conflictKeys = append(conflictKeys, strVal)
			}
		}
	}

	if len(conflictKeys) == 0 {
		return 0, 0, fmt.Errorf("没有有效的冲突键值")
	}

	// 查询已存在的记录
	var existingRecords []map[string]interface{}
	err := b.db.WithContext(ctx).
		Table(b.config.TableName).
		Where(fmt.Sprintf("%s IN ?", keyField), conflictKeys).
		Find(&existingRecords).Error

	if err != nil {
		return 0, 0, fmt.Errorf("查询已存在记录失败: %w", err)
	}

	// 构建已存在记录的映射
	existingMap := make(map[string]map[string]interface{})
	for _, record := range existingRecords {
		if val, ok := record[keyField]; ok {
			if strVal, ok := val.(string); ok {
				existingMap[strVal] = record
			}
		}
	}

	// 分离新记录和更新记录
	var toInsert []map[string]interface{}
	var toUpdate []map[string]interface{}

	for _, record := range records {
		keyVal, exists := record[keyField]
		if !exists || keyVal == nil {
			continue
		}

		keyStr, ok := keyVal.(string)
		if !ok || keyStr == "" {
			continue
		}

		if _, found := existingMap[keyStr]; found {
			// 记录已存在，构建更新数据（只包含有值的字段）
			updateData := make(map[string]interface{})
			updateData[keyField] = keyStr // 用于WHERE条件

			// 获取所有可更新的字段
			updateColumns := b.buildUpdateColumns()

			for _, col := range updateColumns {
				// 跳过冲突键字段
				if col == keyField {
					continue
				}

				// 检查新记录中是否有值
				if newVal, hasValue := record[col]; hasValue && newVal != nil && newVal != "" {
					// 对于字符串，检查是否非空
					if strVal, ok := newVal.(string); ok && strVal != "" {
						updateData[col] = newVal
					} else if !ok {
						// 非字符串类型（如数字），只要有值就更新
						updateData[col] = newVal
					}
				}
			}

			// 添加公共字段
			if record["updated_at"] != nil {
				updateData["updated_at"] = record["updated_at"]
			}

			toUpdate = append(toUpdate, updateData)
		} else {
			// 新记录，直接插入
			toInsert = append(toInsert, record)
		}
	}

	// 执行插入
	insertedCount := 0
	if len(toInsert) > 0 {
		result := b.db.WithContext(ctx).
			Table(b.config.TableName).
			Create(&toInsert)
		if result.Error != nil {
			return 0, 0, fmt.Errorf("批量插入失败: %w", result.Error)
		}
		insertedCount = int(result.RowsAffected)
	}

	// 执行更新（逐条更新，因为每条记录的更新字段可能不同）
	updatedCount := 0
	for _, updateData := range toUpdate {
		keyVal := updateData[keyField]
		delete(updateData, keyField) // 移除键字段，不更新它

		if len(updateData) == 0 {
			continue // 没有需要更新的字段
		}

		result := b.db.WithContext(ctx).
			Table(b.config.TableName).
			Where(fmt.Sprintf("%s = ?", keyField), keyVal).
			Updates(updateData)

		if result.Error != nil {
			return 0, 0, fmt.Errorf("更新记录失败 [%s=%v]: %w", keyField, keyVal, result.Error)
		}

		if result.RowsAffected > 0 {
			updatedCount++
		}
	}

	return insertedCount, updatedCount, nil
}

// buildConflictColumns 构建冲突检测列
// 优先级：UpsertKey > UniqueKeys > ID
func (b *BatchUpsert) buildConflictColumns() []clause.Column {
	var columns []clause.Column

	// 优先使用配置的UpsertKey字段
	for _, col := range b.config.Columns {
		if col.UpsertKey {
			dbField := b.getDBFieldName(col)
			columns = append(columns, clause.Column{Name: dbField})
		}
	}

	// 其次使用配置的UniqueKeys（需要转换为数据库字段名）
	if len(columns) == 0 && len(b.config.UniqueKeys) > 0 {
		for _, key := range b.config.UniqueKeys {
			// 在Columns配置中查找对应的字段定义
			for _, col := range b.config.Columns {
				if col.Field == key {
					dbField := b.getDBFieldName(col)
					if dbField != "" {
						columns = append(columns, clause.Column{Name: dbField})
					}
					break
				}
			}
		}
	}

	// 最后使用ID（默认回退）
	if len(columns) == 0 {
		columns = []clause.Column{{Name: "id"}}
	}

	return columns
}

// buildUpdateColumns 构建需要更新的列
// 排除：ID字段、创建时间、冲突检测列、引用字段（仅用于解析的字段）、无DBField的临时字段
func (b *BatchUpsert) buildUpdateColumns() []string {
	updateColumns := make([]string, 0)
	conflictFields := b.collectConflictFields()

	for _, col := range b.config.Columns {
		// 获取数据库字段名（如果DBField为空，使用Field本身）
		dbField := b.getDBFieldName(col)

		// 跳过没有有效字段名的（临时处理字段，不应写入数据库）
		if dbField == "" {
			continue
		}

		// 跳过ID、创建时间、冲突检测列和引用字段
		if b.shouldSkipFromUpdate(dbField, col, conflictFields) {
			continue
		}
		updateColumns = append(updateColumns, dbField)
	}

	return updateColumns
}

// getDBFieldName 获取数据库字段名
func (b *BatchUpsert) getDBFieldName(col ExcelColumn) string {
	// 如果配置了DBField，优先使用配置
	if col.DBField != "" {
		return col.DBField
	}
	// 否则使用字段名本身
	return col.Field
}

// collectConflictFields 收集所有冲突检测字段
// 优化版本：预先建立字段映射避免嵌套循环
func (b *BatchUpsert) collectConflictFields() map[string]bool {
	conflictFields := make(map[string]bool)

	// 建立字段名到列配置的映射（避免嵌套循环）
	fieldToCol := make(map[string]ExcelColumn)
	for _, col := range b.config.Columns {
		dbField := b.getDBFieldName(col)
		fieldToCol[dbField] = col
		if col.DBField != "" {
			fieldToCol[col.DBField] = col
		}
		fieldToCol[col.Field] = col
	}

	// 收集UpsertKey字段（使用数据库字段名）
	for _, col := range b.config.Columns {
		if col.UpsertKey {
			dbField := b.getDBFieldName(col)
			conflictFields[dbField] = true
		}
	}

	// 收集UniqueKeys字段（从映射中快速查找）
	for _, key := range b.config.UniqueKeys {
		if col, exists := fieldToCol[key]; exists {
			dbField := b.getDBFieldName(col)
			conflictFields[dbField] = true
		}
	}

	return conflictFields
}

// shouldSkipFromUpdate 判断字段是否应该从更新中排除
func (b *BatchUpsert) shouldSkipFromUpdate(field string, col ExcelColumn, conflictFields map[string]bool) bool {
	// 排除ID和创建时间字段
	if field == "id" || field == "createdAt" || field == "created_at" || field == "updatedAt" || field == "updated_at" {
		return true
	}

	// 排除引用字段（仅用于解析，不在数据库中存储）
	// 但如果配置了 DBField 且 DBField != Field，说明这是要更新的目标字段，不应跳过
	if col.Reference != "" {
		// 如果没有配置 DBField，或者 DBField == Field（相同），说明是原始引用字段，应该跳过
		if col.DBField == "" || col.DBField == col.Field {
			return true
		}
	}

	// 排除冲突检测字段
	return conflictFields[field]
}

// isValidFieldName 验证字段名是否安全（防止 SQL 注入）
// 只允许字母、数字和下划线，且必须以字母开头
func isValidFieldName(field string) bool {
	if field == "" {
		return false
	}

	for i, r := range field {
		if i == 0 && !isAlpha(r) {
			return false // 第一个字符必须是字母
		}
		if !isAlphaNum(r) && r != '_' {
			return false // 只允许字母、数字和下划线
		}
	}
	return true
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAlphaNum(r rune) bool {
	return isAlpha(r) || (r >= '0' && r <= '9')
}
