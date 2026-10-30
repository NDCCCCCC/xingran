//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 06: account_pool.go 剩余分支 (Task 5)
//
// 复用 account_pool_test.go 的 setupTestPool / insertAccount（D-78-06e 禁止重定义）。
// 本文件补：(a) RecoverExpiredBreakers 4 个用例 + DBError；(b) StartHotReload NilPubSub；
// (c) PickAvailable 多账号 + DBError；(d) InvalidateCache 可观察口径。
//
// 仅覆盖 redisPubSub == nil 形态（D-78-06a）；不引入 miniredis 测 Redis 订阅路径。
// 零生产 .go 改动。

package addomain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// markBroken78 把 insertAccount 创建的账号标记为熔断态（含 failure_count 与
// circuit_breaker_until），复用同包 insertAccount（D-78-06e 禁重定义）。
//
// until 可设为过去（应被 RecoverExpiredBreakers 恢复）或未来（不应被恢复）。
func markBroken78(t *testing.T, db *gorm.DB, accountID string, until time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(`
		UPDATE sys_ad_service_accounts
		SET status = ?, failure_count = ?, circuit_breaker_until = ?
		WHERE id = ?
	`, AccountStatusCircuitBroken, 3, until, accountID).Error)
}

// =============================================================================
// Task 5: account_pool 剩余分支
// =============================================================================

// TestPool78_RecoverExpiredBreakers_Basic 三形态预置：(a) 过期熔断 → 应被恢复；
// (b) 未过期熔断 → 不动；(c) 状态为 Available → 不动。断言 n=1、状态回 Available、
// circuit_breaker_until=NULL、failure_count=0。
func TestPool78_RecoverExpiredBreakers_Basic(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()
	now := time.Now()

	// (a) 过期熔断
	aID := insertAccount(t, db, configID, "acct-expired", AccountStatusAvailable)
	markBroken78(t, db, aID, now.Add(-1*time.Minute))
	// (b) 未过期熔断
	bID := insertAccount(t, db, configID, "acct-future", AccountStatusAvailable)
	markBroken78(t, db, bID, now.Add(10*time.Minute))
	// (c) 状态 Available（不应被改动）
	cID := insertAccount(t, db, configID, "acct-available", AccountStatusAvailable)
	require.NoError(t, db.Exec(`UPDATE sys_ad_service_accounts SET failure_count = 5 WHERE id = ?`, cID).Error)

	n, err := pool.RecoverExpiredBreakers(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "仅 (a) 过期熔断 1 条被恢复")

	// (a) 状态回 Available + until=NULL + failure_count=0（用 db.QueryRow 替代 GORM Scan 字段映射陷阱）
	var aStatus, aCount int
	var aUntil *time.Time
	require.NoError(t, db.Raw(`SELECT status, failure_count, circuit_breaker_until FROM sys_ad_service_accounts WHERE id = ?`, aID).
		Row().Scan(&aStatus, &aCount, &aUntil))
	assert.Equal(t, AccountStatusAvailable, aStatus)
	assert.Equal(t, 0, aCount)
	assert.Nil(t, aUntil, "circuit_breaker_until 应被置 NULL")

	// (b) 未过期熔断不应被改动
	var bStatus int
	require.NoError(t, db.Raw(`SELECT status FROM sys_ad_service_accounts WHERE id = ?`, bID).Row().Scan(&bStatus))
	assert.Equal(t, AccountStatusCircuitBroken, bStatus)

	// (c) Available 行不应被改动
	var cStatus, cCount int
	require.NoError(t, db.Raw(`SELECT status, failure_count FROM sys_ad_service_accounts WHERE id = ?`, cID).Row().Scan(&cStatus, &cCount))
	assert.Equal(t, AccountStatusAvailable, cStatus)
	assert.Equal(t, 5, cCount, "failure_count 应保持原值")
}

// TestPool78_RecoverExpiredBreakers_None 无过期熔断 → 返回 0 且无副作用。
func TestPool78_RecoverExpiredBreakers_None(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()

	// 几条非熔断账号
	insertAccount(t, db, configID, "acct-1", AccountStatusAvailable)
	insertAccount(t, db, configID, "acct-2", AccountStatusDisabled)

	n, err := pool.RecoverExpiredBreakers(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

// TestPool78_RecoverExpiredBreakers_MultiConfig 两个 configID 各有过期熔断账号 → 两者都恢复 + 各自缓存失效。
//
// 可观察口径：先 ListAvailable 预热缓存（account 直到 expire 前过滤 → 空）→ 用 raw UPDATE 把
// circuit_breaker_until 改成过去 → RecoverExpiredBreakers → 再 ListAvailable 应返回已恢复账号
// （若缓存未失效则仍为空 → 该断言即证明 InvalidateCache 生效）。
func TestPool78_RecoverExpiredBreakers_MultiConfig(t *testing.T) {
	pool, db, cfgA := setupTestPool(t)
	ctx := context.Background()
	cfgB := uuid.NewString()

	// cfgA 1 个过期熔断 + cfgB 1 个过期熔断
	aID := insertAccount(t, db, cfgA, "acct-A", AccountStatusAvailable)
	markBroken78(t, db, aID, time.Now().Add(10*time.Minute)) // until 在未来
	bID := insertAccount(t, db, cfgB, "acct-B", AccountStatusAvailable)
	markBroken78(t, db, bID, time.Now().Add(10*time.Minute))

	// 预热缓存：ListAvailable(cfgA) → [] (未到期熔断)；ListAvailable(cfgB) → []
	gotA, err := pool.ListAvailable(ctx, cfgA)
	require.NoError(t, err)
	gotB, err := pool.ListAvailable(ctx, cfgB)
	require.NoError(t, err)
	assert.Len(t, gotA, 0, "预热：cfgA 缓存 = []")
	assert.Len(t, gotB, 0, "预热：cfgB 缓存 = []")

	// 把两个 until 改成过去（直接 raw UPDATE 绕过 pool.Create 触发的缓存失效）
	require.NoError(t, db.Exec(`UPDATE sys_ad_service_accounts SET circuit_breaker_until = ? WHERE id IN (?, ?)`,
		time.Now().Add(-1*time.Minute), aID, bID).Error)

	// 恢复
	n, err := pool.RecoverExpiredBreakers(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n, "两个 configID 各 1 条过期熔断 → 总 2 条")

	// 缓存应已被 InvalidateCache（每 configID 一次，:523-525）
	gotA, err = pool.ListAvailable(ctx, cfgA)
	require.NoError(t, err)
	gotB, err = pool.ListAvailable(ctx, cfgB)
	require.NoError(t, err)
	assert.Len(t, gotA, 1, "cfgA 缓存失效 → 重新查询 → [aID]")
	assert.Len(t, gotB, 1, "cfgB 缓存失效 → 重新查询 → [bID]")
	assert.Equal(t, aID, gotA[0].ID)
	assert.Equal(t, bID, gotB[0].ID)
}

// TestPool78_RecoverExpiredBreakers_DBError DROP 表 → 返回 (0, error)。
func TestPool78_RecoverExpiredBreakers_DBError(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()
	insertAccount(t, db, configID, "acct", AccountStatusCircuitBroken)

	require.NoError(t, db.Exec(`DROP TABLE sys_ad_service_accounts`).Error)

	n, err := pool.RecoverExpiredBreakers(ctx)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.Contains(t, err.Error(), "no such table")
}

// TestPool78_RecoverExpiredBreakers_NoPubSub redisPubSub=nil → Publish 分支被跳过且不报错。
// setupTestPool 内部 NewAccountPool(db, nil) — redisPubSub=nil（account_pool.go:528 守卫）。
func TestPool78_RecoverExpiredBreakers_NoPubSub(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()
	id := insertAccount(t, db, configID, "acct-pubsub", AccountStatusAvailable)
	markBroken78(t, db, id, time.Now().Add(-1*time.Minute))

	n, err := pool.RecoverExpiredBreakers(ctx)
	require.NoError(t, err, "redisPubSub=nil 时 Publish 分支被跳过（account_pool.go:528 守卫）")
	assert.Equal(t, 1, n)
}

// TestPool78_StartHotReload_NilPubSub NewAccountPool(db, nil).StartHotReload(ctx) → 返回 nil。
//
// D-78-06a：仅覆盖 redisPubSub == nil 早退路径；不引入 miniredis 测 Redis 订阅路径
// （78-04 已用尽 miniredis 预算，pubsub 订阅是跨进程语义单测价值低）。
func TestPool78_StartHotReload_NilPubSub(t *testing.T) {
	_, db, _ := setupTestPool(t)
	defer closeDB(t, db)
	pool := NewAccountPool(db, nil)

	require.NoError(t, pool.StartHotReload(context.Background()),
		"D-78-06a: redisPubSub=nil → StartHotReload 早退 (account_pool.go:554-557)")
}

// TestPool78_PickAvailable_Branches 补 PickAvailable 剩余分支（基线 44.4%）。
//
//   - 空池 → ErrAllAccountsUnavailable
//   - 单账号 → 返回该账号
//   - 多账号 → 返回值 ∈ 集合（D-78-06d 禁断言具体挑中项）
//   - ListAvailable 错误透传（DROP 表）
func TestPool78_PickAvailable_Branches(t *testing.T) {
	t.Run("EmptyPool", func(t *testing.T) {
		pool, _, configID := setupTestPool(t)
		ctx := context.Background()
		_, err := pool.PickAvailable(ctx, configID)
		assert.True(t, errors.Is(err, ErrAllAccountsUnavailable), "空池应返回 ErrAllAccountsUnavailable")
	})

	t.Run("SingleAccount", func(t *testing.T) {
		pool, db, configID := setupTestPool(t)
		ctx := context.Background()
		id := insertAccount(t, db, configID, "only-one", AccountStatusAvailable)
		got, err := pool.PickAvailable(ctx, configID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, id, got.ID)
	})

	t.Run("MultiAccounts", func(t *testing.T) {
		pool, db, configID := setupTestPool(t)
		ctx := context.Background()
		ids := map[string]bool{}
		for i := 0; i < 5; i++ {
			id := insertAccount(t, db, configID, "svc-"+uuid.NewString()[:6], AccountStatusAvailable)
			ids[id] = true
		}
		got, err := pool.PickAvailable(ctx, configID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.True(t, ids[got.ID], "返回值 ∈ 可用账号集合 (D-78-06d:禁断言具体挑中)")
	})

	t.Run("DBError", func(t *testing.T) {
		pool, db, configID := setupTestPool(t)
		ctx := context.Background()
		insertAccount(t, db, configID, "x", AccountStatusAvailable)
		require.NoError(t, db.Exec(`DROP TABLE sys_ad_service_accounts`).Error)
		_, err := pool.PickAvailable(ctx, configID)
		require.Error(t, err, "ListAvailable DB 错误应透传")
	})
}

// TestPool78_InvalidateCache 可观察口径：warm → mutate DB（绕过 pool.Create 触发失效）
// → ListAvailable 仍走缓存（命中）→ InvalidateCache → ListAvailable 落 DB。
func TestPool78_InvalidateCache(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()

	// 1. 初始空
	got, err := pool.ListAvailable(ctx, configID)
	require.NoError(t, err)
	assert.Len(t, got, 0)

	// 2. 直接 INSERT（绕过 pool.Create 自动失效）→ 模拟"另一进程写入"
	id2 := uuid.NewString()
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_service_accounts
		(id, config_id, username, password_ciphertext, status, failure_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, ?, ?)
	`, id2, configID, "ghost-account", "encrypted_pwd", AccountStatusAvailable, now, now).Error)

	// 3. ListAvailable 命中缓存 → 仍 0 条
	got, err = pool.ListAvailable(ctx, configID)
	require.NoError(t, err)
	assert.Len(t, got, 0, "30s 缓存 TTL 内不应重读 DB")

	// 4. 显式 InvalidateCache
	pool.InvalidateCache(configID)

	// 5. ListAvailable 落 DB → 1 条
	got, err = pool.ListAvailable(ctx, configID)
	require.NoError(t, err)
	assert.Len(t, got, 1, "InvalidateCache 后下次查询应回落 DB")
	assert.Equal(t, id2, got[0].ID)
}

// TestPool78_RecoverExpiredBreakers_Idempotent 多次连续调用稳定（幂等性）。
// 3 个过期熔断行 → 第一次恢复 3 条 → 第二次恢复 0 条（已恢复）。
func TestPool78_RecoverExpiredBreakers_Idempotent(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		id := insertAccount(t, db, configID, "idem-"+uuid.NewString()[:6], AccountStatusAvailable)
		markBroken78(t, db, id, time.Now().Add(-2*time.Minute))
	}

	n, err := pool.RecoverExpiredBreakers(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = pool.RecoverExpiredBreakers(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "第二次调用应恢复 0 条（已全部恢复）")

	var availableCnt int64
	require.NoError(t, db.Model(&models.ADServiceAccount{}).
		Where("config_id = ? AND status = ?", configID, AccountStatusAvailable).
		Count(&availableCnt).Error)
	assert.EqualValues(t, 3, availableCnt)
}

// =============================================================================
// 覆盖边界块注释
// =============================================================================
//
// D-78-06a StartHotReload 仅覆盖 redisPubSub == nil 早退路径（account_pool.go:553-557）；
// 不引入 miniredis 测 Redis pub/sub 订阅路径。
//   - 订阅是跨进程语义，单测价值低；
//   - miniredis 预算已分配给 78-01 (captcha 限流) 与 78-02 (core initCache)；
//   - 78-RESEARCH §3 表格未把 account_pool pubsub 列入 Phase 78 范围（标 LOW/"已覆盖主体"）。
// 若 78-07 或后续 plan 需要 Redis pub/sub 真实路径，需评估是否引入 miniredis 或本进程 fake broker。