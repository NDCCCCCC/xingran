package operations

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ReferenceResolver 引用解析器接口
// 将Excel中用户填写的名称/编码转换为数据库中的ID
type ReferenceResolver interface {
	ResolveBatch(ctx context.Context, refs []ReferenceRequest) (map[string]string, error)
	ResolveSingle(ctx context.Context, ref ReferenceRequest) (string, error)
}

// ReferenceRequest 引用解析请求
type ReferenceRequest struct {
	Reference string // 引用配置，格式："table.field"，例如："sys_dept.dept_code"
	Value     string // 待解析的值（用户在Excel中填写的名称/编码）
}

// NewReferenceResolver 创建引用解析器
func NewReferenceResolver(db *gorm.DB) ReferenceResolver {
	return &referenceResolverImpl{db: db}
}

// referenceResolverImpl 引用解析器实现
type referenceResolverImpl struct {
	db *gorm.DB
}

// ResolveBatch 批量解析引用值
// 通过按引用类型分组，每种类型只执行一次查询，大幅提升性能
func (r *referenceResolverImpl) ResolveBatch(
	ctx context.Context,
	refs []ReferenceRequest,
) (map[string]string, error) {
	if len(refs) == 0 {
		return make(map[string]string), nil
	}

	// 按引用类型分组
	grouped := r.groupByReference(refs)

	result := make(map[string]string)
	var firstErr error

	// 对每种引用类型执行一次查询
	for refType, requests := range grouped {
		table, field := r.parseReference(refType)

		// 验证引用配置格式
		if table == "" || field == "" {
			return nil, fmt.Errorf("无效的引用配置: %s", refType)
		}

		// 提取所有待解析的值（去重）
		values := r.extractValues(requests)

		// 批量查询数据库
		idMap, err := r.batchQueryIDs(ctx, table, field, values)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("批量查询失败 [%s.%s]: %w", table, field, err)
			}
			// 继续处理其他引用类型
			continue
		}

		// 组装结果
		for _, req := range requests {
			key := r.makeKey(req.Reference, req.Value)
			if id, exists := idMap[req.Value]; exists {
				result[key] = id
			}
			// 如果找不到对应ID，不设置result中的key，调用者可以判断引用不存在
		}
	}

	return result, firstErr
}

// ResolveSingle 单个引用解析（用于回退或单个查询）
func (r *referenceResolverImpl) ResolveSingle(
	ctx context.Context,
	ref ReferenceRequest,
) (string, error) {
	table, field := r.parseReference(ref.Reference)

	if table == "" || field == "" {
		return "", fmt.Errorf("无效的引用配置: %s", ref.Reference)
	}

	var result struct {
		ID string `gorm:"column:id"`
	}

	err := r.db.WithContext(ctx).
		Table(table).
		Select("id").
		Where(field+" = ?", ref.Value).
		Where("deleted_at IS NULL").
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("引用记录不存在 [%s.%s = %s]", table, field, ref.Value)
		}
		return "", fmt.Errorf("引用解析失败 [%s.%s = %s]: %w", table, field, ref.Value, err)
	}

	return result.ID, nil
}

// groupByReference 按引用类型分组
func (r *referenceResolverImpl) groupByReference(
	refs []ReferenceRequest,
) map[string][]ReferenceRequest {
	grouped := make(map[string][]ReferenceRequest)
	for _, ref := range refs {
		grouped[ref.Reference] = append(grouped[ref.Reference], ref)
	}
	return grouped
}

// parseReference 解析引用配置 "table.field" -> (table, field)
func (r *referenceResolverImpl) parseReference(ref string) (table, field string) {
	parts := strings.Split(ref, ".")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// extractValues 提取所有待解析的值（去重）
func (r *referenceResolverImpl) extractValues(requests []ReferenceRequest) []string {
	values := make([]string, 0, len(requests))
	seen := make(map[string]bool)

	for _, req := range requests {
		if !seen[req.Value] {
			values = append(values, req.Value)
			seen[req.Value] = true
		}
	}

	return values
}

// batchQueryIDs 批量查询ID映射 value -> id
func (r *referenceResolverImpl) batchQueryIDs(
	ctx context.Context,
	table, field string,
	values []string,
) (map[string]string, error) {
	if len(values) == 0 {
		return make(map[string]string), nil
	}

	type Result struct {
		ID    string `gorm:"column:id"`
		Value string `gorm:"column:value"`
	}

	var results []Result

	// 构建查询：SELECT id, {field} FROM {table} WHERE {field} IN (...) AND deleted_at IS NULL
	err := r.db.WithContext(ctx).
		Table(table).
		Select("id, "+field+" as value").
		Where(field+" IN ?", values).
		Where("deleted_at IS NULL").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为 value -> id 映射
	idMap := make(map[string]string)
	for _, r := range results {
		idMap[r.Value] = r.ID
	}

	return idMap, nil
}

// makeKey 生成缓存键，格式："reference:value"
func (r *referenceResolverImpl) makeKey(ref, value string) string {
	return ref + ":" + value
}
