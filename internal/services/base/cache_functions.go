// =====================================================================
// Phase 92-01 (D-03/D-04): 缓存泛型包级函数族。
//
// 技术约束依据（92-RESEARCH 编译实证）：Go method 不能有自己的类型参数
// （`func (b *CacheServiceBase) Get[T any]()` → syntax error），泛型只能放
// 包级函数——本函数族是 CACHE-UNIFY-01 "模板方法"的实际形态（D-03）。
//
// 薄委托红线（Shared Pattern 3）：GetOrSetJSON 只做类型包装，委托
// p.GetOrSet，JSON 往返 / P0 #9 同步写语义逐字保留，禁止重写 Get/Set 逻辑
// （重新实现 = 行为漂移，违背 v1.29 D-05 零业务行为变更底线）。
// =====================================================================
package base

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// GetOrSetJSON 类型安全的读穿透缓存：命中反序列化，未命中执行 query 并同步写缓存。
//
// 薄包装：委托 p.GetOrSet（dest 传 &result，query 闭包适配 interface{} 签名），
// 语义与 provider.GetOrSet 逐字一致——JSON 往返、未命中回源后同步写缓存
// （P0 #9）、错误透传均由底层提供，本函数不重写任何 Get/Set 逻辑。
//
// 泛型 T 消除 interface{} 闭包 + var 中转 + NoOp.setValue 反射兜底三类样板
// （D-03）；T 为指针类型时经 &result 双指针 assignable 成立，NoOp 路径不再
// 静默丢零值（92-RESEARCH Pitfall 5，单向改善）。
func GetOrSetJSON[T any](
	ctx context.Context,
	p CacheProvider,
	key string,
	ttl time.Duration,
	query func() (T, error),
) (T, error) {
	var result T
	err := p.GetOrSet(ctx, key, &result, ttl, func() (interface{}, error) {
		return query()
	})
	return result, err
}

// Invalidate 按键列表失效缓存（D-04 唯一失效底层之一）。
//
// void 返回 + warn 日志（对齐 system.InvalidateCacheByKey 现状语义）；
// 统一日志格式取信息量大的 key 字段版（对齐 operations/cache_invalidator.go
// 的 pattern=%s 格式）。p == nil 时静默跳过（nil 防护内聚，消除调用点
// 重复判空）。
func Invalidate(ctx context.Context, p CacheProvider, keys []string, module string) {
	if p == nil {
		logger.Debugf("[%s] 未配置缓存提供者，跳过缓存清理", module)
		return
	}
	for _, key := range keys {
		if err := p.Delete(ctx, key); err != nil {
			logger.Warnf("[%s] 清除缓存失败: key=%s, error=%v", module, key, err)
		}
	}
}

// InvalidatePattern 按模式列表失效缓存（D-04 唯一失效底层之一）。
//
// 同构 Invalidate：循环 p.DeleteByPattern，void 返回 + warn 日志
// （pattern=%s 字段），p == nil 时静默跳过。
func InvalidatePattern(ctx context.Context, p CacheProvider, patterns []string, module string) {
	if p == nil {
		logger.Debugf("[%s] 未配置缓存提供者，跳过缓存清理", module)
		return
	}
	for _, pattern := range patterns {
		if err := p.DeleteByPattern(ctx, pattern); err != nil {
			logger.Warnf("[%s] 清除缓存失败: pattern=%s, error=%v", module, pattern, err)
		}
	}
}
