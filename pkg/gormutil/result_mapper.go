package gormutil

// MapBuilder 查询结果映射构建器
type MapBuilder struct {
	maps []map[string]interface{}
}

// NewMapBuilder 创建映射构建器
func NewMapBuilder() *MapBuilder {
	return &MapBuilder{
		maps: make([]map[string]interface{}, 0),
	}
}

// Add 添加映射
func (b *MapBuilder) Add(m map[string]interface{}) *MapBuilder {
	b.maps = append(b.maps, m)
	return b
}

// Build 构建映射列表
func (b *MapBuilder) Build() []map[string]interface{} {
	return b.maps
}

// ToIDMap 将切片转换为ID到对象的映射
// T必须是具有GetID() string方法或ID字段的结构体
func ToIDMap[T any](items []T, getID func(T) string) map[string]T {
	result := make(map[string]T, len(items))
	for _, item := range items {
		id := getID(item)
		result[id] = item
	}
	return result
}

// GroupBy 按指定字段分组
func GroupBy[T any](items []T, keyFunc func(T) string) map[string][]T {
	result := make(map[string][]T)
	for _, item := range items {
		key := keyFunc(item)
		result[key] = append(result[key], item)
	}
	return result
}

// ExtractIDs 从切片中提取ID列表
func ExtractIDs[T any](items []T, idFunc func(T) string) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		id := idFunc(item)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// IndexBy 将切片索引化，便于快速查找
func IndexBy[T any, K comparable](items []T, keyFunc func(T) K) map[K]T {
	result := make(map[K]T, len(items))
	for _, item := range items {
		key := keyFunc(item)
		result[key] = item
	}
	return result
}

// BatchMap 批量映射，将源数据映射到目标
// 用于批量查询后的数据组装
func BatchMap[S any, T any](
	source []S,
	targetIDs []string,
	getSourceID func(S) string,
	mapFunc func(S) T,
) map[string][]T {
	// 先按ID分组源数据
	grouped := GroupBy(source, getSourceID)

	// 为每个目标ID组装数据
	result := make(map[string][]T, len(targetIDs))
	for _, targetID := range targetIDs {
		if sources, ok := grouped[targetID]; ok {
			mapped := make([]T, 0, len(sources))
			for _, s := range sources {
				mapped = append(mapped, mapFunc(s))
			}
			result[targetID] = mapped
		}
	}

	return result
}

// MergeMaps 合并多个map，后面的覆盖前面的
func MergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// MapSlice 将切片映射为另一个切片
func MapSlice[T any, R any](items []T, mapFunc func(T) R) []R {
	result := make([]R, 0, len(items))
	for _, item := range items {
		result = append(result, mapFunc(item))
	}
	return result
}

// FilterSlice 过滤切片
func FilterSlice[T any](items []T, filterFunc func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if filterFunc(item) {
			result = append(result, item)
		}
	}
	return result
}

// ReduceSlice 归约切片
func ReduceSlice[T any, R any](items []T, initial R, reduceFunc func(R, T) R) R {
	result := initial
	for _, item := range items {
		result = reduceFunc(result, item)
	}
	return result
}
