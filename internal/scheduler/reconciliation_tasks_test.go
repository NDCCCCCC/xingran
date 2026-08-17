package scheduler

// Phase 44 R3 / Plan 44-02 Task 1 — cleanupExpiredExceptions 软停用 + 转单 SQL no_workorder 过滤
//
// 本测试集聚焦两个 cron 行为:
//  1. cleanupExpiredExceptions: 软停用过期规则 (UPDATE is_active=0→1) + 幂等 + 软删除/无 expires_at 跳过
//  2. createWorkorderBySeverity: 转单 SQL 加 no_workorder 过滤,含 applied_actions IS NULL 兜底
//     (BLOCKER-4: applied_actions=NULL 的 critical/high 异常必须被转单,防 PG 三值逻辑漏转)
//
// 测试用 SQLite in-memory + 直接调内部函数(非 cron 触发,避免 cron 引擎依赖)。

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupReconExceptionTestDB 构造内存 sqlite + sys_reconciliation_exception 表
func setupReconExceptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:recon_task_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_reconciliation_exception (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			name TEXT,
			ip_range TEXT,
			conflict_types TEXT,
			exception_actions TEXT,
			severity_override TEXT,
			scope_type TEXT,
			scope_id TEXT,
			reason TEXT,
			is_active INTEGER DEFAULT 0,
			expires_at DATETIME
		)
	`).Error)
	return db
}

// 注:转单 SQL 测试不用 SQLite runtime —— PG `ANY(text[])` 不可移植。
// 改用静态源码扫描锁定 WHERE 文本(BLOCKER-4: IS NULL 兜底),与 44-01 Task 4
// silence 过滤测试模式一致(参见 reconciliation_service_test.go 静态断言)。

// ============================================================================
// cleanupExpiredExceptions (内部辅助函数 cleanupExpiredExceptionsDirect)
// ============================================================================

// TestCleanupExpiredExceptions 验证过期规则被软停用(is_active=0→1),
// 未过期/无 expires_at 规则保持启用;软停用的记录 deleted_at 仍 NULL(审计链不断)。
func TestCleanupExpiredExceptions(t *testing.T) {
	db := setupReconExceptionTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	// 1 条过期 + 1 条未过期
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, is_active, expires_at, created_at, reason) VALUES (?, ?, ?, 0, ?, ?, ?)`,
		"rule-past", "过期规则", "192.168.0.0/16", past, past, "测试").Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, is_active, expires_at, created_at, reason) VALUES (?, ?, ?, 0, ?, ?, ?)`,
		"rule-future", "未过期规则", "10.0.0.0/8", future, time.Now(), "测试").Error)

	rowsAffected, err := cleanupExpiredExceptionsDirect(ctx, db, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected, "仅过期那条被软停用")

	// 验证状态
	var pastActive, futureActive int
	require.NoError(t, db.Raw(`SELECT is_active FROM sys_reconciliation_exception WHERE id = 'rule-past'`).Scan(&pastActive).Error)
	assert.Equal(t, 1, pastActive, "过期规则应被软停用(is_active=1)")

	require.NoError(t, db.Raw(`SELECT is_active FROM sys_reconciliation_exception WHERE id = 'rule-future'`).Scan(&futureActive).Error)
	assert.Equal(t, 0, futureActive, "未过期规则应保持启用(is_active=0)")

	// deleted_at 仍 NULL(软停用不软删除,D-R3-A4-03 审计链)
	var pastDeletedAt sql.NullTime
	require.NoError(t, db.Raw(`SELECT deleted_at FROM sys_reconciliation_exception WHERE id = 'rule-past'`).Scan(&pastDeletedAt).Error)
	assert.False(t, pastDeletedAt.Valid, "软停用 deleted_at 必须为 NULL(审计链不断,T-44-07)")
}

// TestCleanupExpiredExceptionsIdempotent 二次调用 rowsAffected=0(WHERE is_active=0 已不匹配)
func TestCleanupExpiredExceptionsIdempotent(t *testing.T) {
	db := setupReconExceptionTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-24 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, is_active, expires_at, created_at, reason) VALUES (?, ?, ?, 0, ?, ?, ?)`,
		"rule-past", "过期规则", "192.168.0.0/16", past, past, "测试").Error)

	first, err := cleanupExpiredExceptionsDirect(ctx, db, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(1), first, "第一次应停用 1 条")

	second, err := cleanupExpiredExceptionsDirect(ctx, db, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(0), second, "第二次调用应 rowsAffected=0(幂等)")
}

// TestCleanupExpiredExceptionsNoExpiresAt expires_at=nil 的规则不被停用(WHERE expires_at IS NOT NULL)
func TestCleanupExpiredExceptionsNoExpiresAt(t *testing.T) {
	db := setupReconExceptionTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, is_active, expires_at, created_at, reason) VALUES (?, ?, ?, 0, NULL, ?, ?)`,
		"rule-no-exp", "无过期规则", "192.168.0.0/16", time.Now(), "测试").Error)

	rowsAffected, err := cleanupExpiredExceptionsDirect(ctx, db, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(0), rowsAffected, "无 expires_at 的规则不被处理")
}

// TestCleanupExpiredExceptionsAlreadyDeleted deleted_at 非 NULL 的规则不被处理(WHERE deleted_at IS NULL)
func TestCleanupExpiredExceptionsAlreadyDeleted(t *testing.T) {
	db := setupReconExceptionTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-24 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, is_active, expires_at, deleted_at, created_at, reason) VALUES (?, ?, ?, 0, ?, ?, ?, ?)`,
		"rule-deleted", "已软删规则", "192.168.0.0/16", past, past, time.Now(), "测试").Error)

	rowsAffected, err := cleanupExpiredExceptionsDirect(ctx, db, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(0), rowsAffected, "软删除的规则不被处理(WHERE deleted_at IS NULL)")
}

// ============================================================================
// createWorkorderBySeverity SQL 静态源码扫描(BLOCKER-4: IS NULL 兜底)
//
// 转单 SQL 在 PG 跑真实 ANY(),SQLite 不可移植;改用源码扫描锁定 SQL 文本。
// 重点是 BLOCKER-4: WHERE 必须含 `applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions)`,
// 不能裸 `'no_workorder' != ANY(applied_actions)`(否则 applied_actions=NULL 漏转)。
// ============================================================================

// TestCreateWorkorderNoWorkorderFilterStatic 静态断言 SQL 文本含 IS NULL 兜底(BLOCKER-4)
func TestCreateWorkorderNoWorkorderFilterStatic(t *testing.T) {
	src := mustReadReconTasksSrc(t)
	assert.Contains(t, src,
		"applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions)",
		"createWorkorderBySeverity 的 WHERE 必须含 IS NULL 兜底(BLOCKER-4:防 applied_actions=NULL 漏转)")
	// 反向校验:不允许裸 != ANY 不带 IS NULL(防止回退)
	assert.NotContains(t, src,
		"AND 'no_workorder' != ANY(applied_actions))",
		"禁止裸 'no_workorder' != ANY(applied_actions)(BLOCKER-4 三值逻辑漏转风险)")
}

// TestCleanupExpiredExceptionsNoPlaceholder 静态断言:R1 placeholder 字符串已消失
func TestCleanupExpiredExceptionsNoPlaceholder(t *testing.T) {
	src := mustReadReconTasksSrc(t)
	assert.NotContains(t, src,
		"R1 placeholder, R3 真实实现",
		"R1 placeholder 字符串必须消失(R3 真实实现已落地)")
	assert.Contains(t, src, `Update("is_active", 1)`,
		"软停用实现必须用 Update(is_active, 1)")
	// WHERE 用 ? 参数化(now time.Time),NOW() 仅出现在 doc comment 中。
	// 实际运行时 SQL 等价于 PG 的 `expires_at < NOW()`(由 cron 调度方传 time.Now())。
	assert.Contains(t, src,
		`Where("expires_at IS NOT NULL AND expires_at < ? AND is_active = ? AND deleted_at IS NULL"`,
		"WHERE 必须含 expires_at + is_active=0 + deleted_at IS NULL(幂等 + 审计链)")
}

// helpers

func mustReadReconTasksSrc(t *testing.T) string {
	t.Helper()
	// go test 的工作目录是包目录,直接相对路径读取
	data, err := os.ReadFile("reconciliation_tasks.go")
	require.NoError(t, err, "must read reconciliation_tasks.go")
	return string(data)
}

// ==================== 2026-07-05 cron 基建收口回归守护 ====================
//
// incident 260705-fix-suggestion-flood 根因之一:cron 表达式硬编码内联在 reconJobs
// slice,代码 5-field vs DB 6-field 不一致。下列测试锁定:
//  1. 非 descriptor cron 必须 6-field(scheduler 用 cron.WithSeconds)
//  2. 历史脏值必须在 legacyCronOverrides 自愈黑名单
//  3. reconciliation_crons.go 是唯一权威源(reconJobs 不再内联 cron 字面量)

// TestReconCronsAreSixField 锁定:scheduler 用 cron.WithSeconds() (cron.go:186),
// 非 descriptor 表达式必须 6-field(秒分时日月周)。5-field 会被 parser 误解析。
func TestReconCronsAreSixField(t *testing.T) {
	nonDescriptor := []struct {
		name string
		expr string
	}{
		{"detectExpiredSilence", reconCronDetectExpiredSilence},
		{"cleanupExpiredExceptions", reconCronCleanupExpiredExceptions},
		{"monitorFixSuggestionMisFix", reconCronMonitorFixSuggestionMisFix},
	}
	for _, c := range nonDescriptor {
		fields := strings.Fields(c.expr)
		assert.Lenf(t, fields, 6,
			"%s cron 必须是 6-field (秒分时日月周),scheduler 用 cron.WithSeconds() 解析;当前 %d field: %q",
			c.name, len(fields), c.expr)
	}

	// descriptor 表达式(@every / @daily / ...)不归 6-field 约束,但必须以 @ 开头
	descriptors := []struct {
		name string
		expr string
	}{
		{"refreshView", reconCronRefreshView},
		{"detectLayer3", reconCronDetectLayer3},
		{"createWorkorderCritical", reconCronCreateWorkorderCritical},
		{"createWorkorderHigh", reconCronCreateWorkorderHigh},
		{"generateFixSuggestions", reconCronGenerateFixSuggestions},
	}
	for _, d := range descriptors {
		assert.Truef(t, strings.HasPrefix(d.expr, "@"),
			"%s cron 应是 descriptor 表达式(以 @ 开头): %q", d.name, d.expr)
	}
}

// TestReconJobGroupLowercase 锁定:JobGroup 统一小写,防再次出现 RECONCILIATION 大小写分裂。
func TestReconJobGroupLowercase(t *testing.T) {
	assert.Equal(t, "reconciliation", reconciliationJobGroup,
		"JobGroup 必须小写 'reconciliation'(与 migration_169 历史一致)")
}

// TestLegacyCronOverridesContainsKnownDirtyValue 守护:incident 260705 的 5-field 脏值
// 必须在自愈黑名单。任何删 map 项的改动立即失败。
func TestLegacyCronOverridesContainsKnownDirtyValue(t *testing.T) {
	assert.Contains(t, legacyCronOverrides, "7,17,27,37,47,57 * * * *",
		"历史脏 cron (5-field monitor) 必须留在 legacyCronOverrides 自愈黑名单")
	assert.Equal(t, reconCronMonitorFixSuggestionMisFix,
		legacyCronOverrides["7,17,27,37,47,57 * * * *"],
		"脏值必须映射到权威 6-field cron")
}

// TestReconJobsNoInlineCronLiteral 守护:reconJobs slice 不应再内联 cron 字面量,
// 必须引用 reconCronXxx 常量(单一权威源,防双轨漂移)。
func TestReconJobsNoInlineCronLiteral(t *testing.T) {
	src := mustReadReconTasksSrc(t)
	// 这些是历史上内联过的字面量,现应消失(改引用常量)
	bannedLiterals := []string{
		`cronExpression: "@every`,
		`cronExpression: "0 0 2`,
		`cronExpression: "0 0 3`,
		`cronExpression: "0 7,17,27,37,47,57`,
		`cronExpression: "7,17,27,37,47,57`, // incident 260705 脏值
	}
	for _, lit := range bannedLiterals {
		assert.NotContains(t, src, lit,
			"reconJobs 不应含内联 cron 字面量 %q(应引用 reconciliation_crons.go 常量)", lit)
	}
	// 应出现常量引用
	assert.Contains(t, src, "reconCronRefreshView",
		"reconJobs 必须引用 reconCronXxx 常量")
	assert.Contains(t, src, "reconciliationJobGroup",
		"reconJobs JobGroup 必须引用 reconciliationJobGroup 常量")
}

// TestJobNameToInvokeTargetRemoved 守护:jobNameToInvokeTarget switch 函数已删除。
// InvokeTarget 改为 reconJobs slice 字段直接引用常量。
func TestJobNameToInvokeTargetRemoved(t *testing.T) {
	src := mustReadReconTasksSrc(t)
	assert.NotContains(t, src, "func jobNameToInvokeTarget(",
		"jobNameToInvokeTarget 函数应已删除(InvokeTarget 改为 slice 字段引用常量)")
}
