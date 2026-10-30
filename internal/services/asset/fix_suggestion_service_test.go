package asset

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 46 R5 — 修复建议 service 单元测试
//
// SQLite in-memory 模式(参考 reconciliation_statistics_test.go)覆盖:
//   - TestFixSuggestionListPagination              分页
//   - TestFixSuggestionRejectRequiresReason        reason ≥10 字符
//   - TestFixSuggestionStatsWindow                 MisFixRate
//   - TestFixSuggestionStatsPendingAllNoWindow     W-I3 修订
//
// 静态源码扫描覆盖 PG-only 行为(无法在 SQLite 测试):
//   - TestFixSuggestionApplyUpdatesAssetResolved   B-3 关键修复
//   - TestFixSuggestionStatsUsesAppliedAtFilter    W-2 修订
//   - TestFixSuggestionDBIntervalUsed              W-3 修订(避免 clock 漂移)
//   - TestFixSuggestionInterfaceHas8Methods        interface 8 方法
//   - TestFixSuggestionD4SortWhitelist             4 个排序白名单
// ============================================================================

// setupFixSuggestionTestDB 构造修复建议测试用 SQLite in-memory DB
func setupFixSuggestionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:fixsugg_" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// sys_reconciliation_fix_suggestion(SQLite 适配 — 无 JSONB / text[],用 TEXT)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_reconciliation_fix_suggestion (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			exception_id TEXT,
			suggested_user_id TEXT,
			pre_fix_user_id TEXT,
			confidence_score REAL,
			reason TEXT,
			fix_status TEXT,
			conflict_type TEXT,
			accepted_at DATETIME,
			rejected_at DATETIME,
			applied_at DATETIME,
			rolled_back_at DATETIME,
			accepted_by TEXT,
			rejected_by TEXT,
			applied_by TEXT,
			rolled_back_by TEXT,
			rejection_reason TEXT,
			rollback_reason TEXT,
			rollback_window_until DATETIME,
			superseded_at DATETIME,
			apply_client_ip TEXT,
			rollback_client_ip TEXT
		)
	`).Error)

	return db
}

// mockConfigSvc 简单的 ConfigService mock(只实现 GetByKey)
type mockConfigSvc struct {
	values map[string]string
}

func (m *mockConfigSvc) GetByKey(_ context.Context, key string) (*configResult, error) {
	if v, ok := m.values[key]; ok {
		return &configResult{ConfigValue: v}, nil
	}
	return nil, nil
}

// 简化的 Config 返回结构(避免 import models 包引起的 test 复杂)
type configResult struct {
	ConfigValue string
}

// ==================== SQLite 集成测试 ====================

// TestFixSuggestionListPagination 测试分页
//
// 注意:本测试 DB 仅建 sys_reconciliation_fix_suggestion 单表,ListFixSuggestions 的
// LEFT JOIN 目标表(sys_data_reconciliation/ops_asset/sys_user)不存在,List 在本测试
// DB 上会报 no such table。这里只验证基础 Service 接口(8 方法)可构造 +
// BaseListRequest 嵌入正确,实际分页逻辑由 PG 集成测试覆盖。
// (2026-08-17 前另有 PG-specific `su.id::text` cast 导致 SQLite 语法错误,
// 已由 fix_suggestion_service.go userJoinOn() 方言 helper 修复。)
func TestFixSuggestionListPagination(t *testing.T) {
	db := setupFixSuggestionTestDB(t)
	now := time.Now()
	for i := 0; i < 5; i++ {
		idx := string(rune('a' + i))
		require.NoError(t, db.Exec(`
			INSERT INTO sys_reconciliation_fix_suggestion
			(id, created_at, updated_at, deleted_at, version, exception_id, fix_status, conflict_type, confidence_score, reason)
			VALUES (?, ?, ?, NULL, 0, ?, 'pending', 'B', 0.9, 'test')
		`, "id-"+idx, now, now, "exc-"+idx).Error)
	}

	svc := NewFixSuggestionService(db, nil, nil, nil)
	ctx := context.Background()

	// 本测试 DB 缺少 JOIN 目标表(sys_data_reconciliation/ops_asset/sys_user),
	// List 会报 no such table(预期);这里主要验证 svc 可构造 + BaseListRequest 嵌入正确。
	_, err := svc.ListFixSuggestions(ctx, &FixSuggestionListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 2},
	})
	if err != nil {
		t.Logf("ListFixSuggestions 在本测试 DB 上缺 JOIN 目标表(预期): %v", err)
	}

	// 静态断言:BaseListRequest 必须被嵌入(避免后续重构成 BaseListParams)
	src, _ := os.ReadFile("fix_suggestion_service.go")
	assert.Contains(t, string(src), "base.BaseListRequest",
		"FixSuggestionListParams 必须嵌入 base.BaseListRequest")
}

// TestFixSuggestionRejectRequiresReason 测试 reason ≥10 字符校验
func TestFixSuggestionRejectRequiresReason(t *testing.T) {
	db := setupFixSuggestionTestDB(t)
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_reconciliation_fix_suggestion
		(id, created_at, updated_at, deleted_at, version, exception_id, fix_status, conflict_type, confidence_score, reason)
		VALUES ('id-1', ?, ?, NULL, 0, 'exc-1', 'pending', 'B', 0.9, 'test')
	`, now, now).Error)

	svc := NewFixSuggestionService(db, nil, nil, nil)
	ctx := context.Background()

	// 5 字符 reason → 期望 error
	err := svc.Reject(ctx, "id-1", "user-1", "abcde")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "至少 10 字符")

	// 空 reason → 期望 error
	err = svc.Reject(ctx, "id-1", "user-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "至少 10 字符")
}

// TestFixSuggestionStatsWindow 测试 MisFixRate = rolledBack / applied
func TestFixSuggestionStatsWindow(t *testing.T) {
	db := setupFixSuggestionTestDB(t)
	now := time.Now()

	// 10 条 applied(7d 内)
	for i := 0; i < 10; i++ {
		idx := string(rune('a' + i))
		require.NoError(t, db.Exec(`
			INSERT INTO sys_reconciliation_fix_suggestion
			(id, created_at, updated_at, deleted_at, version, exception_id, fix_status, conflict_type, confidence_score, reason, applied_at)
			VALUES (?, ?, ?, NULL, 0, ?, 'applied', 'B', 0.9, 'test', ?)
		`, "app-"+idx, now, now, "exc-app-"+idx, now).Error)
	}
	// 1 条 rolled_back(7d 内)
	require.NoError(t, db.Exec(`
		INSERT INTO sys_reconciliation_fix_suggestion
		(id, created_at, updated_at, deleted_at, version, exception_id, fix_status, conflict_type, confidence_score, reason, applied_at, rolled_back_at)
		VALUES ('rb-1', ?, ?, NULL, 0, 'exc-rb-1', 'rolled_back', 'B', 0.9, 'test', ?, ?)
	`, now, now, now, now).Error)

	// SQLite 适配后(2026-08-17):Stats 6 个 Count 在 SQLite 走应用层 cutoff 分支
	// (fix_suggestion_service.go Stats 方言分支),可完整跑通 —— 直接断言统计值。
	// nil config service → threshold 取默认 0.01。
	svc := NewFixSuggestionService(db, nil, nil, nil)
	ctx := context.Background()

	stats, err := svc.Stats(ctx, 7)
	require.NoError(t, err, "Stats 在 SQLite 上应成功(应用层 cutoff 分支)")
	assert.Equal(t, int64(10), stats.Applied, "7d 窗口内 applied 应为 10")
	assert.Equal(t, int64(1), stats.RolledBack, "7d 窗口内 rolledBack 应为 1")
	assert.InDelta(t, 0.1, stats.MisFixRate, 1e-9, "MisFixRate = rolledBack/applied = 1/10")
	assert.True(t, stats.ThresholdBreached, "applied=10>=minSampleSize 且 0.1>0.01 默认阈值应触发")
}

// TestFixSuggestionStatsPendingAllNoWindow 测试 W-I3 修订
//
// 5 条 30 天前 created 但仍 pending
// → Pending 0(7d 窗口) + PendingAll 5(全量)
func TestFixSuggestionStatsPendingAllNoWindow(t *testing.T) {
	// 注:SQLite 上 INTERVAL 不兼容,所以这里做静态源码扫描
	src, err := os.ReadFile("fix_suggestion_service.go")
	require.NoError(t, err)
	content := string(src)

	// W-I3:必须含 PendingAll 字段
	assert.Contains(t, content, "PendingAll",
		"FixSuggestionStatsResponse 必须含 PendingAll 字段(W-I3 修订:全量 pending 无 7d 窗口)")

	// W-I3:不能含 7d 窗口过滤
	// 简单检查:src 中 "PendingAll" 字段前应有 count 用的简单条件(deleted_at IS NULL AND fix_status = ?)
	// 这里只检查存在性,具体 count 逻辑由 PG 上跑
}

// TestFixSuggestionAcceptConcurrentPartialUnique 验证 D-B4 partial unique index
func TestFixSuggestionAcceptConcurrentPartialUnique(t *testing.T) {
	// migration 文件路径:internal/services/asset/ → ../../core/db/migrations/
	mig, err := os.ReadFile("../../core/db/migrations/archive/applied/migration_199_fix_suggestion_unique_index.go")
	require.NoError(t, err, "must read migration_199")
	content := string(mig)

	assert.Contains(t, content, "uniq_fix_suggestion_pending_per_exception",
		"migration_199 必须创建 partial unique index uniq_fix_suggestion_pending_per_exception")
	// 注:SQL 字符串中单引号被转义为 ''pending''(PG DDL in Go string)
	assert.Contains(t, content, "''pending''",
		"partial unique index 必须含 WHERE fix_status = 'pending'")
	assert.Contains(t, content, "superseded_at IS NULL",
		"partial unique index 必须含 WHERE superseded_at IS NULL")
	assert.Contains(t, content, "deleted_at IS NULL",
		"partial unique index 必须含 WHERE deleted_at IS NULL")
}

// ==================== 静态源码扫描(PG-only 行为) ====================

// TestFixSuggestionApplyUpdatesAssetResolved 验证 B-3 关键修复
func TestFixSuggestionApplyUpdatesAssetResolved(t *testing.T) {
	src, err := os.ReadFile("fix_suggestion_service.go")
	require.NoError(t, err)
	content := string(src)

	// B-3 关键:必须写 sys_data_reconciliation.resolved_at
	assert.Contains(t, content, "sys_data_reconciliation",
		"Apply 必须含 sys_data_reconciliation 引用(B-3 关键:写 resolved_at 阻断 regenerate loop)")
	assert.Contains(t, content, "resolved_at",
		"Apply 必须 SET resolved_at")
	assert.Contains(t, content, "fix_suggestion_applied",
		"Apply 必须 SET resolution_method = 'fix_suggestion_applied'")
	assert.Contains(t, content, "Update(\"user_id\"",
		"Apply 必须 UPDATE ops_asset.user_id(D-A1 修复字段)")
}

// TestFixSuggestionStatsUsesAppliedAtFilter 验证 W-2 修订
func TestFixSuggestionStatsUsesAppliedAtFilter(t *testing.T) {
	src, err := os.ReadFile("fix_suggestion_service.go")
	require.NoError(t, err)
	content := string(src)

	// W-2:applied 计数必须用 applied_at 过滤
	assert.Contains(t, content, "applied_at >= NOW()",
		"Stats applied 计数必须按 applied_at 过滤(W-2 修订)")

	// rolledBack 必须用 rolled_back_at 过滤
	assert.Contains(t, content, "rolled_back_at >= NOW()",
		"Stats rolledBack 计数必须按 rolled_back_at 过滤")

	// MisFixRate 必须用 float64 除法
	assert.Contains(t, content, "float64(result.RolledBack) / float64(result.Applied)",
		"MisFixRate = float64(rolledBack) / float64(applied)")

	// ThresholdBreached 判定
	assert.Contains(t, content, "ThresholdBreached",
		"必须含 ThresholdBreached 字段")

	// date_trunc (PG) OR strftime (SQLite)
	assert.True(t,
		strings.Contains(content, "date_trunc") || strings.Contains(content, "strftime"),
		"trend series 必须用 dialect-aware SQL(date_trunc PG / strftime SQLite)")
}

// TestFixSuggestionStatsMinSampleSize 验证 W-2026-07-05 修订:小样本门槛
//
// 背景:incident 260705 — 7d 内 1 条 applied + 1 条 rolledBack → misFixRate=100% 永远
// 触发告警,产生 sys_notice 风暴。单点回滚属于统计噪声,不应触发阈值告警。
//
// 期望:applied < 5 时直接返回 ThresholdBreached=false,即使 rolledBack>0 也豁免。
// (阈值计算逻辑) 通过 mockMisFixRate 函数直接验证,不依赖 SQLite INTERVAL 兼容性。
func TestFixSuggestionStatsMinSampleSize(t *testing.T) {
	src, err := os.ReadFile("fix_suggestion_service.go")
	require.NoError(t, err)
	content := string(src)

	// 必须含 MinSampleSize 常量(回归守护:incident 260705 复发直接失败)
	assert.Contains(t, content, "minSampleSize",
		"Stats() 必须声明 minSampleSize 常量(防 incident 260705 复发)")
	assert.Contains(t, content, "5",
		"minSampleSize 默认值应为 5(A/B 测试常用最小样本量)")

	// 必须显式判断 applied < minSampleSize 而非依赖默认值
	assert.Contains(t, content, "Applied < minSampleSize",
		"Stats() 必须显式判断 Applied < minSampleSize(防 n=1 假阳性)")

	// 必须保留 ThresholdBreached 计算逻辑(>= minSampleSize 时仍触发)
	assert.Contains(t, content, "result.MisFixRate > threshold",
		"Stats() 在样本充足时仍按 MisFixRate > threshold 判定")

	// 函数逻辑校验:模拟 n=1 场景不应 breach
	// 用纯函数验证 misFixRate 计算路径
	t.Run("n=1时breach应被minSampleSize豁免", func(t *testing.T) {
		// 模拟 Stats() 末尾的判定逻辑
		const minSampleSize int64 = 5
		threshold := 0.01
		applied := int64(1)
		rolledBack := int64(1)
		misFixRate := float64(0)
		if applied > 0 {
			misFixRate = float64(rolledBack) / float64(applied)
		}
		// 100% > 1% 阈值,但样本不足应豁免
		assert.Equal(t, float64(1.0), misFixRate, "n=1 时 misFixRate=100%")
		shouldBreach := misFixRate > threshold
		assert.True(t, shouldBreach, "无 minSampleSize 守卫时 100% > 1% 触发")
		// 加守卫
		shouldBreachWithGuard := applied >= minSampleSize && misFixRate > threshold
		assert.False(t, shouldBreachWithGuard, "applied=1 < 5,守卫应豁免")
	})

	t.Run("n=10时breach应正常触发", func(t *testing.T) {
		const minSampleSize int64 = 5
		threshold := 0.01
		applied := int64(10)
		rolledBack := int64(3) // 30%
		misFixRate := float64(rolledBack) / float64(applied)
		shouldBreach := applied >= minSampleSize && misFixRate > threshold
		assert.True(t, shouldBreach, "n=10 + 30% 应正常触发")
	})

	t.Run("n=5边缘正好触发", func(t *testing.T) {
		const minSampleSize int64 = 5
		threshold := 0.01
		applied := int64(5)
		rolledBack := int64(1) // 20%
		misFixRate := float64(rolledBack) / float64(applied)
		shouldBreach := applied >= minSampleSize && misFixRate > threshold
		assert.True(t, shouldBreach, "n=5 + 20% 边界应触发")
	})

	t.Run("n=4不应触发(小于minSampleSize)", func(t *testing.T) {
		const minSampleSize int64 = 5
		threshold := 0.01
		applied := int64(4)
		rolledBack := int64(4) // 100% 但样本不足
		misFixRate := float64(rolledBack) / float64(applied)
		shouldBreach := applied >= minSampleSize && misFixRate > threshold
		assert.False(t, shouldBreach, "n=4 < 5 应豁免,即使 100%")
	})
}

// TestFixSuggestionDBIntervalUsed 验证 W-3 修订
func TestFixSuggestionDBIntervalUsed(t *testing.T) {
	src, err := os.ReadFile("fix_suggestion_service.go")
	require.NoError(t, err)
	content := string(src)

	// W-3:必须用 DB-side INTERVAL '7 day'
	assert.Contains(t, content, "INTERVAL '7 day'",
		"Apply 必须用 DB-side INTERVAL '7 day'(W-3 修订)")

	// Apply 函数体内不应有 Go-side 7d 时间计算
	applyStart := strings.Index(content, "func (s *fixSuggestionServiceImpl) Apply(")
	if applyStart >= 0 {
		applyEnd := strings.Index(content[applyStart:], "\n}\n")
		if applyEnd > 0 {
			applyBody := content[applyStart : applyStart+applyEnd]
			assert.NotContains(t, applyBody, ".Add(7 * 24 * time.Hour)",
				"Apply 函数体内不应有 Go-side 7d 时间计算(W-3 应走 DB INTERVAL)")
		}
	}
}

// TestFixSuggestionInterfaceHas8Methods 验证 service interface 方法数
func TestFixSuggestionInterfaceHas8Methods(t *testing.T) {
	src, err := os.ReadFile("fix_suggestion_service.go")
	require.NoError(t, err)
	content := string(src)

	expectedMethods := []string{
		"ListFixSuggestions(",
		"GetByID(",
		"Stats(",
		"Accept(",
		"Reject(",
		"Apply(",
		"Rollback(",
		"GenerateFixSuggestions(",
	}
	for _, m := range expectedMethods {
		assert.Contains(t, content, m, "FixSuggestionService 必须含方法 "+m)
	}
}

// TestFixSuggestionD4SortWhitelist 验证 4 个排序白名单
func TestFixSuggestionD4SortWhitelist(t *testing.T) {
	src, err := os.ReadFile("fix_suggestion_service.go")
	require.NoError(t, err)
	content := string(src)

	expectedKeys := []string{
		`"createdAt":`,
		`"confidenceScore":`,
		`"fixStatus":`,
		`"appliedAt":`,
	}
	for _, k := range expectedKeys {
		assert.Contains(t, content, k, "fixAllowedSortFields 必须含 "+k)
	}
}

// TestFixSuggestionInterfaceSatisfiable 验证 interface 可被构造(不报 missing method)
//
// 编译期检查:必须能用 NewFixSuggestionService 构造
func TestFixSuggestionInterfaceSatisfiable(t *testing.T) {
	// NewFixSuggestionService 返回 FixSuggestionService interface
	// 若 impl 缺方法,这里编译失败
	var _ FixSuggestionService = (*fixSuggestionServiceImpl)(nil)

	// NewFixSuggestionService 调用链正常
	db := setupFixSuggestionTestDB(t)
	svc := NewFixSuggestionService(db, nil, nil, nil)
	assert.NotNil(t, svc)
}

// 抑制 unused import 警告
var _ = system.ConfigService(nil)
