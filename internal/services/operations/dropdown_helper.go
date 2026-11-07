package operations

import (
	"context"
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

// BuildScopedDropdownCacheKey 在 BuildDropdownCacheKey 基础上追加「scope 维度」段
// （PMODEL-01 防跨范围泄漏）。Phase 131 所有 dropdown-options 调用点必须改用本函数。
//
// key 形态：dropdown:{entity}:s{scopeKey}:{all|h<hash>}。
// scopeKey 由 ScopeResult 稳定派生（scopeCacheKey）：
//   - 未注入 / All → "all"（超管短路路径同样带 scope 段，LOCKED §1 关键不变量 3）
//   - FailClosed → "closed"；Self → "self<hash>"；DeptIDs → "d<hash>"
//
// 保留旧 BuildDropdownCacheKey 的签名与逐字行为（dropdown_helper_test.go 既有断言
// 不破）：本函数只在其结果里插入一段，不改动旧函数任何一行。
//
// ctx 用于读取 ScopeResult；调用方应传 service 收到的请求 ctx（携带注入结果）。
func BuildScopedDropdownCacheKey(ctx context.Context, entity string, filters map[string]any) string {
	baseKey := BuildDropdownCacheKey(entity, filters)
	sk := scopeCacheKey(ctx)
	return strings.Replace(baseKey, dropdownCacheKeyPrefix+":"+entity+":",
		dropdownCacheKeyPrefix+":"+entity+":s"+sk+":", 1)
}