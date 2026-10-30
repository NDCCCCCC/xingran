package asset

import (
	"context"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// Cache key 常量模板 — 资产对账模块
//
// 命名规则: `reconciliation:<sub>:<suffix>`
//
// Redis 前缀处理 (CLAUDE.md Cache Key Prefix Handling):
//   - 实际 Redis key 会自动加 `xingran:` 前缀,本文件定义的 key 是"逻辑键"
//   - 调用 cache.Set(ctx, key, ...) 时,直接传本文件的 key 即可,不要手动加前缀
//   - 从 UI / 用户输入读 key 时,若已含 `xingran:` 前缀,需 strip 后再操作
//
// R1/R2/R3 演进:
//   - R1: 仅 cache_keys 文件就位,无运行时调用(INFRA-03 占位)
//   - R2: F1/F3 启用 reconciliation:dashboard / reconciliation:exception:list / reconciliation:view:lastRefresh
//   - R3: F4 启用 reconciliation:exceptionRule:list / reconciliation:exceptionRule:byID
//   - R4: F5 启用 reconciliation:health:workstation:* / reconciliation:health:asset:*
const (
	// CacheKeyReconciliationDashboard Dashboard 数据缓存 key(按 scope 分桶)
	// scope: global / dept / workstation 等
	CacheKeyReconciliationDashboard = "reconciliation:dashboard:%s"

	// CacheKeyReconciliationExceptionList 异常列表分页结果缓存 key(按 scope + 分页条件 hash)
	CacheKeyReconciliationExceptionList = "reconciliation:exception:list:%s"

	// CacheKeyReconciliationExceptionByID 单条异常详情缓存 key
	CacheKeyReconciliationExceptionByID = "reconciliation:exception:byID:%s"

	// CacheKeyReconciliationExceptionRuleList 例外规则列表缓存 key (R3 启用)
	CacheKeyReconciliationExceptionRuleList = "reconciliation:exceptionRule:list"

	// CacheKeyReconciliationExceptionRuleByID 单条例外规则详情缓存 key (R3 启用)
	CacheKeyReconciliationExceptionRuleByID = "reconciliation:exceptionRule:byID:%s"

	// CacheKeyReconciliationViewLastRefresh 物化视图最近刷新时间缓存 key
	// 用于 dashboard 显示 "数据更新于 X 分钟前"
	CacheKeyReconciliationViewLastRefresh = "reconciliation:view:lastRefresh"

	// CacheKeyReconciliationHealthByWorkstation 工位健康度缓存 key (R4 启用)
	CacheKeyReconciliationHealthByWorkstation = "reconciliation:health:workstation:%s"

	// CacheKeyReconciliationHealthByAsset 资产健康度缓存 key (R4 启用)
	CacheKeyReconciliationHealthByAsset = "reconciliation:health:asset:%s"
)

// GetReconciliationDashboardKey 根据 scope 构建 dashboard 缓存键
// 参数:scope — global / dept / workstation 等
// 返回:reconciliation:dashboard:{scope}
func GetReconciliationDashboardKey(scope string) string {
	return fmt.Sprintf(CacheKeyReconciliationDashboard, scope)
}

// GetReconciliationExceptionListKey 根据 scope 构建异常列表缓存键
func GetReconciliationExceptionListKey(scope string) string {
	return fmt.Sprintf(CacheKeyReconciliationExceptionList, scope)
}

// GetReconciliationExceptionByIDKey 根据异常 ID 构建详情缓存键
func GetReconciliationExceptionByIDKey(id string) string {
	return fmt.Sprintf(CacheKeyReconciliationExceptionByID, id)
}

// GetReconciliationExceptionRuleListKey 返回例外规则列表缓存键
func GetReconciliationExceptionRuleListKey() string {
	return CacheKeyReconciliationExceptionRuleList
}

// GetReconciliationExceptionRuleByIDKey 根据例外规则 ID 构建详情缓存键
func GetReconciliationExceptionRuleByIDKey(id string) string {
	return fmt.Sprintf(CacheKeyReconciliationExceptionRuleByID, id)
}

// GetReconciliationViewLastRefreshKey 返回物化视图最近刷新时间缓存键
func GetReconciliationViewLastRefreshKey() string {
	return CacheKeyReconciliationViewLastRefresh
}

// GetReconciliationHealthByWorkstationKey 根据工位 ID 构建健康度缓存键
func GetReconciliationHealthByWorkstationKey(workstationID string) string {
	return fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, workstationID)
}

// GetReconciliationHealthByAssetKey 根据资产 ID 构建健康度缓存键
func GetReconciliationHealthByAssetKey(assetID string) string {
	return fmt.Sprintf(CacheKeyReconciliationHealthByAsset, assetID)
}

// StripCachePrefix 去掉 Redis 自动加上的 `xingran:` 前缀,用于用户输入的 key
//
// CLAUDE.md Cache Prefix Handling:
//   - 缓存写入时自动加 `xingran:` 前缀,本文件定义的 key 是"逻辑键"(无前缀)
//   - 当从 UI / 用户输入读取 key 时,若已含 `xingran:` 前缀,需先 strip 再做 cache 操作
//
// 典型用法 (R2+):
//
//	if userKey := c.Query("cacheKey"); userKey != "" {
//	    rawKey := asset.StripCachePrefix(userKey)
//	    cache.Delete(ctx, rawKey)
//	}
func StripCachePrefix(key string) string {
	const prefix = "xingran:"
	return strings.TrimPrefix(key, prefix)
}

// InvalidateWorkstationHealth 删除该工位的健康度缓存 (D-A4-04 R4 触发点)
//
// 调用顺序约定 (D-A4-04 锁定):
//
//	asset.InvalidateWorkstationHealth(ctx, c, wsID)  // 先失效
//	operlog.Record(...)                              // 再写日志
//	response.Success(c, ...)                         // 最后返响应
//
// 参数:
//   - ctx: 请求上下文
//   - c:   cache.Cache 实例(nil-safe,nil 时直接返回 nil)
//   - workstationID: 工位 uuid
//
// 返回:
//   - error: 来自 cache.Delete(Redis 删除失败);调用方决定是否 abort
//
// 本 plan (45-01) 仅定义,实际调用在 Plan 02 (ResolveException + R2 scheduler)。
func InvalidateWorkstationHealth(ctx context.Context, c cache.Cache, workstationID string) error {
	if c == nil || workstationID == "" {
		return nil
	}
	return c.Delete(ctx, GetReconciliationHealthByWorkstationKey(workstationID))
}