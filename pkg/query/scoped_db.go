// Package query: Phase 151 / ARCH-01 统一 ScopedDB（GORM Scopes 唯一查询起链点）。
//
// 设计契约（PLAN §4.2 / §3 F-1、F-2、F-11）：
//   - 唯一查询起链点：所有 ops 读链的 `s.db.WithContext(ctx)` 一律改走
//     `ScopedQuery`/`ScopedDB`，由 GORM `Scopes` 在 SQL 构建时按表名查注册表
//     强制挂可见集谓词；禁止裸起链（静态断言守护）。
//   - 零回归短路：`datascope.FromContext(ctx)` 返回 ok=false（未挂注入中间件，
//     如 context.Background() 直调的 SQLite 单测 / 内部任务）或 r.All=true 时，
//     **原样返回 db.WithContext(ctx)**，零额外 SQL、零额外 WHERE（硬约束）。
//   - 单一真实源：谓词由 ops 侧注册表回调引用 scope_filter.go /
//     scope_entities.go 的既有实现；本文件不内联复制 COALESCE/NULLIF/1=0/
//     EXISTS 骨架，不 import internal/**。
//   - 写链禁挂：GORM Scopes 会把谓词带进 UPDATE/DELETE 的 WHERE，故
//     GORMRepository 的 Create/Update/Delete/BatchDelete 保持裸起链（红线）。
package query

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
)

// resolveTable 在 Scopes 回调内解析当前链的业务表名。
//
// 关键实证：GORM 的 Scopes 回调在 Statement.Parse **之前**执行，此时
// Statement.Table 与 Statement.Schema 通常为空，但 Statement.Model 已就位；
// 故优先级为 Table → Schema.Table → Model 的 Tabler 断言（与
// internal/services/base/service.go:83-89 同款）。取不到表名时返回空串，
// 由 ScopedDB 跳过谓词（不静默放行全库：ops 起链「必登记」由静态断言兜住）。
func resolveTable(tx *gorm.DB) string {
	if tx == nil || tx.Statement == nil {
		return ""
	}
	if tx.Statement.Table != "" {
		return tx.Statement.Table
	}
	if tx.Statement.Schema != nil && tx.Statement.Schema.Table != "" {
		return tx.Statement.Schema.Table
	}
	if tabler, ok := tx.Statement.Model.(interface{ TableName() string }); ok {
		return tabler.TableName()
	}
	return ""
}

// ScopedDB 是 ops 读链的统一查询起链点。
//
//	未注入 ctx / All 档 → 原样返回 db.WithContext(ctx)（零回归短路）；
//	其余档位         → db.WithContext(ctx).Scopes(scopeFn)，scopeFn 在
//	                    SQL 构建时按 resolveTable 查注册表并应用谓词。
//
// FailClosed 由注册表谓词统一处理（1=0 空集），本函数不另写 1=0，
// 避免出现第二套档位判定实现。
func ScopedDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	r, ok := datascope.FromContext(ctx)
	if !ok || r.All {
		return db.WithContext(ctx)
	}
	return db.WithContext(ctx).Scopes(func(tx *gorm.DB) *gorm.DB {
		table := resolveTable(tx)
		if table == "" {
			return tx
		}
		e, found := Lookup(table)
		if !found || e.Predicate == nil {
			return tx
		}
		return e.Predicate(tx, r)
	})
}

// ScopedDBPending 把已带 ctx 的链再包一层 Scopes（供 base.Scope 兼容备选，
// 主路径以 service.go 起链替换为准；避免双起链）。
func ScopedDBPending(tx *gorm.DB, ctx context.Context) *gorm.DB {
	r, ok := datascope.FromContext(ctx)
	if !ok || r.All {
		return tx
	}
	return tx.Scopes(func(q *gorm.DB) *gorm.DB {
		table := resolveTable(q)
		if table == "" {
			return q
		}
		e, found := Lookup(table)
		if !found || e.Predicate == nil {
			return q
		}
		return e.Predicate(q, r)
	})
}

// ScopeFor 返回与 base.Scope 同形的兼容闭包（可选；主路径是起链替换）。
func ScopeFor(ctx context.Context) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		return ScopedDBPending(q, ctx)
	}
}

// ScopeCacheKey 由 ctx 派生缓存 key 的 scope 维度段。
//
// 未注入 / All → "all"（与 scope_filter.go:59-62 一致的短路常量，非算法复制）；
// 其余档位委托注册表首个可用 CacheKey 回调（ops 侧注册 ScopedCacheKey，
// 唯一算法实现仍是 scope_filter.go:53-74 scopeCacheKey）。零算法复制。
func ScopeCacheKey(ctx context.Context) string {
	r, ok := datascope.FromContext(ctx)
	if !ok || r.All {
		return "all"
	}
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, e := range registry {
		if e.CacheKey != nil {
			return e.CacheKey(ctx)
		}
	}
	return "all"
}
