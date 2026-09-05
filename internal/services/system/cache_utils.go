package system

import (
	"strings"
)

// filterSlice 通用切片过滤
func filterSlice[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// contains 字符串包含检查（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// paginate 内存分页工具函数
func paginate[T any](items []T, current, pageSize int) ([]T, int64) {
	total := int64(len(items))
	if total == 0 {
		return []T{}, 0
	}

	start := (current - 1) * pageSize
	if start >= len(items) {
		return []T{}, total
	}

	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	return items[start:end], total
}

// ==================== 缓存服务通用辅助函数 ====================

// CacheServiceBase 与 GetExpiration 已迁至 internal/services/base（Phase 92-01，
// D-01/D-02），本包经 cache_provider.go 的 type alias 原位引用，嵌入点零改动。
//
// 原 InvalidateCacheByPattern/InvalidateCacheByKey 已删除（Phase 92-03 D-04）：
// 失效底层唯一权威为 base.Invalidate/base.InvalidatePattern。
