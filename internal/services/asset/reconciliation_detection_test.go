package asset

import (
	"context"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// DetectLayer3 Layer 3.5 例外匹配插入测试(WARN-5 端到端覆盖)
//
// 关键:本测试集验证 DetectLayer3 在 INSERT sys_data_reconciliation 时,
// 若资产 IP 命中 active 例外规则,则:
//   - ExceptionRuleID 非空(指向首条命中规则)
//   - AppliedActions 非空(合并后 actions)
//   - Severity 可能因 skip_severity / severity_override 降级(D-R3-A2-02)
//   - silence 命中仍写表(D-R3-A1-01)
//
// 详见 44-01-PLAN.md Task 4 behavior。
// ============================================================================

// TestDetectLayer3ExceptionHit 命中 global 规则,ExceptionRuleID + AppliedActions 写入
func TestDetectLayer3ExceptionHit(t *testing.T) {
	db := setupTestDB(t, "test_layer35_hit")
	// seed 1 条 active global 规则 IP 192.168.0.0/16 actions=no_alert
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception
		(id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at)
		VALUES ('rule-hit', 'hit-rule', '192.168.0.0/16', '{no_alert}', 'global', '测试原因文本', 0, 1, NULL)`).Error)

	// 资产 IP 192.168.0.10 命中
	uid := "00000000-0000-0000-0000-000000000010"
	ip := "192.168.0.10"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES (?, ?, ?, NULL, 0, NULL)`, uid, "IP-001", ip).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'alice')`, uid, uid).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)

	// 验证 INSERT 的记录含 ExceptionRuleID + AppliedActions
	var rec struct {
		ExceptionRuleID *string       `gorm:"column:exception_rule_id"`
		AppliedActions  pq.StringArray `gorm:"column:applied_actions"`
	}
	require.NoError(t, db.Raw(`SELECT exception_rule_id, applied_actions FROM sys_data_reconciliation WHERE asset_id = ?`, uid).Scan(&rec).Error)
	require.NotNil(t, rec.ExceptionRuleID, "ExceptionRuleID 应非空(命中规则)")
	assert.Equal(t, "rule-hit", *rec.ExceptionRuleID)
	assert.Contains(t, rec.AppliedActions, "no_alert")
}

// TestDetectLayer3NoExceptionNoChange 资产 IP 不命中,字段保持 nil
func TestDetectLayer3NoExceptionNoChange(t *testing.T) {
	db := setupTestDB(t, "test_layer35_nomatch")
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception
		(id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at)
		VALUES ('rule-x', 'x-rule', '192.168.0.0/16', '{no_alert}', 'global', '测试原因文本', 0, 1, NULL)`).Error)

	uid := "00000000-0000-0000-0000-000000000020"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES (?, ?, ?, NULL, 0, NULL)`, uid, "IP-002", "10.0.0.1").Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'bob')`, uid, uid).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)

	var rec struct {
		ExceptionRuleID *string       `gorm:"column:exception_rule_id"`
		AppliedActions  pq.StringArray `gorm:"column:applied_actions"`
	}
	require.NoError(t, db.Raw(`SELECT exception_rule_id, applied_actions FROM sys_data_reconciliation WHERE asset_id = ?`, uid).Scan(&rec).Error)
	assert.Nil(t, rec.ExceptionRuleID, "不命中规则 → ExceptionRuleID=nil")
}

// TestDetectLayer3SilenceStillWrites silence 命中仍写表(D-R3-A1-01)
func TestDetectLayer3SilenceStillWrites(t *testing.T) {
	db := setupTestDB(t, "test_layer35_silence")
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception
		(id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at)
		VALUES ('rule-sil', 'sil-rule', '192.168.0.0/16', '{silence}', 'global', '测试原因文本', 0, 1, NULL)`).Error)

	uid := "00000000-0000-0000-0000-000000000030"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES (?, ?, ?, NULL, 0, NULL)`, uid, "IP-003", "192.168.0.30").Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'carol')`, uid, uid).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted, "D-R3-A1-01 silence 命中仍 INSERT")

	var count int64
	require.NoError(t, db.Table("sys_data_reconciliation").Where("asset_id = ?", uid).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestDetectLayer3SkipSeverityDegrades skip_severity 降级 critical→high(D-R3-A2-02)
//
// 注意:D-R3-A2-02 降级链需在 DetectLayer3 内体现:Type B/C severity 默认 high,
// skip_severity 触发后应为 medium(high→medium)。本测试用 Type C(physical+declared
// 不匹配,severity=high)验证降级。
func TestDetectLayer3SkipSeverityDegrades(t *testing.T) {
	db := setupTestDB(t, "test_layer35_skip")
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception
		(id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at)
		VALUES ('rule-skip', 'skip-rule', '192.168.0.0/16', '{skip_severity}', 'global', '测试原因文本', 0, 1, NULL)`).Error)

	// Type C: physical+declared 都有但不匹配 → severity=high
	uid := "00000000-0000-0000-0000-000000000040"
	otherUID := "00000000-0000-0000-0000-000000000041"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES (?, ?, ?, ?, 0, NULL)`, uid, "IP-004", "192.168.0.40", otherUID).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'dave')`, uid, uid).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)

	var severity string
	require.NoError(t, db.Raw(`SELECT severity FROM sys_data_reconciliation WHERE asset_id = ?`, uid).Scan(&severity).Error)
	// Type C 默认 high,skip_severity 降级 → medium
	assert.Equal(t, "medium", severity, "Type C (high) --skip_severity--> medium")
}

// TestDetectLayer3DeptScopeMatch dept scope IP+user 命中(WARN-5 端到端覆盖)
func TestDetectLayer3DeptScopeMatch(t *testing.T) {
	db := setupTestDB(t, "test_layer35_dept_match")
	deptID := "00000000-0000-0000-0000-000000000099"
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception
		(id, name, ip_range, exception_actions, scope_type, scope_id, reason, is_active, created_by, deleted_at)
		VALUES ('rule-dept', 'dept-rule', '192.168.0.0/16', '{no_alert}', 'dept', ?, '测试原因文本', 0, 1, NULL)`, deptID).Error)

	// 资产责任人 user_id == deptID(dept scope 用 ScopeID 关联责任人 user_id,
	// 这里简化用 asset.user_id 模拟"资产属于该 dept")
	uid := "00000000-0000-0000-0000-000000000050"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES (?, ?, ?, ?, 0, NULL)`, uid, "IP-005", "192.168.0.50", deptID).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'eve')`, uid, uid).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)

	var ruleID *string
	require.NoError(t, db.Raw(`SELECT exception_rule_id FROM sys_data_reconciliation WHERE asset_id = ?`, uid).Scan(&ruleID).Error)
	require.NotNil(t, ruleID, "dept scope IP+userID 命中 → ExceptionRuleID 非空")
	assert.Equal(t, "rule-dept", *ruleID)
}

// TestDetectLayer3DeptScopeNoMatch dept scope IP 命中但 user 不匹配(WARN-5 反例)
func TestDetectLayer3DeptScopeNoMatch(t *testing.T) {
	db := setupTestDB(t, "test_layer35_dept_nomatch")
	deptA := "00000000-0000-0000-0000-0000000000a1"
	deptB := "00000000-0000-0000-0000-0000000000b2"
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception
		(id, name, ip_range, exception_actions, scope_type, scope_id, reason, is_active, created_by, deleted_at)
		VALUES ('rule-deptA', 'deptA-rule', '192.168.0.0/16', '{no_alert}', 'dept', ?, '测试原因文本', 0, 1, NULL)`, deptA).Error)

	// 资产属于 deptB,但规则 scope_id=deptA
	uid := "00000000-0000-0000-0000-000000000060"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES (?, ?, ?, ?, 0, NULL)`, uid, "IP-006", "192.168.0.60", deptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'frank')`, uid, uid).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted, "仍 INSERT(异常检测本身不阻塞),只是不命中例外规则")

	var ruleID *string
	require.NoError(t, db.Raw(`SELECT exception_rule_id FROM sys_data_reconciliation WHERE asset_id = ?`, uid).Scan(&ruleID).Error)
	assert.Nil(t, ruleID, "dept scope IP 命中但 user 不匹配 → ExceptionRuleID=nil")
}

// ============================================================================
// 防止 setupTestDB 重复定义,这里只引用同包的 setupTestDB(已定义在
// reconciliation_test.go)。抑制 unused import(time)。
// ============================================================================

var _ = time.Now

// ============================================================================
// Phase 47 R3 UPSERT 行为测试(2026-07-03)
//
// 验证 DetectLayer3 由 INSERT-only 改造为 GORM UPSERT 后的三层命中场景:
//   - TestDetectLayer3UpsertInsertNoExisting:无 open → INSERT 路径
//   - TestDetectLayer3UpsertUpdateExisting:有 open → UPDATE 路径
//   - TestDetectLayer3UpsertInsertAfterResolved:resolved 后 INSERT 新行
// ============================================================================

// TestDetectLayer3UpsertInsertNoExisting 无 open → UPSERT INSERT 路径
func TestDetectLayer3UpsertInsertNoExisting(t *testing.T) {
	db := setupTestDB(t, "test_upsert_insert")
	uid := "00000000-0000-0000-0000-000000000100"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at)
		VALUES (?, 'IP-100', '10.0.0.100', NULL, 0, NULL)`, uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username)
		VALUES (?, ?, 'alice')`, uid, uid).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)

	var rowCount int64
	require.NoError(t, db.Model(&models.SysDataReconciliation{}).
		Where("asset_id = ? AND deleted_at IS NULL", uid).Count(&rowCount).Error)
	assert.Equal(t, int64(1), rowCount, "无 open 时 UPSERT INSERT 应产生 1 行")

	var severity string
	require.NoError(t, db.Raw(`SELECT severity FROM sys_data_reconciliation WHERE asset_id = ?`, uid).
		Scan(&severity).Error)
	assert.NotEmpty(t, severity, "severity 应非空")
}

// TestDetectLayer3UpsertUpdateExisting 有 open 时的行为 (b8fd2f45 后: NULL 不冲突 → INSERT 新行, 非 UPDATE)
func TestDetectLayer3UpsertUpdateExisting(t *testing.T) {
	db := setupTestDB(t, "test_upsert_update")
	// partial unique index 已由 setupTestDB 统一建 (uniq_recon_asset_type_cat_open 三列,
	// 模拟 migration_201)。b8fd2f45 后 DetectLayer3 写 recon_category=NULL, 三列 index
	// NULL 语义不冲突 → pre-existing open (NULL) 不会被 UPDATE, DetectLayer3 INSERT 新行。
	// 防重复靠 24h 节流 guard (D-A3-02); 此处 pre-existing 48h 前 → guard 不阻止 → INSERT。

	uid := "00000000-0000-0000-0000-000000000200"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at)
		VALUES (?, 'IP-200', '10.0.0.200', NULL, 0, NULL)`, uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username)
		VALUES (?, ?, 'bob')`, uid, uid).Error)

	// 预存一条 open 冲突行(模拟卡住的 B 异常 — 该资产 has_physical 但无 declared,
	// ClassifyType 判定为 Type B,故 pre-existing 也须 conflict_type='B' 才能触发 UPSERT UPDATE)
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, raw_snapshot, physical_value, declared_value, ad_value,
		 confidence_score, detected_at, resolved_at, deleted_at)
		VALUES (?, ?, 'B', 'medium', '{}', '{}', '{}', '{}', 0.5, ?, NULL, NULL)`,
		"rec-pre-existing", uid, time.Now().Add(-48*time.Hour)).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted, "DetectLayer3 INSERT 1 行 (NULL 不冲突, 计入 inserted)")

	// b8fd2f45 (Phase 48-01) 后实际行为: 三列 partial index 含 recon_category,
	// DetectLayer3 写 recon_category=NULL → NULL 语义不与 pre-existing (NULL) 冲突 →
	// 走 INSERT 而非 UPDATE。防重复靠 24h 节流 guard (D-A3-02); 此处 pre-existing
	// detected_at=now-48h (>24h 窗口) → guard 不阻止 → INSERT 新行。
	// 结果: pre-existing (medium) + 新行 (high) = 2 行。
	// TODO(Phase 48 后续): 若 DetectLayer3 普通对账需靠 index 防重复, 需为其指定固定
	// recon_category (当前字典仅 component_serial/future_expansion, NULL 表"普通对账")。
	var rowCount int64
	require.NoError(t, db.Model(&models.SysDataReconciliation{}).
		Where("asset_id = ? AND deleted_at IS NULL", uid).Count(&rowCount).Error)
	assert.Equal(t, int64(2), rowCount, "NULL 不冲突 → pre-existing + 新 INSERT = 2 行")

	// pre-existing (medium) 保留未被覆盖; 新行 ComputeSeverity('B')='high'
	var severities []string
	require.NoError(t, db.Raw(`SELECT severity FROM sys_data_reconciliation WHERE asset_id = ? AND deleted_at IS NULL`, uid).
		Scan(&severities).Error)
	assert.Contains(t, severities, "medium", "pre-existing medium 保留 (未被覆盖)")
	assert.Contains(t, severities, "high", "新行 severity=high")
}

// TestDetectLayer3UpsertInsertAfterResolved resolved 后 INSERT 新行
func TestDetectLayer3UpsertInsertAfterResolved(t *testing.T) {
	db := setupTestDB(t, "test_upsert_resolved")
	uid := "00000000-0000-0000-0000-000000000300"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at)
		VALUES (?, 'IP-300', '10.0.0.300', NULL, 0, NULL)`, uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username)
		VALUES (?, ?, 'charlie')`, uid, uid).Error)

	// 预存一条 resolved 历史行(不参与 partial unique index 冲突)
	resolvedAt := time.Now().Add(-72 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, raw_snapshot, physical_value, declared_value, ad_value,
		 confidence_score, detected_at, resolved_at, resolved_by, deleted_at)
		VALUES (?, ?, 'D', 'medium', '{}', '{}', '{}', '{}', 0.5, ?, ?, 'manual', NULL)`,
		"rec-resolved", uid, resolvedAt, resolvedAt).Error)

	svc := &reconciliationDetectionImpl{db: db}
	inserted, _, _, _, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)

	// 验证: 历史 resolved 行保留 + 新 INSERT 1 行 = 2 行(partial index 不约束 resolved)
	var rowCount int64
	require.NoError(t, db.Model(&models.SysDataReconciliation{}).
		Where("asset_id = ? AND deleted_at IS NULL", uid).Count(&rowCount).Error)
	assert.Equal(t, int64(2), rowCount, "resolved 历史 + 新 open 行 = 2 行")
}
