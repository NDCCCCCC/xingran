// =====================================================================
// Phase 92-01 (D-02): 缓存抽象全套已迁至 internal/services/base 单一权威位置。
//
// 本文件原位保留 type alias（Go spec：alias = 同一类型的替代名，无适配层），
// 20+ 消费文件零改动编译通过——api/router.go、6 处 NoOpCacheProvider 路由
// fallback、duty/knowledge/network/workorder 测试的 var _ 编译期断言、
// system.WithCacheProvider（user_sync_service.go）、core.go 装配等。
// alias 后续可删（D-02 锁定，删除属 92-03 收尾范围）。
// =====================================================================
package system

import (
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// CacheProvider 缓存提供者接口（权威定义：base.CacheProvider）
type CacheProvider = base.CacheProvider

// NoOpCacheProvider 空缓存提供者（权威定义：base.NoOpCacheProvider）
type NoOpCacheProvider = base.NoOpCacheProvider

// CacheStats 缓存统计信息（权威定义：base.CacheStats）
type CacheStats = base.CacheStats

// CacheEntry 缓存条目详情（权威定义：base.CacheEntry）
type CacheEntry = base.CacheEntry

// CacheServiceBase 缓存服务基础结构（权威定义：base.CacheServiceBase，
// D-01 TTLResolver 薄基类；10 个嵌入点构造字面量零改动）
type CacheServiceBase = base.CacheServiceBase

// 编译期双保险：NoOpCacheProvider 满足 CacheProvider 全部 9 方法
var _ CacheProvider = (*NoOpCacheProvider)(nil)
