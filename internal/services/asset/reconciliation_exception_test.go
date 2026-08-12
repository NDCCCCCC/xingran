package asset

import (
	"context"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Validators
// ============================================================================

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{"IPv4 /16 ok", "192.168.0.0/16", false},
		{"IPv4 /32 ok", "10.0.0.1/32", false},
		{"IPv6 ok", "2001:db8::/32", false},
		{"no mask err", "192.168.1.10", true},
		{"octet overflow err", "999.999.0.0/16", true},
		{"not a cidr err", "not-a-cidr", true},
		{"empty err", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCIDR(tt.cidr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateActions(t *testing.T) {
	tests := []struct {
		name    string
		actions []string
		wantErr bool
	}{
		{"single no_alert ok", []string{"no_alert"}, false},
		{"multiple whitelist ok", []string{"no_alert", "silence", "no_workorder"}, false},
		{"all 5 ok", []string{"no_alert", "no_notice", "no_workorder", "skip_severity", "silence"}, false},
		{"empty err", []string{}, true},
		{"nil err", nil, true},
		{"evil value err", []string{"no_alert", "evil"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActions(pq.StringArray(tt.actions))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSeverityOverride(t *testing.T) {
	tests := []struct {
		name string
		sev  *string
		ok   bool
	}{
		{"nil ok", nil, true},
		{"low ok", strPtr("low"), true},
		{"medium ok", strPtr("medium"), true},
		{"high ok", strPtr("high"), true},
		{"critical err (Pitfall 8)", strPtr("critical"), false},
		{"evil err", strPtr("evil"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSeverityOverride(tt.sev)
			if tt.ok {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateReason(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		ok     bool
	}{
		{"len=10 ok", "1234567890", true},
		{"len=20 ok", "12345678901234567890", true},
		{"中文 ≥10 字符 ok", "这是至少十个字符的中文说明", true},
		{"len=9 err", "123456789", false},
		{"empty err", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReason(tt.reason)
			if tt.ok {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// ============================================================================
// CRUD tests (SQLite in-memory)
// ============================================================================

func setupExceptionRuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:excrule_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
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

func TestCreateExceptionRule(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)
	expires := time.Now().Add(24 * time.Hour)
	req := &CreateExceptionRuleRequest{
		Name:             "test-rule",
		IPRange:          "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"},
		ScopeType:        "global",
		Reason:           "测试原因至少10字符",
		ExpiresAt:        &expires,
	}
	rule, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, rule)
	assert.Equal(t, "test-rule", rule.Name)
	assert.Equal(t, "192.168.0.0/16", rule.IPRange)
	assert.Equal(t, 0, rule.IsActive, "默认启用(is_active=0)")
	assert.NotEmpty(t, rule.ID)
}

func TestCreateExceptionRuleValidationFail(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)

	// 非法 CIDR
	_, err := svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "x", IPRange: "999.999.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, ScopeType: "global", Reason: "1234567890",
	})
	assert.Error(t, err)

	// reason < 10 字符
	_, err = svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "x", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, ScopeType: "global", Reason: "short",
	})
	assert.Error(t, err)

	// actions 非法值
	_, err = svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "x", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"evil"}, ScopeType: "global", Reason: "1234567890",
	})
	assert.Error(t, err)

	// severity_override=critical (Pitfall 8)
	_, err = svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "x", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, SeverityOverride: strPtr("critical"),
		ScopeType: "global", Reason: "1234567890",
	})
	assert.Error(t, err)
}

func TestUpdateExceptionRule(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)

	// 先 create
	rule, err := svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "orig", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, ScopeType: "global", Reason: "1234567890",
	})
	require.NoError(t, err)

	// 更新
	err = svc.Update(context.Background(), rule.ID, &UpdateExceptionRuleRequest{
		Name: "updated", IPRange: "10.0.0.0/8",
		ExceptionActions: pq.StringArray{"no_notice"}, Reason: "更新后的原因足够十个字符",
	})
	require.NoError(t, err)

	// 验证
	got, err := svc.GetByID(context.Background(), rule.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "updated", got.Name)
	assert.Equal(t, "10.0.0.0/8", got.IPRange)
}

func TestUpdateExceptionRuleNotExist(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)
	err := svc.Update(context.Background(), "non-existent-uuid", &UpdateExceptionRuleRequest{
		Name: "x", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, Reason: "1234567890",
	})
	assert.Error(t, err)
}

func TestDeleteExceptionRule(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)

	rule, err := svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "to-delete", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, ScopeType: "global", Reason: "1234567890",
	})
	require.NoError(t, err)

	err = svc.Delete(context.Background(), rule.ID)
	require.NoError(t, err)

	// 软删除:List 查不到(GetByID 也查不到)
	list, err := svc.List(context.Background(), &ExceptionRuleListParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), list.Total, "软删除后 List 应返回 0")
}

// ============================================================================
// MatchTest (GiST SQL SQLite 降级 — 使用内存匹配)
// ============================================================================

func TestMatchTestGlobalScope(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)

	// seed 1 条 global 规则
	_, err := svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "global-rule", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, ScopeType: "global", Reason: "1234567890",
	})
	require.NoError(t, err)

	// MatchTest
	result, err := svc.MatchTest(context.Background(), "192.168.0.10", "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.MatchedRules, "应命中 global 规则")
	assert.True(t, result.NeedsUserDept, "未传 userID/deptID → needsUserDept=true")
	// MergedActions 应含 no_alert
	found := false
	for _, a := range result.MergedActions {
		if a == "no_alert" {
			found = true
		}
	}
	assert.True(t, found, "MergedActions 含 no_alert")
}

func TestMatchTestWithUserID(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)

	userA := "user-a-uuid"
	_, err := svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "user-rule", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, ScopeType: "user", ScopeID: &userA,
		Reason: "1234567890",
	})
	require.NoError(t, err)

	// 传 userID 命中
	result, err := svc.MatchTest(context.Background(), "192.168.0.10", userA, "")
	require.NoError(t, err)
	assert.NotEmpty(t, result.MatchedRules)
	assert.False(t, result.NeedsUserDept, "已传 userID → needsUserDept=false")

	// 不传 userID 不命中
	result2, err := svc.MatchTest(context.Background(), "192.168.0.10", "", "")
	require.NoError(t, err)
	// 仅 user scope 规则,不传 user → 空
	for _, r := range result2.MatchedRules {
		if r.ScopeType == "user" {
			t.Fatalf("user scope 规则在未传 userID 时不应命中")
		}
	}
}

func TestMatchTestExcludesInactiveRules(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)

	// 直接 INSERT 一条停用规则(is_active=1),绕过 service 的默认启用
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception
		(id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at)
		VALUES ('inactive', 'inactive-rule', '192.168.0.0/16', '{no_alert}', 'global', '1234567890', 1, 1, NULL)`).Error)

	result, err := svc.MatchTest(context.Background(), "192.168.0.10", "", "")
	require.NoError(t, err)
	for _, r := range result.MatchedRules {
		if r.ID == "inactive" {
			t.Fatalf("停用规则(is_active=1)不应参与 MatchTest")
		}
	}
}

// ============================================================================
// List/GetByID R1 已覆盖,这里只验证 list 兼容
// ============================================================================

func TestExceptionRuleListR3(t *testing.T) {
	db := setupExceptionRuleTestDB(t)
	svc := NewReconciliationExceptionService(db)
	_, err := svc.Create(context.Background(), &CreateExceptionRuleRequest{
		Name: "rule-1", IPRange: "192.168.0.0/16",
		ExceptionActions: pq.StringArray{"no_alert"}, ScopeType: "global", Reason: "1234567890",
	})
	require.NoError(t, err)

	result, err := svc.List(context.Background(), &ExceptionRuleListParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	list, ok := result.List.([]models.SysReconciliationException)
	require.True(t, ok, "List 应返回 []SysReconciliationException")
	assert.Len(t, list, 1)
	assert.Equal(t, "rule-1", list[0].Name)
}
