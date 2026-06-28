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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
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

// setupReconDataTestDB 构造内存 sqlite + sys_data_reconciliation 表(用于转单 SQL 测试)
//
// applied_actions 用 TEXT 模拟 PG TEXT[](SQLite 无原生数组类型),
// 测试 SQL 用 ANY() 不可移植;故转单 SQL 测试改用静态源码扫描 + 集成断言双轨制。
func setupReconDataTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:recon_data_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_data_reconciliation (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			asset_id TEXT,
			conflict_type TEXT,
			severity TEXT,
			physical_value TEXT,
			declared_value TEXT,
			ad_value TEXT,
			confidence_score REAL,
			raw_snapshot TEXT,
			asset_ip TEXT,
			exception_rule_id TEXT,
			applied_actions TEXT,
			detected_at DATETIME,
			resolved_at DATETIME,
			resolved_by TEXT,
			resolution_note TEXT,
			workorder_id TEXT
		)
	`).Error)
	return db
}

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

	rowsAffected, err := cleanupExpiredExceptionsDirect(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected, "仅过期那条被软停用")

	// 验证状态
	var pastActive, futureActive int
	require.NoError(t, db.Raw(`SELECT is_active FROM sys_reconciliation_exception WHERE id = 'rule-past'`).Scan(&pastActive).Error)
	assert.Equal(t, 1, pastActive, "过期规则应被软停用(is_active=1)")

	require.NoError(t, db.Raw(`SELECT is_active FROM sys_reconciliation_exception WHERE id = 'rule-future'`).Scan(&futureActive).Error)
	assert.Equal(t, 0, futureActive, "未过期规则应保持启用(is_active=0)")

	// deleted_at 仍 NULL(软停用不软删除,D-R3-A4-03 审计链)
	var pastDeletedAt *time.Time
	require.NoError(t, db.Raw(`SELECT deleted_at FROM sys_reconciliation_exception WHERE id = 'rule-past'`).Scan(&pastDeletedAt).Error)
	assert.Nil(t, pastDeletedAt, "软停用 deleted_at 必须为 NULL(审计链不断,T-44-07)")
}

// TestCleanupExpiredExceptionsIdempotent 二次调用 rowsAffected=0(WHERE is_active=0 已不匹配)
func TestCleanupExpiredExceptionsIdempotent(t *testing.T) {
	db := setupReconExceptionTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-24 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, is_active, expires_at, created_at, reason) VALUES (?, ?, ?, 0, ?, ?, ?)`,
		"rule-past", "过期规则", "192.168.0.0/16", past, past, "测试").Error)

	first, err := cleanupExpiredExceptionsDirect(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), first, "第一次应停用 1 条")

	second, err := cleanupExpiredExceptionsDirect(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), second, "第二次调用应 rowsAffected=0(幂等)")
}

// TestCleanupExpiredExceptionsNoExpiresAt expires_at=nil 的规则不被停用(WHERE expires_at IS NOT NULL)
func TestCleanupExpiredExceptionsNoExpiresAt(t *testing.T) {
	db := setupReconExceptionTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, is_active, expires_at, created_at, reason) VALUES (?, ?, ?, 0, NULL, ?, ?)`,
		"rule-no-exp", "无过期规则", "192.168.0.0/16", time.Now(), "测试").Error)

	rowsAffected, err := cleanupExpiredExceptionsDirect(ctx, db)
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

	rowsAffected, err := cleanupExpiredExceptionsDirect(ctx, db)
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
	assert.Contains(t, src,
		"expires_at IS NOT NULL AND expires_at < NOW() AND is_active = ? AND deleted_at IS NULL",
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
