package asset

// Phase 147 · CACHE 失效契约测试（CACHE-01 / CACHE-02 / CACHE-03）
//
// 本文件锁定以下契约：
//  1. 例外规则写成功后，GetByWorkstation 5min TTL 缓存被广播失效，写后读必须回源
//     重算（不得返回陈旧快照）；直查 DB 的 List/MatchTest/MatchException 立即一致。
//     反向红绿验证由 Task 5 T5-2 注释掉失效调用后本测试必须红。
//  2. 失效动作可观测（DeleteByPattern 调用次数 ≥1）、失败可传播（返回 error）、
//     失败不阻断主写路径（DB 已提交）、nil-safe。
//  3. memory 配置下 DeleteByPattern 真删（Keys → MDelete）。
//  4. 死键 / 空函数 / 错误处方全仓零命中（含 *_test.go 与注释）。
//  5. 广播 pattern 与读缓存键同源；现成范式仍存在；5min TTL 与读路径未改。
//
// 判据口径（重要，SUMMARY 同步登记）：
//   computeByWorkstation（reconciliation_service.go:1074）先 append(assetItems, item)
//   再在 :1085-1094 修改 item 的 ExceptionRuleID / AppliedActions，切片里存的是副本，
//   故 per-asset matcher 结果未进入响应——这是生产既有缺陷，本 phase 禁区 8 禁止改
//   6 步聚合。因此「写后立即一致」用更严格且不受该缺陷影响的等价判据：
//     (a) 写路径真删 reconciliation:health:workstation:* 缓存键；
//     (b) 写后读确实回源重算（computeByWorkstation / matcher 调用次数增加）；
//     (c) 直查 DB 的读路径（MatchException / List）同进程立即反映新规则。
//   缓存未失效时 (a)(b) 不成立，反向验证必红。
//
// 复用的同包基建：newGapTestDB / seedWorkstationHealth / buildGapXLSX /
// gapXlsxFileHeader（asset_gapfill_test.go、reconciliation_test.go）。
// 不新造 SQL、不启动长驻进程、不连生产 Redis、不落盘 .xlsx。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	services "github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	pkgcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// =====================================================================
// contractCacheSpy：实现 base.CacheProvider 全部 9 方法。
//
// GetOrSet 复刻 DataCacheService.GetOrSet 的 JSON 往返语义（命中反序列化到 dest，
// 未命中执行 query 并同步写缓存），因此 GetByWorkstation 的读穿透与生产一致；
// DeleteByPattern 真删匹配前缀的键，从而让「写后读」必然回源，也让反向验证
// （注释掉失效调用 → 缓存保留旧值 → 不回源 → 测试红）成立。
// =====================================================================

type contractCacheSpy struct {
	mu sync.Mutex

	data map[string]string // key → JSON 序列化值

	deleteByPatternCalls []string
	deleteCalls          []string
	mDeleteKeys          [][]string
	getOrSetCalls        int
	missCalls            int

	deleteByPatternErr error // 可插桩：模拟 DeleteByPattern 失败
}

var _ base.CacheProvider = (*contractCacheSpy)(nil)

func newContractCacheSpy() *contractCacheSpy {
	return &contractCacheSpy{data: make(map[string]string)}
}

func (s *contractCacheSpy) GetOrSet(
	ctx context.Context,
	key string,
	dest interface{},
	expiration time.Duration,
	query func() (interface{}, error),
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getOrSetCalls++
	if raw, ok := s.data[key]; ok {
		// 命中：反序列化到 dest（与 DataCacheService.Get 语义一致）
		return json.Unmarshal([]byte(raw), dest)
	}
	s.missCalls++
	data, err := query()
	if err != nil {
		return err
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// 未命中：先赋值给调用方变量（json.Unmarshal 支持多级指针），再同步写缓存
	if err := json.Unmarshal(b, dest); err != nil {
		return err
	}
	s.data[key] = string(b)
	return nil
}

func (s *contractCacheSpy) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCalls = append(s.deleteCalls, key)
	delete(s.data, key)
	return nil
}

func (s *contractCacheSpy) DeleteByPattern(ctx context.Context, pattern string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteByPatternCalls = append(s.deleteByPatternCalls, pattern)
	if s.deleteByPatternErr != nil {
		return s.deleteByPatternErr
	}
	// 真删：模拟 pkg/cache.Keys(pattern) → MDelete 的 glob 前缀匹配
	prefix := strings.TrimSuffix(pattern, "*")
	for k := range s.data {
		if strings.HasPrefix(k, prefix) {
			delete(s.data, k)
		}
	}
	return nil
}

func (s *contractCacheSpy) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		if v, ok := s.data[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (s *contractCacheSpy) MDelete(ctx context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mDeleteKeys = append(s.mDeleteKeys, append([]string(nil), keys...))
	for _, k := range keys {
		delete(s.data, k)
	}
	return nil
}

func (s *contractCacheSpy) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[key]
	return ok, nil
}

func (s *contractCacheSpy) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

func (s *contractCacheSpy) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (s *contractCacheSpy) GetStats(ctx context.Context) (*base.CacheStats, error) {
	return &base.CacheStats{}, nil
}

// snapshot 快照（加锁）：失效调用 pattern 列表、健康度缓存键是否仍存在、回源次数。
func (s *contractCacheSpy) snapshot() (patterns []string, hasCacheKey bool, missCalls int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	patterns = append([]string(nil), s.deleteByPatternCalls...)
	_, hasCacheKey = s.data[GetReconciliationHealthByWorkstationKey("ws-1")]
	missCalls = s.missCalls
	return
}

// =====================================================================
// 计数 matcher：证明「写后读是否回源重算」（computeByWorkstation 每资产调一次）。
// =====================================================================

type countingExceptionSvc struct {
	ReconciliationExceptionService
	matchCalls int
}

func (c *countingExceptionSvc) MatchException(ctx context.Context, ip string, userID string, conflictType string) (*ExceptionMatch, error) {
	c.matchCalls++
	return c.ReconciliationExceptionService.MatchException(ctx, ip, userID, conflictType)
}

// =====================================================================
// fixture
// =====================================================================

// contractFixture 组装：sqlite 内存 DB + 工位/资产/异常 fixture + spy +
// 注入同一个 spy 的例外规则服务与聚合读服务（缓存与失效共后端，等价于
// 生产 core.DataCacheService 同时供 NewReconciliationService 与
// system.NewCacheProvider 使用，见 PLAN §3.5）。
type contractFixture struct {
	db           *gorm.DB
	spy          *contractCacheSpy
	exceptionSvc ReconciliationExceptionService
	counter      *countingExceptionSvc
	reconSvc     ReconciliationService
	ctx          context.Context
	cacheKey     string
}

// seedOldHitRule 预置一条命中 a-1（10.1.0.5）的 active 规则，actions=no_alert。
func seedOldHitRule(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_reconciliation_exception
			(id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at)
		VALUES ('rule-old', '旧命中规则', '10.1.0.0/24', '{no_alert}', 'global', '用于契约测试的旧规则原因', 0, 1, NULL)
	`).Error)
}

func newContractFixture(t *testing.T, seedHit bool) *contractFixture {
	t.Helper()
	db := newGapTestDB(t)
	seedWorkstationHealth(t, db)
	if seedHit {
		seedOldHitRule(t, db)
	}
	spy := newContractCacheSpy()
	exceptionSvc := NewReconciliationExceptionService(db, spy)
	counter := &countingExceptionSvc{ReconciliationExceptionService: exceptionSvc}
	reconSvc := NewReconciliationService(db, spy, counter)
	return &contractFixture{
		db:           db,
		spy:          spy,
		exceptionSvc: exceptionSvc,
		counter:      counter,
		reconSvc:     reconSvc,
		ctx:          context.Background(),
		cacheKey:     GetReconciliationHealthByWorkstationKey("ws-1"),
	}
}

// =====================================================================
// T4-1 写后读必一致（4 个写路径）
// =====================================================================

func TestExceptionRuleWrite_ReadYourWrites(t *testing.T) {
	const createReason = "用于契约测试的创建原因数据" // ≥10 字符

	cases := []struct {
		name      string
		seedHit   bool
		write     func(t *testing.T, f *contractFixture)
		wantMatch bool // 写后直查 DB 读路径（MatchException）是否应命中 a-1
	}{
		{
			name:    "Create",
			seedHit: false,
			write: func(t *testing.T, f *contractFixture) {
				_, err := f.exceptionSvc.Create(f.ctx, &CreateExceptionRuleRequest{
					Name:             "契约创建规则",
					IPRange:          "10.1.0.0/24",
					ExceptionActions: pq.StringArray{"no_alert"},
					ScopeType:        "global",
					Reason:           createReason,
				})
				require.NoError(t, err)
			},
			wantMatch: true,
		},
		{
			name:    "Update",
			seedHit: true,
			write: func(t *testing.T, f *contractFixture) {
				require.NoError(t, f.exceptionSvc.Update(f.ctx, "rule-old", &UpdateExceptionRuleRequest{
					Name:             "契约更新规则",
					IPRange:          "10.9.0.0/24", // 改到 a-1 之外的网段 → 不再命中
					ExceptionActions: pq.StringArray{"no_alert"},
					ScopeType:        "global",
					Reason:           "用于契约测试的更新原因数据",
				}))
			},
			wantMatch: false,
		},
		{
			name:    "Delete",
			seedHit: true,
			write: func(t *testing.T, f *contractFixture) {
				require.NoError(t, f.exceptionSvc.Delete(f.ctx, "rule-old"))
			},
			wantMatch: false,
		},
		{
			name:    "ImportFromExcel",
			seedHit: false,
			write: func(t *testing.T, f *contractFixture) {
				rows := [][]string{
					{"规则名称", "IP段(CIDR)", "冲突类型(逗号分隔B/C/D/E/F,空=全部)",
						"动作(逗号分隔no_alert/no_notice/no_workorder/skip_severity/silence)",
						"严重度覆盖(low/medium/high,可空)", "范围(global/dept/user)",
						"范围名称(部门名/用户名,global留空)", "过期时间(可空,YYYY-MM-DD)", "原因(≥10字符)"},
					{"契约导入规则", "10.1.0.0/24", "B", "no_alert", "", "global", "", "", "用于契约测试的导入原因数据"},
				}
				fh := gapXlsxFileHeader(t, buildGapXLSX(t, "对账例外规则", rows), "contract-rules.xlsx")
				result, err := f.exceptionSvc.ImportFromExcel(f.ctx, fh)
				require.NoError(t, err)
				require.Equal(t, 1, result.Inserted, "导入规则应真实入表")
			},
			wantMatch: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newContractFixture(t, tc.seedHit)

			// 1) 预热 5min TTL 缓存：必须回源（miss），缓存键写入 spy
			resp0, err := f.reconSvc.GetByWorkstation(f.ctx, "ws-1", "7d")
			require.NoError(t, err)
			require.NotNil(t, resp0)
			_, hasKey, missesAfterWarm := f.spy.snapshot()
			require.True(t, hasKey, "预热应把健康度缓存键写入 spy")
			require.Equal(t, 1, missesAfterWarm, "预热应回源一次（miss）")
			callsAfterWarm := f.counter.matchCalls
			require.GreaterOrEqual(t, callsAfterWarm, 2, "预热回源应执行 per-asset 匹配")

			// 2) 写路径（DB 成功后广播失效）
			tc.write(t, f)

			// 2.5) 写后、读前：失效动作真发生 → 缓存键已被真删（下次读必然回源）
			patterns, hasKeyAfterWrite, _ := f.spy.snapshot()
			require.GreaterOrEqual(t, len(patterns), 1, "写路径必须触发 DeleteByPattern")
			wantPattern := fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")
			assert.Equal(t, wantPattern, patterns[len(patterns)-1])
			assert.False(t, hasKeyAfterWrite,
				"写成功后缓存键必须真删；否则 GetByWorkstation 会返回陈旧 exception 字段（CACHE-02）")

			// 3) 立刻再读：必须回源重算（不等 5min / cron @every 6m）
			resp1, err := f.reconSvc.GetByWorkstation(f.ctx, "ws-1", "7d")
			require.NoError(t, err)
			require.NotNil(t, resp1)
			_, _, missesAfterRead := f.spy.snapshot()
			assert.GreaterOrEqual(t, missesAfterRead, 2,
				"写后读必须回源重算（缓存已失效），不得命中陈旧缓存")
			assert.GreaterOrEqual(t, f.counter.matchCalls, callsAfterWarm+2,
				"写后读必须重新执行 computeByWorkstation 的 per-asset 匹配（回源证据）")

			// 4) 直查 DB 的读路径同进程立即反映新规则（CACHE-03 回归守护）
			m, err := f.exceptionSvc.MatchException(f.ctx, "10.1.0.5", "", "B")
			require.NoError(t, err)
			if tc.wantMatch {
				require.NotNil(t, m, "写后 MatchException 应立即命中新规则")
				assert.NotEmpty(t, m.MatchedRuleID)
			} else {
				assert.Nil(t, m, "写后 MatchException 应立即不再命中旧规则")
			}
			rules, err := f.exceptionSvc.List(f.ctx, &ExceptionRuleListParams{})
			require.NoError(t, err)
			assert.NotNil(t, rules)
			_, err = f.exceptionSvc.MatchTest(f.ctx, "10.1.0.5", "", "")
			require.NoError(t, err)
		})
	}
}

// =====================================================================
// T3-4 / T4-3 失效失败不阻断主写但可观测、失败可传播、nil-safe
// =====================================================================

func TestInvalidate_ReturnsError_OnDeleteByPatternFailure(t *testing.T) {
	db := newGapTestDB(t)
	spy := newContractCacheSpy()
	spy.deleteByPatternErr = errors.New("boom")
	svc := NewReconciliationExceptionService(db, spy)
	impl := svc.(*reconciliationExceptionServiceImpl)

	err := impl.invalidateExceptionRuleCache(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	assert.Contains(t, err.Error(), fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*"))
	assert.True(t, errors.Is(err, spy.deleteByPatternErr), "底层 error 必须用 %w 包装传播")
}

func TestWritePath_InvalidationFailure_DoesNotBlockButObservable(t *testing.T) {
	f := newContractFixture(t, false)
	f.spy.deleteByPatternErr = errors.New("boom")

	// 失效失败不阻断主写路径：Create 仍成功、规则仍落库
	_, err := f.exceptionSvc.Create(f.ctx, &CreateExceptionRuleRequest{
		Name:             "失效失败仍应落库",
		IPRange:          "10.1.0.0/24",
		ExceptionActions: pq.StringArray{"no_alert"},
		ScopeType:        "global",
		Reason:           "用于契约测试的失效失败原因数据",
	})
	require.NoError(t, err)

	var cnt int64
	require.NoError(t, f.db.Table("sys_reconciliation_exception").
		Where("name = ?", "失效失败仍应落库").Count(&cnt).Error)
	assert.Equal(t, int64(1), cnt, "主写路径（DB）必须已提交")

	// 可观测：DeleteByPattern 仍被调用 ≥1（调用方 Warnf 观测该失败）
	patterns, _, _ := f.spy.snapshot()
	assert.GreaterOrEqual(t, len(patterns), 1, "失效失败必须可观测（DeleteByPattern 被调用）")
}

func TestInvalidate_NilCache_IsNoOp(t *testing.T) {
	db := newGapTestDB(t)
	svc := NewReconciliationExceptionService(db) // 零参 → cache=nil
	impl := svc.(*reconciliationExceptionServiceImpl)
	require.Nil(t, impl.cache)
	require.NoError(t, impl.invalidateExceptionRuleCache(context.Background()))
}

func TestInvalidate_DeleteByPattern_CalledAtLeastOnce(t *testing.T) {
	// 4 个写路径各自断言 DeleteByPattern 调用次数 ≥1（与 ReadYourWrites 互补，
	// 这里只锁定「每个写路径都接了失效」，不依赖缓存预热）。
	writers := []struct {
		name    string
		seedHit bool
		write   func(t *testing.T, f *contractFixture)
	}{
		{
			name: "Create",
			write: func(t *testing.T, f *contractFixture) {
				_, err := f.exceptionSvc.Create(f.ctx, &CreateExceptionRuleRequest{
					Name:             "计数创建规则",
					IPRange:          "10.1.0.0/24",
					ExceptionActions: pq.StringArray{"no_alert"},
					ScopeType:        "global",
					Reason:           "用于契约测试的计数创建原因数据",
				})
				require.NoError(t, err)
			},
		},
		{
			name:    "Update",
			seedHit: true,
			write: func(t *testing.T, f *contractFixture) {
				require.NoError(t, f.exceptionSvc.Update(f.ctx, "rule-old", &UpdateExceptionRuleRequest{
					Name:             "计数更新规则",
					IPRange:          "10.9.0.0/24",
					ExceptionActions: pq.StringArray{"no_alert"},
					ScopeType:        "global",
					Reason:           "用于契约测试的计数更新原因数据",
				}))
			},
		},
		{
			name:    "Delete",
			seedHit: true,
			write: func(t *testing.T, f *contractFixture) {
				require.NoError(t, f.exceptionSvc.Delete(f.ctx, "rule-old"))
			},
		},
		{
			name: "ImportFromExcel",
			write: func(t *testing.T, f *contractFixture) {
				rows := [][]string{
					{"规则名称", "IP段(CIDR)", "冲突类型(逗号分隔B/C/D/E/F,空=全部)",
						"动作(逗号分隔no_alert/no_notice/no_workorder/skip_severity/silence)",
						"严重度覆盖(low/medium/high,可空)", "范围(global/dept/user)",
						"范围名称(部门名/用户名,global留空)", "过期时间(可空,YYYY-MM-DD)", "原因(≥10字符)"},
					{"计数导入规则", "10.1.0.0/24", "B", "no_alert", "", "global", "", "", "用于契约测试的计数导入原因数据"},
				}
				fh := gapXlsxFileHeader(t, buildGapXLSX(t, "对账例外规则", rows), "count-rules.xlsx")
				_, err := f.exceptionSvc.ImportFromExcel(f.ctx, fh)
				require.NoError(t, err)
			},
		},
	}
	for _, w := range writers {
		t.Run(w.name, func(t *testing.T) {
			f := newContractFixture(t, w.seedHit)
			w.write(t, f)
			patterns, _, _ := f.spy.snapshot()
			assert.GreaterOrEqual(t, len(patterns), 1,
				"%s 写路径必须调用 DeleteByPattern 至少一次", w.name)
		})
	}
}

// =====================================================================
// T4-2 memory 配置真删（不伪造 Redis 双跑）
// =====================================================================

func TestInvalidate_MemoryCache_ReallyDeletes(t *testing.T) {
	ctx := context.Background()
	mem := pkgcache.NewMemoryCache(100, time.Minute)
	dcs := services.NewDataCacheService(mem)
	cp := system.NewCacheProvider(dcs) // 生产同款装配：cacheProviderAdapter

	key := GetReconciliationHealthByWorkstationKey("ws-mem")
	var out map[string]string
	require.NoError(t, cp.GetOrSet(ctx, key, &out, time.Minute, func() (interface{}, error) {
		return map[string]string{"v": "1"}, nil
	}))
	_, err := mem.Get(ctx, key)
	require.NoError(t, err, "写入后键应存在")

	pattern := fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")
	require.NoError(t, cp.DeleteByPattern(ctx, pattern))
	_, err = mem.Get(ctx, key)
	require.Error(t, err, "memory 配置下 DeleteByPattern 必须真删该键（Keys→MDelete）")
}

func TestInvalidate_MemoryAndRedis_BehaveIdentically(t *testing.T) {
	// Redis 侧需真 *MultiLevelCache + 真 Redis 服务；本 phase 不启动长驻进程、
	// 不连生产 Redis（PLAN T4-2 / 禁区 6），故显式 Skip 并登记 Known Gap。
	t.Skip("Redis 真机双跑需 *MultiLevelCache + 真 Redis：本 phase 不启动长驻进程/不连生产，交集成环境验证；" +
		"两条生产 CacheProvider 路径（cacheProviderAdapter / NoOpCacheProvider）共用 DataCacheService.DeleteByPattern，静态见 TestCacheProviderImplementations_ShareDeleteByPattern")
}

func TestCacheProviderImplementations_ShareDeleteByPattern(t *testing.T) {
	// 静态断言（零运行时依赖）：两种生产实现都实现 DeleteByPattern，
	// 且 DataCacheService.DeleteByPattern 走 Keys→MDelete（memory/redis 真删同源）。
	baseSrc, err := os.ReadFile("../base/cache_provider.go")
	require.NoError(t, err)
	assert.Contains(t, string(baseSrc), "DeleteByPattern(ctx context.Context, pattern string) error",
		"base.CacheProvider 接口必须含 DeleteByPattern（本 phase 不改该接口）")
	require.Contains(t, string(baseSrc), "func (n *NoOpCacheProvider) DeleteByPattern")

	dcsSrc, err := os.ReadFile("../data_cache_service.go")
	require.NoError(t, err)
	assert.Contains(t, string(dcsSrc), "func (s *DataCacheService) DeleteByPattern")
	assert.Contains(t, string(dcsSrc), "s.cache.Keys(ctx, pattern)")
	assert.Contains(t, string(dcsSrc), "s.cache.MDelete(ctx, keys...)")
}

// =====================================================================
// T4-4 死键 / 空函数 / 错误处方静态断言（覆盖 *_test.go 与注释）
// =====================================================================

func TestDeadSymbols_AreGone(t *testing.T) {
	// 拼接构造被扫字符串，避免本测试文件自身含有这些字面量造成自命中。
	dead := []string{
		"invalidate" + "Cache",
		"CacheKeyReconciliationExceptionRule" + "List",
		"GetReconciliationExceptionRule" + "ListKey",
		"CacheKeyReconciliationExceptionRule" + "ByID",
		"GetReconciliationExceptionRule" + "ByIDKey",
		"接入 Redis " + "后补", "未来接入 Redis " + "再补",
		"core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRule" + "List)",
	}

	// 扫描范围：本包 internal/services/asset/*.go（含 *_test.go 与注释）
	// + internal/api/v1/asset/*.go（写路径装配点）。
	var files []string
	own, err := filepath.Glob("*.go")
	require.NoError(t, err)
	files = append(files, own...)
	apiFiles, err := filepath.Glob("../../api/v1/asset/*.go")
	require.NoError(t, err)
	files = append(files, apiFiles...)
	require.NotEmpty(t, files, "静态扫描必须覆盖到 .go 文件")

	// 必须清零清单（PLAN T1-5 / T4-4 / T5-3）：
	//   reconciliation_exception.go:381/:449/:474/:622（4 写路径）、:578 函数定义
	//   及 :574-580 定义段注释、:337 悬空注释；
	//   asset_gapfill_test.go:768 注释与 :771 测试调用、:117,118 死键断言。
	for _, path := range files {
		src, err := os.ReadFile(path)
		require.NoError(t, err, "读取 %s", path)
		text := string(src)
		for _, sym := range dead {
			assert.NotContains(t, text, sym,
				"%s 不应再出现已删除符号/错误处方: %s", path, sym)
		}
	}

	// 正向断言：现成范式与读缓存键必须保留（防误删）。
	keysSrc, err := os.ReadFile("cache_keys.go")
	require.NoError(t, err)
	assert.Contains(t, string(keysSrc), "func InvalidateWorkstationHealth(")
	assert.Contains(t, string(keysSrc), "func GetReconciliationHealthByWorkstationKey(")
	assert.Contains(t, string(keysSrc), "CacheKeyReconciliationHealthByWorkstation")
	baseFnSrc, err := os.ReadFile("../base/cache_functions.go")
	require.NoError(t, err)
	assert.Contains(t, string(baseFnSrc), "func InvalidatePattern(")
}

// =====================================================================
// T4-5 广播 pattern 与读缓存键同源（防止「删了一个平行宇宙的键」）
// =====================================================================

func TestBroadcastPattern_SharesPrefixWithReadKey(t *testing.T) {
	pattern := fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")
	prefix := strings.TrimSuffix(pattern, "*")
	readKey := GetReconciliationHealthByWorkstationKey("ws-1")
	assert.True(t, strings.HasPrefix(readKey, prefix),
		"广播 pattern 前缀 %q 必须覆盖读缓存键 %q", prefix, readKey)
	assert.Equal(t, "reconciliation:health:workstation:*", pattern)
}

// TestReadPath_TTL_Unchanged 守护「保留 5min TTL、读路径零改动」。
func TestReadPath_TTL_Unchanged(t *testing.T) {
	assert.Equal(t, 5*time.Minute, reconciliationHealthCacheTTL)
	svcSrc, err := os.ReadFile("reconciliation_service.go")
	require.NoError(t, err)
	assert.Contains(t, string(svcSrc), "func (s *reconciliationServiceImpl) getByWorkstationWithCache(")
	assert.Contains(t, string(svcSrc), "func (s *reconciliationServiceImpl) computeByWorkstation(")
	assert.Contains(t, string(svcSrc), "base.GetOrSetJSON[*ByWorkstationResponse]")
}
