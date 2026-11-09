// Package query: Phase 151 / ARCH-01 资源归属注册表（pkg 层）。
//
// 分层硬约束（PLAN §3 F-4 / §4.2）：
//   - 本文件只允许 import context / gorm.io/gorm / pkg/datascope；
//     **不得 import internal/** **（避免 import cycle，保持 pkg/datascope
//     「不感知 ops 业务表名」的分层契约）。
//   - pkg 层只存「表名 → 回调」，零业务表名常量、零 SQL、零档位判定；
//     谓词的唯一实现由 internal/services/operations/scope_registry.go 在
//     ops 包内注册（引用 149 的 entityExistsSub / effectiveOrgExpr /
//     wrapScopeExists / dept_id 直挂，禁止内联复制）。
package query

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
)

// ScopeFactory 由 ops 侧注册的「归属链谓词」构造器。
//
// tx 是当前 GORM 链（Scopes 回调执行时 Statement.Context 已带请求 ctx），
// r 是 datascope.FromContext 解析出的 ScopeResult（单一真实源、本层不重裁定）。
// 实现必须引用既有 scope_filter.go / scope_entities.go 的函数，禁止内联复制
// COALESCE/NULLIF/EXISTS 骨架。
type ScopeFactory func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB

// Entry 一张业务表的注册项。
type Entry struct {
	// Predicate 归属链谓词（引用既有单一真实源）。
	Predicate ScopeFactory
	// CacheKey 由 ctx 派生缓存 key 的 scope 维度段（委托 ops 侧 ScopedCacheKey）。
	CacheKey func(ctx context.Context) string
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Entry)
)

// Register 登记一张业务表的归属链谓词。
//
// 重复注册返回 error（项目禁 panic，AGENTS.md「Return errors, don't panic」）；
// MustRegister 供启动期装配使用，内部 logrus.Fatalf。
func Register(table string, e Entry) error {
	if table == "" {
		return fmt.Errorf("query: 注册表名不能为空")
	}
	if e.Predicate == nil {
		return fmt.Errorf("query: 表 %q 的 Predicate 不能为 nil", table)
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[table]; exists {
		return fmt.Errorf("query: 表 %q 重复注册（单一真实源禁止同表多套谓词）", table)
	}
	registry[table] = e
	return nil
}

// MustRegister 是 Register 的启动期包装：注册失败直接 logrus.Fatalf。
// 仅在 ops 侧 SetupScopedDBRegistry 装配路径使用；返回 error 的路径请用 Register。
func MustRegister(table string, e Entry) {
	if err := Register(table, e); err != nil {
		logrus.Fatalf("Phase 151 ScopedDB 注册表装配失败: %v", err)
	}
}

// Lookup 按表名取注册项；未登记返回 found=false。
func Lookup(table string) (Entry, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	e, ok := registry[table]
	return e, ok
}

// RegisteredTables 返回已登记表名的稳定排序快照（供静态断言测试枚举）。
func RegisteredTables() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for t := range registry {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// resetRegistryForTest 仅供本包单测清空注册表，避免测试间相互污染。
// 生产路径不调用（注册表只在启动期装配一次）。
func resetRegistryForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]Entry)
}

// ResetRegistryForTest 是 resetRegistryForTest 的导出包装，供其它包
// （internal/services/operations）的就近 *_test.go 在装配前清空注册表。
// 生产路径不调用。
func ResetRegistryForTest() { resetRegistryForTest() }
