package services

// =====================================================================
// Phase 79-01 Task 3 + Task 4: SC-2 具名文件清欠 + 缓存基建小文件收尾。
//
// 覆盖目标(79-RESEARCH §2 基线):
//   - token_blacklist_service.go   73.3% → ≥85%(SC-2 点名:RemoveFromBlacklist :129
//     从 0% 清起 + rememberNegative :115 容量/过期尾支 + AddToBlacklist :61 尾支)
//   - template_cache.go            0%    → ≥70%(Get 未命中/命中/错误 + Clear)
//   - mac_history_cache_decorator.go 0%  → 100%(BuildMACQueryCacheKey 4 前缀 + 未知
//     方法 + 序列化失败)
//   - rate_limiter.go              84.1% → ≥90%(calculateRemaining / calculateReset
//     空切片 / getOrCreateWindow 复用 / cleanOlderThan cutoff)
//   - mac_normalize.go             93.3% → 100%(isCanonicalMAC 尾支)
//
// 纪律:helper 带 plan 后缀(R5);禁 t.Parallel();禁裸 time.Sleep;
// t.Cleanup 单次 Close;状态/语义断言禁裸 0/1 字面量。
// =====================================================================

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// ============================== Task 3: token_blacklist ==============================

// newTbl7901 装配 TokenBlacklistService + MemoryCache(t.Cleanup 单次 Close)。
func newTbl7901(t *testing.T) (TokenBlacklistService, *cache.MemoryCache) {
	t.Helper()
	mc := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mc.Close() })
	return NewTokenBlacklistService(mc), mc
}

// tbl7901SetDown / tbl7901DelDown 故障哨兵错误(%w 包装,可用 errors.Is 解包)。
var (
	errTbl7901SetDown = errors.New("tbl7901: redis set down")
	errTbl7901DelDown = errors.New("tbl7901: redis del down")
)

// tbl7901FailCache 可注入 Set/Delete 故障的 cache.Cache 装饰器,驱动
// AddToBlacklist(:72)/ RemoveFromBlacklist(:132)的错误包装分支。
type tbl7901FailCache struct {
	cache.Cache
	failSet    bool
	failDelete bool
}

func (m *tbl7901FailCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if m.failSet {
		return errTbl7901SetDown
	}
	return m.Cache.Set(ctx, key, value, expiration)
}

func (m *tbl7901FailCache) Delete(ctx context.Context, key string) error {
	if m.failDelete {
		return errTbl7901DelDown
	}
	return m.Cache.Delete(ctx, key)
}

// TestTbl7901_RemoveFromBlacklist_AfterAdd AddToBlacklist → IsBlacklisted true →
// RemoveFromBlacklist → IsBlacklisted false(:129 0% 主力缺口,SC-2 点名);
// 移除不存在的 token → 不报错。
func TestTbl7901_RemoveFromBlacklist_AfterAdd(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTbl7901(t)
	token := "tbl7901-remove-target"

	require.NoError(t, svc.AddToBlacklist(ctx, token, time.Now().Add(time.Hour)))
	blacklisted, err := svc.IsBlacklisted(ctx, token)
	require.NoError(t, err)
	assert.True(t, blacklisted, "拉黑后必须命中")

	require.NoError(t, svc.RemoveFromBlacklist(ctx, token), ":129 RemoveFromBlacklist 不应报错")
	blacklisted, err = svc.IsBlacklisted(ctx, token)
	require.NoError(t, err)
	assert.False(t, blacklisted, "RemoveFromBlacklist 后 IsBlacklisted=false(SC-2 点名缺口)")

	require.NoError(t, svc.RemoveFromBlacklist(ctx, "tbl7901-never-added"),
		"移除不存在的 token 应为幂等 no-op")
}

// TestTbl7901_RemoveFromBlacklist_ErrorBranch 底层 Delete 故障 →
// "从令牌黑名单移除失败: %w"(:131-133),负缓存仍被清除(删除语义优先于缓存一致性)。
func TestTbl7901_RemoveFromBlacklist_ErrorBranch(t *testing.T) {
	ctx := context.Background()
	_, mem := newTbl7901(t)
	svc := NewTokenBlacklistService(&tbl7901FailCache{Cache: mem, failDelete: true})

	require.NoError(t, mem.Set(ctx, "token:blacklist:tbl7901-del-fail", "1", time.Hour))
	err := svc.RemoveFromBlacklist(ctx, "tbl7901-del-fail")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "从令牌黑名单移除失败")
	assert.ErrorIs(t, err, errTbl7901DelDown, "错误应以 %w 包装,支持 errors.Is 解包")
}

// TestTbl7901_AddToBlacklist_TailBranch 对照 :61-82 实现补未覆盖尾支:
// 已过期 token(ttl <= 0)不写缓存直接返回 nil;写入故障 → "加入令牌黑名单失败";
// happy path 键 TTL 与 expiry 语义一致。
func TestTbl7901_AddToBlacklist_TailBranch(t *testing.T) {
	ctx := context.Background()
	svc, mem := newTbl7901(t)
	impl := svc.(*tokenBlacklistService)

	// 尾支 1:令牌已过期(ttl <= 0)→ 无需加入黑名单,键不得落库(:65-68)
	expired := "tbl7901-already-expired"
	require.NoError(t, svc.AddToBlacklist(ctx, expired, now79().Add(-time.Minute)))
	exists, err := mem.Exists(ctx, impl.getBlacklistKey(expired))
	require.NoError(t, err)
	assert.False(t, exists, "已过期 token 不应写入黑名单键")

	// 尾支 2:底层 Set 故障 → "加入令牌黑名单失败: %w"(:71-73)
	failSvc := NewTokenBlacklistService(&tbl7901FailCache{Cache: mem, failSet: true})
	err = failSvc.AddToBlacklist(ctx, "tbl7901-set-fail", now79().Add(time.Hour))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "加入令牌黑名单失败")

	// happy path:写入键 TTL ≈ expiry - now(1 小时档内,与令牌过期语义一致)
	target := "tbl7901-ttl-check"
	expiry := now79().Add(time.Hour)
	require.NoError(t, svc.AddToBlacklist(ctx, target, expiry))
	ttl, err := mem.TTL(ctx, impl.getBlacklistKey(target))
	require.NoError(t, err)
	assert.Greater(t, ttl, 55*time.Minute, "黑名单键 TTL 应接近令牌剩余寿命")
	assert.LessOrEqual(t, ttl, time.Hour, "黑名单键 TTL 不得超过令牌过期时间")
}

// TestTbl7901_RememberNegative_TailBranch 同包白盒驱动 :115 rememberNegative:
// 容量上限触发时先清理全部过期项;过期负缓存不再命中(回源并刷新时间戳)。
func TestTbl7901_RememberNegative_TailBranch(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTbl7901(t)
	impl := svc.(*tokenBlacklistService)
	now := now79()

	// 预填容量上限个已过期负缓存项 → 触发 :118-124 清理循环
	for i := 0; i < blacklistNegativeMaxEntries; i++ {
		impl.negUntil[fmt.Sprintf("tbl7901-stale-%d", i)] = now.Add(-time.Hour)
	}
	require.Len(t, impl.negUntil, blacklistNegativeMaxEntries)

	impl.rememberNegative("tbl7901-fresh", now)

	assert.Len(t, impl.negUntil, 1, "容量上限触发时应清掉全部过期负缓存项,仅剩新项")
	until, ok := impl.negUntil["tbl7901-fresh"]
	require.True(t, ok, "新负缓存项必须写入")
	assert.True(t, until.After(now), "新项应记录 now+blacklistNegativeTTL")

	// 过期尾支:已过期负缓存不命中,IsBlacklisted 回源并刷新时间戳
	staleSvc, _ := newTbl7901(t)
	staleImpl := staleSvc.(*tokenBlacklistService)
	staleImpl.negUntil["tbl7901-expired-entry"] = now.Add(-time.Second)
	staleBefore := staleImpl.negUntil["tbl7901-expired-entry"]

	blacklisted, err := staleImpl.IsBlacklisted(ctx, "tbl7901-expired-entry")
	require.NoError(t, err)
	assert.False(t, blacklisted, "过期负缓存不得放行判定为「已缓存」,必须回源")
	staleAfter, ok := staleImpl.negUntil["tbl7901-expired-entry"]
	require.True(t, ok)
	assert.True(t, staleAfter.After(staleBefore), "回源后负缓存时间戳应被刷新")
}

// TestTbl7901_GetBlacklistKey 同包直调 getBlacklistKey,断言键前缀形态(:141,
// 引用 constants.TokenBlacklistKeyFormat,禁裸字符串)。
func TestTbl7901_GetBlacklistKey(t *testing.T) {
	svc, _ := newTbl7901(t)
	impl := svc.(*tokenBlacklistService)

	assert.Equal(t, fmt.Sprintf(constants.TokenBlacklistKeyFormat, "abc"), impl.getBlacklistKey("abc"))
	assert.Equal(t, "token:blacklist:abc", impl.getBlacklistKey("abc"),
		"键形态应与 constants.TokenBlacklistKeyFormat 一致")
	assert.Equal(t, "token:blacklist:", impl.getBlacklistKey(""), "空 token 也应保持前缀形态")
}

// ============================== Task 3: template_cache ==============================

// tmc7901SourceTemplate 仓库内嵌真实 TextFSM 模板(相对 internal/services 包目录),
// 复制到 t.TempDir() 作为 fixture(威胁模型:禁写仓库模板目录)。
const tmc7901SourceTemplate = "../templates/embedded/templates/huawei_ont_display_lanmac.textfsm"

// newTmc7901 装配 TemplateCache + 真实 TextFSM 模板副本(保证解析合法)。
func newTmc7901(t *testing.T) (*TemplateCache, string) {
	t.Helper()
	src, err := os.ReadFile(filepath.FromSlash(tmc7901SourceTemplate))
	require.NoError(t, err, "仓库内嵌 TextFSM 模板应可读(fixture 来源)")
	path := filepath.Join(t.TempDir(), "lanmac_79_01.textfsm")
	require.NoError(t, os.WriteFile(path, src, 0o644))
	return NewTemplateCache(), path
}

// TestTmc7901_Get_MissHitError 三态 + Clear:
// 首次 Get 未命中 → 解析并写缓存;同路径二次 Get 命中(可观察证据:源文件删除后仍成功
// = 走缓存而非回源);不存在路径 → error;Clear 后重新解析(源文件已删 → 报错证明缓存已清)。
func TestTmc7901_Get_MissHitError(t *testing.T) {
	tc, path := newTmc7901(t)

	// 未命中 → 解析模板 → 写缓存
	fsm1, err := tc.Get(path)
	require.NoError(t, err, "合法 TextFSM 模板首次 Get 应解析成功")
	require.NotNil(t, fsm1)

	// 命中证据:删除源文件后二次 Get 仍成功 → 未回源,走 sync.Map/map 缓存
	require.NoError(t, os.Remove(path))
	fsm2, err := tc.Get(path)
	require.NoError(t, err, "源文件已删除仍成功 = 命中缓存而非回源")
	assert.Same(t, fsm1, fsm2, "命中应返回同一 *templates.FSM 实例")

	// 错误态:不存在路径 → error
	_, err = tc.Get(filepath.Join(t.TempDir(), "no-such_79_01.textfsm"))
	require.Error(t, err, "不存在的模板路径必须报错")

	// Clear 后缓存已清:同路径重新解析 → 源文件已删 → 报错(证明 Clear 生效)
	tc.Clear()
	_, err = tc.Get(path)
	require.Error(t, err, "Clear 后应重新解析;源文件已删则必须报错(缓存已清的证据)")
}

// now79 统一取当前时间(命名带后缀,防与其他 plan helper 撞名)。
func now79() time.Time { return time.Now() }
