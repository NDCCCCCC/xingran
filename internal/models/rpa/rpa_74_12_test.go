package rpa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// 74-11 escalation gap-closure: internal/models/rpa — TableName 锁定
// (DB 契约,同 operations 先例)+ StringArray Scan/Value roundtrip +
// AuditLog 状态谓词。
// =====================================================================

func TestRPATableNames(t *testing.T) {
	assert.Equal(t, "sys_rpa_audit_logs", AuditLog{}.TableName())
	assert.Equal(t, "sys_rpa_credentials", RPACredential{}.TableName())
	assert.Equal(t, "sys_rpa_sessions", RPASession{}.TableName())
	assert.Equal(t, "sys_rpa_executions", Execution{}.TableName())
	assert.Equal(t, "sys_rpa_notifications", Notification{}.TableName())
	assert.Equal(t, "sys_rpa_schedules", Schedule{}.TableName())
	assert.Equal(t, "sys_rpa_tasks", Task{}.TableName())
	assert.Equal(t, "sys_rpa_templates", Template{}.TableName())
	assert.Equal(t, "sys_rpa_variables", Variable{}.TableName())
	assert.Equal(t, "sys_rpa_workers", Worker{}.TableName())
}

func TestStringArray_ScanValue(t *testing.T) {
	var sa StringArray

	// nil → 空数组
	require.NoError(t, sa.Scan(nil))
	assert.Empty(t, sa)

	// JSON 字符串 → 解析
	require.NoError(t, sa.Scan(`["a","b"]`))
	assert.Equal(t, StringArray{"a", "b"}, sa)

	// []byte JSON
	var sa2 StringArray
	require.NoError(t, sa2.Scan([]byte(`["x"]`)))
	assert.Equal(t, StringArray{"x"}, sa2)

	// 空字节 → 空数组
	var sa3 StringArray
	require.NoError(t, sa3.Scan([]byte{}))
	assert.Empty(t, sa3)

	// 非 JSON → 空数组(静默)
	var sa4 StringArray
	require.NoError(t, sa4.Scan("not-json"))
	assert.Empty(t, sa4)

	// 不支持的类型 → 错误
	var sa5 StringArray
	assert.Error(t, sa5.Scan(12345))

	// Value → JSON 字节([]uint8 形式)
	v, err := StringArray{"a", "b"}.Value()
	require.NoError(t, err)
	assert.Equal(t, []byte(`["a","b"]`), v)
}

func TestAuditLog_StatusPredicates(t *testing.T) {
	success := AuditLog{Result: AuditResultSuccess}
	assert.True(t, success.IsSuccess())
	assert.False(t, success.IsFailed())

	failed := AuditLog{Result: AuditResultFailed}
	assert.False(t, failed.IsSuccess())
	assert.True(t, failed.IsFailed())

	other := AuditLog{Result: "running"}
	assert.False(t, other.IsSuccess())
	assert.False(t, other.IsFailed())

	// BeforeCreate 钩子 noop
	assert.NoError(t, (&AuditLog{}).BeforeCreate(nil))
}
