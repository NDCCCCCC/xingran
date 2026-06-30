package operations

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DropdownOption 是 dropdown-options 端点的统一响应元素。
// value/label 模式与 antd Select 的 options={[{value, label}]} 直接对齐。
type DropdownOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// DropdownMaxRows 单次返回的最大行数。硬上限,防止前端用 Select 当全集源。
const DropdownMaxRows = 50

// dropdownCacheKeyPrefix 是 Redis 缓存 key 前缀,与其他 cache key 区分。
const dropdownCacheKeyPrefix = "dropdown"

// BuildDropdownCacheKey 根据 entity 与 filters 生成稳定的缓存 key。
//
// 规则:
//   - filters 为空 → "dropdown:{entity}:all"
//   - filters 非空 → 排序后 JSON 序列化 → SHA1 截断 → "dropdown:{entity}:h{hash}"
//
// 排序保证同一语义 filter 集合始终生成同一 key(避免 map 遍历顺序导致 key 抖动)。
func BuildDropdownCacheKey(entity string, filters map[string]any) string {
	if len(filters) == 0 {
		return fmt.Sprintf("%s:%s:all", dropdownCacheKeyPrefix, entity)
	}
	// 提取 key 并排序,保证稳定性
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := filters[k]
		// nil 值不计入缓存 key,避免 nil/"" 不一致
		if v == nil {
			continue
		}
		b, _ := json.Marshal(v)
		parts = append(parts, fmt.Sprintf("%s=%s", k, string(b)))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s:%s:all", dropdownCacheKeyPrefix, entity)
	}
	h := sha1.Sum([]byte(strings.Join(parts, "&")))
	return fmt.Sprintf("%s:%s:h%s", dropdownCacheKeyPrefix, entity, hex.EncodeToString(h[:8]))
}