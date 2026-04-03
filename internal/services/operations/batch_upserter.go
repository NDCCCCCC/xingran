package operations

import (
	"context"
	"fmt"
	"strings"

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
func (b *BatchUpsert) Upsert(ctx context.Context, records []map[string]interface{}) (int, int, error) {
	if len(records) == 0 {
		return 0, 0, nil
	}

	if b.config.PartialUpdate {
		return b.partialUpsert(ctx, records)
	}
	return b.standardUpsert(ctx, records)
}

// standardUpsert 标准批量Upsert（更新所有字段）
func (b *BatchUpsert) standardUpsert(ctx context.Context, records []map[string]interface{}) (int, int, error) {
	conflictColumns := b.buildConflictColumns()
	updateColumns := b.buildUpdateColumns()

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
// 支持单列和多列冲突检测
func (b *BatchUpsert) partialUpsert(ctx context.Context, records []map[string]interface{}) (int, int, error) {
	conflictColumns := b.buildConflictColumns()
	if len(conflictColumns) == 0 {
		return 0, 0, fmt.Errorf("未配置冲突检测列")
	}

	// 查询已存在的记录
	existingMap, err := b.queryExistingRecords(ctx, records, conflictColumns)
	if err != nil {
		return 0, 0, err
	}

	// 分离新记录和更新记录
	toInsert, toUpdate := b.separateRecords(records, existingMap, conflictColumns)

	// 执行插入
	insertedCount, err := b.executeInsert(ctx, toInsert)
	if err != nil {
		return 0, 0, err
	}

	// 执行更新
	updatedCount, err := b.executeUpdate(ctx, toUpdate, conflictColumns)
	if err != nil {
		return 0, 0, err
	}

	return insertedCount, updatedCount, nil
}

// queryExistingRecords 查询已存在的记录并构建映射
func (b *BatchUpsert) queryExistingRecords(ctx context.Context, records []map[string]interface{}, conflictColumns []clause.Column) (map[string]map[string]interface{}, error) {
	// 收集所有冲突键值
	columnValues := make(map[string][]interface{})
	for _, record := range records {
		for _, col := range conflictColumns {
			if val := getValidValue(record, col.Name); val != nil {
				columnValues[col.Name] = append(columnValues[col.Name], val)
			}
		}
	}

	if len(columnValues) == 0 {
		return nil, fmt.Errorf("没有有效的冲突键值")
	}

	// 构建查询
	query := b.db.WithContext(ctx).Table(b.config.TableName)
	for colName, values := range columnValues {
		query = query.Where(fmt.Sprintf("%s IN ?", colName), uniqueValues(values))
	}

	var existingRecords []map[string]interface{}
	if err := query.Find(&existingRecords).Error; err != nil {
		return nil, fmt.Errorf("查询已存在记录失败: %w", err)
	}

	// 构建映射
	existingMap := make(map[string]map[string]interface{})
	for _, record := range existingRecords {
		if key := buildCombinedKey(record, conflictColumns); key != "" {
			existingMap[key] = record
		}
	}

	return existingMap, nil
}

// separateRecords 分离需要插入和更新的记录
func (b *BatchUpsert) separateRecords(records []map[string]interface{}, existingMap map[string]map[string]interface{}, conflictColumns []clause.Column) ([]map[string]interface{}, []map[string]interface{}) {
	var toInsert []map[string]interface{}
	var toUpdate []map[string]interface{}

	conflictNames := make([]string, len(conflictColumns))
	for i, col := range conflictColumns {
		conflictNames[i] = col.Name
	}

	for _, record := range records {
		key := buildCombinedKey(record, conflictColumns)
		if key == "" {
			continue
		}

		if existing, found := existingMap[key]; found {
			// 构建更新数据
			updateData := b.buildUpdateData(record, existing, conflictNames)
			if len(updateData) > 0 {
				toUpdate = append(toUpdate, updateData)
			}
		} else {
			toInsert = append(toInsert, record)
		}
	}

	return toInsert, toUpdate
}

// buildUpdateData 构建更新数据
func (b *BatchUpsert) buildUpdateData(record, existing map[string]interface{}, conflictNames []string) map[string]interface{} {
	updateData := make(map[string]interface{})

	// 保存 ID 用于 WHERE 条件
	if id, ok := existing["id"]; ok {
		updateData["__id__"] = id
	}

	// 收集需要更新的字段
	conflictSet := make(map[string]bool)
	for _, name := range conflictNames {
		conflictSet[name] = true
	}

	updateColumns := b.buildUpdateColumns()
	for _, col := range updateColumns {
		if conflictSet[col] {
			continue
		}
		if val := getValidValue(record, col); val != nil {
			updateData[col] = val
		}
	}

	// 添加更新时间
	if updatedAt := getValidValue(record, "updated_at"); updatedAt != nil {
		updateData["updated_at"] = updatedAt
	}

	return updateData
}

// executeInsert 执行批量插入
func (b *BatchUpsert) executeInsert(ctx context.Context, records []map[string]interface{}) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	result := b.db.WithContext(ctx).Table(b.config.TableName).Create(&records)
	if result.Error != nil {
		return 0, fmt.Errorf("批量插入失败: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// executeUpdate 执行批量更新
func (b *BatchUpsert) executeUpdate(ctx context.Context, records []map[string]interface{}, conflictColumns []clause.Column) (int, error) {
	updatedCount := 0

	for _, updateData := range records {
		id, hasID := updateData["__id__"]
		delete(updateData, "__id__")

		// 移除冲突键字段
		for _, col := range conflictColumns {
			delete(updateData, col.Name)
		}

		if len(updateData) == 0 || !hasID {
			continue
		}

		result := b.db.WithContext(ctx).
			Table(b.config.TableName).
			Where("id = ?", id).
			Updates(updateData)

		if result.Error != nil {
			return 0, fmt.Errorf("更新记录失败 [id=%v]: %w", id, result.Error)
		}

		if result.RowsAffected > 0 {
			updatedCount++
		}
	}

	return updatedCount, nil
}

// buildCombinedKey 构建组合键字符串
func buildCombinedKey(record map[string]interface{}, columns []clause.Column) string {
	parts := make([]string, 0, len(columns))
	for _, col := range columns {
		if val := record[col.Name]; val != nil {
			parts = append(parts, fmt.Sprintf("%v", val))
		}
	}
	if len(parts) != len(columns) {
		return ""
	}
	return strings.Join(parts, "|")
}

// getValidValue 获取有效的非空值
func getValidValue(record map[string]interface{}, key string) interface{} {
	val, ok := record[key]
	if !ok || val == nil {
		return nil
	}
	if strVal, ok := val.(string); ok && strVal == "" {
		return nil
	}
	return val
}

// uniqueValues 去重
func uniqueValues(values []interface{}) []interface{} {
	seen := make(map[interface{}]bool)
	result := make([]interface{}, 0, len(values))
	for _, v := range values {
		key := fmt.Sprintf("%v", v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	return result
}

// buildConflictColumns 构建冲突检测列
// 优先级：UpsertKey > UniqueKeys > ID
func (b *BatchUpsert) buildConflictColumns() []clause.Column {
	var columns []clause.Column

	// 优先使用配置的UpsertKey字段
	for _, col := range b.config.Columns {
		if col.UpsertKey {
			columns = append(columns, clause.Column{Name: b.getDBFieldName(col)})
		}
	}

	// 其次使用配置的UniqueKeys
	if len(columns) == 0 && len(b.config.UniqueKeys) > 0 {
		fieldToCol := b.buildFieldToColMap()
		for _, key := range b.config.UniqueKeys {
			if col, exists := fieldToCol[key]; exists {
				columns = append(columns, clause.Column{Name: b.getDBFieldName(col)})
			}
		}
	}

	// 默认使用ID
	if len(columns) == 0 {
		columns = []clause.Column{{Name: "id"}}
	}

	return columns
}

// buildUpdateColumns 构建需要更新的列
func (b *BatchUpsert) buildUpdateColumns() []string {
	conflictFields := b.collectConflictFields()
	var updateColumns []string

	for _, col := range b.config.Columns {
		dbField := b.getDBFieldName(col)
		if dbField == "" {
			continue
		}
		if b.shouldSkipFromUpdate(dbField, col, conflictFields) {
			continue
		}
		updateColumns = append(updateColumns, dbField)
	}

	return updateColumns
}

// getDBFieldName 获取数据库字段名
func (b *BatchUpsert) getDBFieldName(col ExcelColumn) string {
	if col.DBField != "" {
		return col.DBField
	}
	return col.Field
}

// buildFieldToColMap 构建字段名到列配置的映射
func (b *BatchUpsert) buildFieldToColMap() map[string]ExcelColumn {
	m := make(map[string]ExcelColumn)
	for _, col := range b.config.Columns {
		m[col.Field] = col
		if col.DBField != "" {
			m[col.DBField] = col
		}
		m[b.getDBFieldName(col)] = col
	}
	return m
}

// collectConflictFields 收集所有冲突检测字段
func (b *BatchUpsert) collectConflictFields() map[string]bool {
	conflictFields := make(map[string]bool)
	fieldToCol := b.buildFieldToColMap()

	for _, col := range b.config.Columns {
		if col.UpsertKey {
			conflictFields[b.getDBFieldName(col)] = true
		}
	}

	for _, key := range b.config.UniqueKeys {
		if col, exists := fieldToCol[key]; exists {
			conflictFields[b.getDBFieldName(col)] = true
		}
	}

	return conflictFields
}

// shouldSkipFromUpdate 判断字段是否应该从更新中排除
func (b *BatchUpsert) shouldSkipFromUpdate(field string, col ExcelColumn, conflictFields map[string]bool) bool {
	switch field {
	case "id", "createdAt", "created_at", "updatedAt", "updated_at":
		return true
	}

	// 排除未配置DBField的引用字段
	if col.Reference != "" && (col.DBField == "" || col.DBField == col.Field) {
		return true
	}

	return conflictFields[field]
}
