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
	"strings"
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

func (m *tbl7901FailCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
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

// ==================== Task 4: mac_history_cache_decorator ====================

// TestMcd7901_BuildKey_AllPrefixes 4 个合法 method 各断言键前缀 == 对应
// cacheKeyPrefix* 常量(D-13 锁定,禁裸字符串)+ ":" + 64 位 sha256 hex;
// 同 params 同 method 幂等;不同 method 同 params 键不同。
func TestMcd7901_BuildKey_AllPrefixes(t *testing.T) {
	cases := []struct {
		method string
		prefix string
	}{
		{"port-history", cacheKeyPrefixPortHistory},
		{"device-history", cacheKeyPrefixDeviceHistory},
		{"stats", cacheKeyPrefixStats},
		{"heatmap", cacheKeyPrefixHeatmap},
	}
	params := map[string]any{"mac": "9C:7B:EF:2F:31:B8", "limit": 100}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			key, err := BuildMACQueryCacheKey(tc.method, params)
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(key, tc.prefix+":"),
				"键前缀必须是常量 %s + \":\", got %s", tc.prefix, key)

			sum := strings.TrimPrefix(key, tc.prefix+":")
			require.Len(t, sum, 64, "参数摘要应为 64 位 sha256 hex")
			for _, r := range sum {
				require.True(t, (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f'),
					"摘要必须是小写 hex, got %q", sum)
			}

			// 幂等:同 method 同 params 两次调用同键(Go encoding/json 字段序稳定)
			key2, err := BuildMACQueryCacheKey(tc.method, params)
			require.NoError(t, err)
			assert.Equal(t, key, key2, "同 method 同 params 必须产出同一缓存键")
		})
	}

	// 不同 method 同 params → 前缀不同 → 键不同
	k1, err := BuildMACQueryCacheKey("port-history", params)
	require.NoError(t, err)
	k2, err := BuildMACQueryCacheKey("stats", params)
	require.NoError(t, err)
	assert.NotEqual(t, k1, k2, "不同 method 的键不得碰撞")
}

// TestMcd7901_BuildKey_UnknownMethod method 不在 4 前缀清单 → "未知方法: %s"。
func TestMcd7901_BuildKey_UnknownMethod(t *testing.T) {
	key, err := BuildMACQueryCacheKey("bogus", map[string]int{"x": 1})
	require.Error(t, err)
	assert.Empty(t, key)
	assert.Contains(t, err.Error(), "未知方法")
	assert.Contains(t, err.Error(), "bogus")
}

// TestMcd7901_BuildKey_MarshalFail params 不可 JSON 序列化 → "序列化缓存参数失败: %w"。
func TestMcd7901_BuildKey_MarshalFail(t *testing.T) {
	key, err := BuildMACQueryCacheKey("stats", make(chan int))
	require.Error(t, err)
	assert.Empty(t, key)
	assert.Contains(t, err.Error(), "序列化缓存参数失败")
}

// ==================== Task 4: rate_limiter 尾支 ====================

// TestRlm7901_CalculateRemaining_Edges 表驱动 :229 calculateRemaining 的
// 零计数/满限/超限/三窗口最小值分支(期望值按 min(rM,rH,rD) 手算,见各 why 注释)。
func TestRlm7901_CalculateRemaining_Edges(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())
	limit := RateLimit{PerMinute: 30, PerHour: 500, PerDay: 5000}

	cases := []struct {
		name string
		m    int
		h    int
		d    int
		want int
		why  string
	}{
		{"零计数_分钟分支", 0, 0, 0, 30, "min(30-0, 500-0, 5000-0)=30"},
		{"分钟满限", 30, 100, 1000, 0, "min(0, 400, 4000)=0"},
		{"分钟超限为负", 35, 100, 1000, -5, "min(-5, 400, 4000)=-5(负值合法,Check 在前已拒)"},
		{"小时最小", 5, 495, 1000, 5, "min(25, 5, 4000)=5 → 走小时分支"},
		{"天最小", 5, 100, 4998, 2, "min(25, 400, 2)=2 → 走天分支"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := limiter.calculateRemaining(limit, tc.m, tc.h, tc.d)
			assert.Equal(t, tc.want, got, tc.why)
		})
	}
}

// TestRlm7901_CalculateReset_AndWindow :251 calculateReset 空 times 切片分支 +
// :192 getOrCreateWindow 复用既有窗口分支(往返回窗口写入时间戳后再次取窗仍可见)。
func TestRlm7901_CalculateReset_AndWindow(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())

	// 空 times 切片(:252-254)→ 返回当前时间,不 panic
	reset := limiter.calculateReset([]time.Time{}, time.Minute)
	assert.WithinDuration(t, time.Now(), reset, 2*time.Second,
		"空窗口应返回 now 作为兜底 ResetAt")

	// getOrCreateWindow 复用分支(:194-196 sync.Map Load 命中)
	w1 := limiter.getOrCreateWindow("rlm7901-reuse")
	w1.mu.Lock()
	w1.minute = append(w1.minute, now79())
	w1.mu.Unlock()

	w2 := limiter.getOrCreateWindow("rlm7901-reuse")
	assert.Same(t, w1, w2, "同 key 二次调用必须复用同一 *rateLimitWindow")
	w2.mu.Lock()
	count := len(w2.minute)
	w2.mu.Unlock()
	assert.Equal(t, 1, count, "往返回窗口写入的时间戳在再次取窗后仍可见(复用证据)")
}

// TestRlm7901_CleanOlderThan_Cutoff 构造 mixed 新旧时间戳切片,断言 cutoff
// 边界两侧的保留/剔除(含「等于 cutoff 不算早于」的二分边界语义)。
func TestRlm7901_CleanOlderThan_Cutoff(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())
	base := now79()

	times := []time.Time{
		base.Add(-3 * time.Hour),
		base.Add(-2 * time.Hour),
		base.Add(-30 * time.Second),
		base.Add(-10 * time.Second),
	}

	kept := limiter.cleanOlderThan(times, base.Add(-time.Minute))
	assert.Equal(t, times[2:], kept, "早于 cutoff 的 2 条应被剔除,其余按序保留")

	// 边界相等:Before(cutoff)==false → 保留(二分 left 停在该元素)
	atBoundary := limiter.cleanOlderThan([]time.Time{base}, base)
	assert.Equal(t, []time.Time{base}, atBoundary, "等于 cutoff 的时间戳不算「早于」,应保留")

	// 全部早于 cutoff → 空切片
	assert.Empty(t, limiter.cleanOlderThan([]time.Time{base.Add(-time.Hour)}, base))

	// 空入参 → 空出参,不 panic
	assert.Empty(t, limiter.cleanOlderThan(nil, base))
}

// TestRlm7901_Check_HourDayLimitDenials 驱动 Check 的小时/天级超限分支(:151-171,
// 既有测试只能靠 1500/50000 次请求触达,改用小阈值 provider 直达):
// 第 3 次请求分别被 hour/day 窗口拒绝,ResetAt 按 WR-02 口径 = 最早时间戳 + 对应窗口时长。
func TestRlm7901_Check_HourDayLimitDenials(t *testing.T) {
	t.Run("小时窗口拒绝", func(t *testing.T) {
		provider := &mockRateLimitProvider{limits: map[string]RateLimit{
			APIKeyScopeRead: {PerMinute: 1000, PerHour: 2, PerDay: 1000},
		}}
		limiter := NewRateLimiter(provider)
		start := now79()

		for i := 0; i < 2; i++ {
			allowed, res := limiter.Check("rlm7901-hour", APIKeyScopeRead)
			require.True(t, allowed, "第 %d 次请求应放行", i+1)
			require.NotNil(t, res)
		}
		allowed, res := limiter.Check("rlm7901-hour", APIKeyScopeRead)
		assert.False(t, allowed, "小时窗口满限后必须拒绝")
		require.NotNil(t, res)
		assert.Equal(t, 2, res.Limit, "Limit 应取小时档配置")
		assert.Equal(t, 0, res.Remaining, "拒绝路径 Remaining 必须为 0")
		assert.WithinDuration(t, start.Add(time.Hour), res.ResetAt, 2*time.Second,
			"ResetAt = 最早时间戳 + 1 小时(WR-02 回归口径)")
	})

	t.Run("天窗口拒绝", func(t *testing.T) {
		provider := &mockRateLimitProvider{limits: map[string]RateLimit{
			APIKeyScopeRead: {PerMinute: 1000, PerHour: 1000, PerDay: 2},
		}}
		limiter := NewRateLimiter(provider)
		start := now79()

		for i := 0; i < 2; i++ {
			allowed, res := limiter.Check("rlm7901-day", APIKeyScopeRead)
			require.True(t, allowed, "第 %d 次请求应放行", i+1)
			require.NotNil(t, res)
		}
		allowed, res := limiter.Check("rlm7901-day", APIKeyScopeRead)
		assert.False(t, allowed, "天窗口满限后必须拒绝")
		require.NotNil(t, res)
		assert.Equal(t, 2, res.Limit, "Limit 应取天档配置")
		assert.Equal(t, 0, res.Remaining, "拒绝路径 Remaining 必须为 0")
		assert.WithinDuration(t, start.Add(24*time.Hour), res.ResetAt, 2*time.Second,
			"ResetAt = 最早时间戳 + 24 小时(WR-02 回归口径)")
	})
}

// TestRlm7901_StaticProvider_Tails 兜底 provider 的防御/回退尾支:
// getLimit 的 config==nil 分支(:69-71,生产装配不可达,零值白盒直驱)+
// staticRateLimitProvider.GetRateLimit 的段数异常(:100-102)与未知 scope/粒度(:113)。
func TestRlm7901_StaticProvider_Tails(t *testing.T) {
	// config == nil 防御分支:零值 RateLimiter 白盒直调
	var zeroLimiter RateLimiter
	assert.Equal(t, RateLimit{PerMinute: 120, PerHour: 2000, PerDay: 20000},
		zeroLimiter.getLimit(APIKeyScopeRead),
		"config==nil 应返回 120/2000/20000 兜底档")

	// staticRateLimitProvider 尾支
	p := newStaticRateLimitProvider()
	assert.Equal(t, 42, p.GetRateLimit("badkey", 42),
		"key 段数 != 3 应返回 defaultValue")
	assert.Equal(t, 42, p.GetRateLimit("rate_limit.read.per_week", 42),
		"已知 scope + 未知粒度应返回 defaultValue")
	assert.Equal(t, 42, p.GetRateLimit("rate_limit.noscope.per_minute", 42),
		"未知 scope 应返回 defaultValue")
}

// ==================== Task 4: mac_normalize 收口 ====================

// TestMnm7901_IsCanonicalMAC_LastBranch isCanonicalMAC 全分支表驱动
// (空串守卫 + canonical 正则两态),并回归 NormalizeMACAddress 全链口径
// (引用既有 mac_normalize_test.go 的期望值,只增不改)。
func TestMnm7901_IsCanonicalMAC_LastBranch(t *testing.T) {
	cases := []struct {
		name string
		mac  string
		want bool
	}{
		{"空串守卫", "", false},
		{"标准大写冒号", "9C:7B:EF:2F:31:B8", true},
		{"小写不合规", "9c:7b:ef:2f:31:b8", false},
		{"无分隔符不合规", "9C7BEF2F31B8", false},
		{"非 hex 字符不合规", "ZZ:7B:EF:2F:31:B8", false},
		{"段数不足", "9C:7B:EF:2F:31", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isCanonicalMAC(tc.mac))
		})
	}

	// 全链不回归:归一 → canonical 校验 落地通路
	normalized := NormalizeMACAddress("9c7b.ef2f.31b8")
	assert.Equal(t, "9C:7B:EF:2F:31:B8", normalized)
	assert.True(t, isCanonicalMAC(normalized), "归一化产物必须通过 canonical 校验")
	assert.False(t, isCanonicalMAC(NormalizeMACAddress("not-a-mac")),
		"非法输入归一为空串后 canonical 校验必须为 false")
}
