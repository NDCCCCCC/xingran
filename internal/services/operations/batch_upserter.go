package operations

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// maxBatchSize 单次批量插入的最大行数
// PostgreSQL 限制：65535 参数 / 43 字段 ≈ 1524 行
// 设置为 1000 以留出安全余量
const maxBatchSize = 1000

// NamingStrategy 字段名命名策略接口
// 允许自定义字段名转换逻辑
type NamingStrategy interface {
	// FieldToDB 将字段名转换为数据库列名
	FieldToDB(fieldName string) string
}

// gormNamingStrategy GORM 标准命名策略实现
type gormNamingStrategy struct {
	namer *schema.NamingStrategy
}

// newGORMNamingStrategy 创建 GORM 命名策略实例
func newGORMNamingStrategy() *gormNamingStrategy {
	return &gormNamingStrategy{
		namer: &schema.NamingStrategy{},
	}
}

// FieldToDB 使用 GORM 的 NamingStrategy 将字段名转换为数据库列名
// 例如：deviceCode -> device_code, roomId -> room_id
func (g *gormNamingStrategy) FieldToDB(fieldName string) string {
	return g.namer.ColumnName("", fieldName)
}

// BatchUpsert 批量插入或更新
// 使用GORM的Clause实现PostgreSQL的ON CONFLICT DO UPDATE功能
type BatchUpsert struct {
	db             *gorm.DB
	config         ExcelConfig
	naming         NamingStrategy
	fieldNameCache map[string]string // 字段名缓存，避免重复转换
	fieldNameMutex sync.RWMutex
}

// NewBatchUpsert 创建批量Upsert实例
func NewBatchUpsert(db *gorm.DB, config ExcelConfig) *BatchUpsert {
	return &BatchUpsert{
		db:             db,
		config:         config,
		naming:         newGORMNamingStrategy(),
		fieldNameCache: make(map[string]string),
	}
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

	// 强制包含 deleted_at 以恢复软删除记录
	updateColumns = append(updateColumns, "deleted_at")

	// 转换记录中的字段名为数据库列名
	convertedRecords := b.convertRecordFields(records)

	// 按 conflict columns 去重，保留最后一条（PostgreSQL 不允许单次 batch 中同一冲突键出现多次）
	conflictNames := make([]string, len(conflictColumns))
	for i, col := range conflictColumns {
		conflictNames[i] = col.Name
	}
	seen := make(map[string]int, len(convertedRecords))
	for i, record := range convertedRecords {
		key := buildCombinedKey(record, conflictColumns)
		if key == "" {
			continue
		}
		if prev, exists := seen[key]; exists {
			// 将之前的记录标记为 nil（后面跳过）
			convertedRecords[prev] = nil
		}
		seen[key] = i
	}
	// 过滤掉 nil 记录
	deduped := make([]map[string]interface{}, 0, len(convertedRecords))
	for _, record := range convertedRecords {
		if record != nil {
			deduped = append(deduped, record)
		}
	}
	convertedRecords = deduped

	// 设置 deleted_at 为 nil 以恢复软删除记录
	for i := range convertedRecords {
		convertedRecords[i]["deleted_at"] = nil
	}

	// 预查已存在记录（必须在下方 INSERT 之前 — 插入后所有键必然存在），
	// 用于把 affected 拆分为真实的 inserted/updated 计数。
	// 此前返回 RowsAffected 会把"更新行"也计入 inserted(违反 Upsert 文档契约
	// "返回：插入行数、更新行数"),调用方(Excel 导入结果展示/操作日志)拿到
	// 错误的计数。查询含软删除记录(与 Unscoped 恢复语义一致)。
	//
	// 不复用 queryExistingRecords:它对"无有效冲突键值"返回 error,而
	// standardUpsert 允许 conflict 键为空的记录直接插入(纯插入语义)。
	updatedCount := 0
	if len(conflictColumns) > 0 {
		columnValues := make(map[string][]interface{})
		for _, record := range convertedRecords {
			for _, col := range conflictColumns {
				if val := getValidValue(record, col.Name); val != nil {
					columnValues[col.Name] = append(columnValues[col.Name], val)
				}
			}
		}
		if len(columnValues) > 0 {
			query := b.db.WithContext(ctx).Table(b.config.TableName).Unscoped()
			for colName, values := range columnValues {
				query = query.Where(fmt.Sprintf("%s IN ?", colName), uniqueValues(values))
			}
			var existingRecords []map[string]interface{}
			if err := query.Find(&existingRecords).Error; err != nil {
				return 0, 0, fmt.Errorf("查询已存在记录失败: %w", err)
			}
			existingMap := make(map[string]map[string]interface{}, len(existingRecords))
			for _, record := range existingRecords {
				if key := buildCombinedKey(record, conflictColumns); key != "" {
					existingMap[key] = record
				}
			}
			for _, record := range convertedRecords {
				if key := buildCombinedKey(record, conflictColumns); key != "" {
					if _, found := existingMap[key]; found {
						updatedCount++
					}
				}
			}
		}
	}

	// 分批执行，避免超过 PostgreSQL 参数限制（65535）
	for i := 0; i < len(convertedRecords); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(convertedRecords) {
			end = len(convertedRecords)
		}
		batch := convertedRecords[i:end]

		result := b.db.WithContext(ctx).
			Table(b.config.TableName).
			Unscoped(). // 包含软删除记录
			Clauses(clause.OnConflict{
				Columns:   conflictColumns,
				DoUpdates: clause.AssignmentColumns(updateColumns),
			}).
			Create(&batch)

		if result.Error != nil {
			return 0, 0, fmt.Errorf("批量Upsert失败 (batch %d-%d): %w", i, end, result.Error)
		}
	}

	// updatedCount 已在执行前预查得出（见上方注释）。
	return len(convertedRecords) - updatedCount, updatedCount, nil
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

	// 构建查询（包含软删除的记录，以便恢复它们）
	query := b.db.WithContext(ctx).Table(b.config.TableName).Unscoped()
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
		val := getValidValue(record, col)
		if val == nil {
			continue
		}
		// 只收集值发生变化的字段：与已存在记录对比，值相同则跳过。
		// 避免对大量数据一致的行做无意义 UPDATE（如 AD 同步用户与 Excel 数据一致时），
		// 大幅减少 executeUpdate 的逐条 UPDATE 数量。
		if existingVal, ok := existing[col]; ok && fmt.Sprintf("%v", val) == fmt.Sprintf("%v", existingVal) {
			continue
		}
		updateData[col] = val
	}

	// 添加更新时间
	if updatedAt := getValidValue(record, "updated_at"); updatedAt != nil {
		updateData["updated_at"] = updatedAt
	}

	// 检查是否是软删除记录，如果是则恢复
	// 注意：gorm.DeletedAt 在数据库查询时会被映射为 time.Time
	if deletedAt := existing["deleted_at"]; deletedAt != nil {
		// 统一的软删除检测：检查时间零值
		var isDeleted bool
		switch v := deletedAt.(type) {
		case time.Time:
			isDeleted = !v.IsZero()
		case *time.Time:
			isDeleted = v != nil && !v.IsZero()
		case string:
			isDeleted = v != ""
		default:
			// 其他类型（如 gorm.DeletedAt）通过反射或接口检查
			if tVal, ok := deletedAt.(interface{ IsZero() bool }); ok {
				isDeleted = !tVal.IsZero()
			}
		}

		if isDeleted {
			updateData["deleted_at"] = nil
		}
	}

	return updateData
}

// executeInsert 执行批量插入
func (b *BatchUpsert) executeInsert(ctx context.Context, records []map[string]interface{}) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	// 转换记录中的字段名
	convertedRecords := b.convertRecordFields(records)

	// 分批执行，避免超过 PostgreSQL 参数限制（65535）
	totalInserted := 0
	for i := 0; i < len(convertedRecords); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(convertedRecords) {
			end = len(convertedRecords)
		}
		batch := convertedRecords[i:end]

		result := b.db.WithContext(ctx).Table(b.config.TableName).Create(&batch)
		if result.Error != nil {
			return 0, fmt.Errorf("批量插入失败 (batch %d-%d): %w", i, end, result.Error)
		}
		totalInserted += int(result.RowsAffected)
	}

	return totalInserted, nil
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

		// 清除软删除标记，恢复记录
		updateData["deleted_at"] = nil

		result := b.db.WithContext(ctx).
			Table(b.config.TableName).
			Unscoped(). // 必须使用 Unscoped 才能更新软删除记录
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

// convertRecordFields 将记录中的字段名转换为数据库列名
// 使用缓存避免重复转换，提高性能
func (b *BatchUpsert) convertRecordFields(records []map[string]interface{}) []map[string]interface{} {
	converted := make([]map[string]interface{}, len(records))

	for i, record := range records {
		convertedRecord := make(map[string]interface{}, len(record))
		for field, value := range record {
			dbField := b.resolveFieldName(field)
			if dbField != "" {
				convertedRecord[dbField] = value
			}
		}
		converted[i] = convertedRecord
	}

	return converted
}

// resolveFieldName 解析字段名，返回数据库列名
// 优先级：配置的 DBField > NamingStrategy 转换
// 使用缓存提高性能
func (b *BatchUpsert) resolveFieldName(fieldName string) string {
	// 先检查缓存
	b.fieldNameMutex.RLock()
	if cached, ok := b.fieldNameCache[fieldName]; ok {
		b.fieldNameMutex.RUnlock()
		return cached
	}
	b.fieldNameMutex.RUnlock()

	// 查找配置中是否有 DBField 定义
	var dbField string
	for _, col := range b.config.Columns {
		if col.Field == fieldName {
			dbField = col.DBField
			break
		}
	}

	// 如果没有配置 DBField，使用 NamingStrategy 转换
	var result string
	if dbField != "" {
		result = dbField
	} else {
		result = b.naming.FieldToDB(fieldName)
	}

	// 缓存结果
	b.fieldNameMutex.Lock()
	b.fieldNameCache[fieldName] = result
	b.fieldNameMutex.Unlock()

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
	seen := make(map[string]bool) // 去重：同名 DBField 只出现一次

	for _, col := range b.config.Columns {
		dbField := b.getDBFieldName(col)
		if dbField == "" {
			continue
		}
		if seen[dbField] {
			continue // 跳过重复的 DBField（如 deptCode 和 deptName 都映射到 dept_id）
		}
		seen[dbField] = true
		if b.shouldSkipFromUpdate(dbField, col, conflictFields) {
			continue
		}
		updateColumns = append(updateColumns, dbField)
	}

	return updateColumns
}

// getDBFieldName 获取数据库字段名
// 优先使用配置的 DBField，否则使用 NamingStrategy 自动转换
func (b *BatchUpsert) getDBFieldName(col ExcelColumn) string {
	if col.DBField != "" {
		return col.DBField
	}
	return b.naming.FieldToDB(col.Field)
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
