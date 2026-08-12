package asset

import (
	"net"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// makeRule helper 构造 SysReconciliationException
func makeRule(id, cidr string, actions []string, sevOverride *string, scopeType string, scopeID *string, conflictTypes []string) models.SysReconciliationException {
	return models.SysReconciliationException{
		BaseModel: models.BaseModel{ID: id},
		Name:      "rule-" + id,
		IPRange:   cidr,
		ExceptionActions: pq.StringArray(actions),
		SeverityOverride: sevOverride,
		ScopeType:        scopeType,
		ScopeID:          scopeID,
		ConflictTypes:    pq.StringArray(conflictTypes),
		IsActive:         0,
	}
}

// strPtr helper
func strPtr(s string) *string { return &s }

// ============================================================================
// TestApplySkipSeverity — skip_severity 降级链(D-R3-A2-02)
// ============================================================================

func TestApplySkipSeverity(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"critical → high", "critical", "high"},
		{"high → medium", "high", "medium"},
		{"medium → low", "medium", "low"},
		{"low stays low (不再降)", "low", "low"},
		{"未知 severity 兜底 low", "unknown", "low"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applySkipSeverity(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// TestMergeActions — 多规则 actions 并集 + skip_severity 降级(D-R3-A2-01/02)
// ============================================================================

func TestMergeActions(t *testing.T) {
	matched := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"no_alert"}, nil, "global", nil, nil)},
		{rule: makeRule("r2", "192.168.0.0/16", []string{"no_notice", "skip_severity"}, nil, "global", nil, nil)},
	}
	actions, finalSev, isSilence := mergeActions("critical", matched, "B")
	// actions 并集
	assert.Contains(t, actions, "no_alert")
	assert.Contains(t, actions, "no_notice")
	assert.Contains(t, actions, "skip_severity")
	assert.Len(t, actions, 3, "并集去重后 3 个 actions")
	// skip_severity 降级 critical→high
	assert.Equal(t, "high", finalSev)
	// 无 silence action
	assert.False(t, isSilence)
}

// TestMergeActionsSeverityOverride — severity_override 多规则取最低(D-R3-A2-01)
func TestMergeActionsSeverityOverride(t *testing.T) {
	matched := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{}, strPtr("medium"), "global", nil, nil)},
		{rule: makeRule("r2", "192.168.0.0/16", []string{}, strPtr("low"), "global", nil, nil)},
	}
	_, finalSev, _ := mergeActions("critical", matched, "B")
	assert.Equal(t, "low", finalSev, "多规则 override 取最低(最宽松)")
}

// TestMergeActionsSilenceUnion — silence 进入并集触发 isSilence
func TestMergeActionsSilenceUnion(t *testing.T) {
	matched := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"silence"}, nil, "global", nil, nil)},
	}
	_, _, isSilence := mergeActions("high", matched, "B")
	assert.True(t, isSilence)
}

// TestMergeActionsSkipThenOverride — 先 skip_severity 降级,再 override 取更宽(D-R3-A2-02)
func TestMergeActionsSkipThenOverride(t *testing.T) {
	// critical --skip--> high --override=low--> low(取更宽)
	matched := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"skip_severity"}, strPtr("low"), "global", nil, nil)},
	}
	_, finalSev, _ := mergeActions("critical", matched, "B")
	assert.Equal(t, "low", finalSev)
}

// ============================================================================
// TestMatchException — CIDR 匹配 + scope 双条件(A3-01) + 空类型(A3-02) + 非法 CIDR 跳过(A1-03)
// ============================================================================

func TestMatchExceptionSingleGlobalRule(t *testing.T) {
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.1.0/24", []string{"no_alert"}, nil, "global", nil, nil), ipNet: mustParseCIDR(t, "192.168.1.0/24")},
	}
	ruleID, actions, _, _ := matchException(rules, "192.168.1.10", "", "B")
	assert.Equal(t, "r1", ruleID)
	assert.Contains(t, actions, "no_alert")
}

func TestMatchExceptionCIDRBoundary(t *testing.T) {
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.1.0/24", []string{"no_alert"}, nil, "global", nil, nil), ipNet: mustParseCIDR(t, "192.168.1.0/24")},
	}
	// 网络地址与广播地址都命中
	id1, _, _, _ := matchException(rules, "192.168.1.0", "", "B")
	assert.Equal(t, "r1", id1)
	id2, _, _, _ := matchException(rules, "192.168.1.255", "", "B")
	assert.Equal(t, "r1", id2)
}

func TestMatchExceptionNoMatch(t *testing.T) {
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.1.0/24", []string{"no_alert"}, nil, "global", nil, nil), ipNet: mustParseCIDR(t, "192.168.1.0/24")},
	}
	ruleID, actions, _, _ := matchException(rules, "10.0.0.1", "", "B")
	assert.Empty(t, ruleID)
	assert.Nil(t, actions)
}

func TestMatchExceptionDeptScopeDoubleCondition(t *testing.T) {
	deptA := "dept-a-uuid"
	deptB := "dept-b-uuid"
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"no_alert"}, nil, "dept", &deptA, nil), ipNet: mustParseCIDR(t, "192.168.0.0/16")},
	}
	// 资产属 deptA 命中
	id1, _, _, _ := matchException(rules, "192.168.0.10", deptA, "B")
	assert.Equal(t, "r1", id1, "dept scope + IP CIDR 双条件命中")
	// 资产属 deptB 不命中(不传 deptB 时也视为不命中)
	id2, _, _, _ := matchException(rules, "192.168.0.10", deptB, "B")
	assert.Empty(t, id2, "dept scope IP 命中但 user 不匹配 → 不命中")
}

func TestMatchExceptionUserScopeDoubleCondition(t *testing.T) {
	userA := "user-a-uuid"
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"no_alert"}, nil, "user", &userA, nil), ipNet: mustParseCIDR(t, "192.168.0.0/16")},
	}
	id1, _, _, _ := matchException(rules, "192.168.0.10", userA, "B")
	assert.Equal(t, "r1", id1)
	id2, _, _, _ := matchException(rules, "192.168.0.10", "", "B")
	assert.Empty(t, id2, "user scope 不传 userID → 不命中")
}

func TestMatchExceptionEmptyConflictTypesMatchesAll(t *testing.T) {
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"no_alert"}, nil, "global", nil, nil), ipNet: mustParseCIDR(t, "192.168.0.0/16")},
	}
	// 空 ConflictTypes 匹配全部 B-F
	for _, ct := range []string{"B", "C", "D", "E", "F"} {
		id, _, _, _ := matchException(rules, "192.168.0.10", "", ct)
		assert.Equal(t, "r1", id, "空 ConflictTypes 应匹配全部 B-F,ct=%s", ct)
	}
}

func TestMatchExceptionConflictTypesFilter(t *testing.T) {
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"no_alert"}, nil, "global", nil, []string{"B", "C"}), ipNet: mustParseCIDR(t, "192.168.0.0/16")},
	}
	// 仅匹配 B/C
	id1, _, _, _ := matchException(rules, "192.168.0.10", "", "B")
	assert.Equal(t, "r1", id1, "ConflictTypes 含 B 应命中")
	// 不匹配 D
	id2, _, _, _ := matchException(rules, "192.168.0.10", "", "D")
	assert.Empty(t, id2, "ConflictTypes 不含 D 应不命中")
}

func TestMatchExceptionIPv6(t *testing.T) {
	rules := []compiledRule{
		{rule: makeRule("r1", "2001:db8::/32", []string{"no_alert"}, nil, "global", nil, nil), ipNet: mustParseCIDR(t, "2001:db8::/32")},
	}
	id, _, _, _ := matchException(rules, "2001:db8::1", "", "B")
	assert.Equal(t, "r1", id)
}

func TestMatchExceptionInvalidIP(t *testing.T) {
	rules := []compiledRule{
		{rule: makeRule("r1", "192.168.0.0/16", []string{"no_alert"}, nil, "global", nil, nil), ipNet: mustParseCIDR(t, "192.168.0.0/16")},
	}
	id, _, _, _ := matchException(rules, "not-an-ip", "", "B")
	assert.Empty(t, id)
}

// ============================================================================
// TestPreloadActiveRules — 跳过非法 CIDR(D-R3-A1-03)
// ============================================================================

func TestPreloadActiveRulesSkipsInvalidCIDR(t *testing.T) {
	db := setupMatcherTestDB(t)
	// 1 条合法 + 1 条非法 CIDR + 1 条已停用(is_active=1)
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at) VALUES
		('r-valid', 'valid-rule', '192.168.0.0/16', '{no_alert}', 'global', '测试原因文本', 0, 1, NULL),
		('r-bad',    'bad-rule',   '999.999.0.0/16', '{no_alert}', 'global', '测试原因文本', 0, 1, NULL),
		('r-inactive','inactive-rule','10.0.0.0/8', '{no_alert}', 'global', '测试原因文本', 1, 1, NULL)`).Error)

	rules := preloadActiveRules(db)
	// 仅保留合法 + 启用规则,非法 CIDR 与停用规则被跳过
	assert.Len(t, rules, 1, "应跳过非法 CIDR + 停用规则")
	assert.Equal(t, "r-valid", rules[0].rule.ID)
}

// ============================================================================
// helpers
// ============================================================================

func mustParseCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, ipNet, err := net.ParseCIDR(cidr)
	require.NoError(t, err, "test setup ParseCIDR failed: %s", cidr)
	return ipNet
}

func setupMatcherTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:matcher_test_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
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
