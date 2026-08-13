package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSelectScope_InheritPermsShortCircuit D-13: inherit_perms=true 短路,
// 走 default 限额,不做 action-based 检查(即使 scopes 是细粒度 permission code)
func TestSelectScope_InheritPermsShortCircuit(t *testing.T) {
	scope, allowed := SelectScope([]string{"system:user:list"}, true, "list")
	assert.Equal(t, "default", scope)
	assert.True(t, allowed)
}

// TestSelectScope_ExactMatch D-11/D-12: list → read,scopes 含 read 精确匹配
func TestSelectScope_ExactMatch(t *testing.T) {
	scope, allowed := SelectScope([]string{"read"}, false, "list")
	assert.Equal(t, "read", scope)
	assert.True(t, allowed)
}

// TestSelectScope_ExactMatchWrite D-12: edit → write,scopes 含 write 精确匹配
func TestSelectScope_ExactMatchWrite(t *testing.T) {
	scope, allowed := SelectScope([]string{"write"}, false, "edit")
	assert.Equal(t, "write", scope)
	assert.True(t, allowed)
}

// TestSelectScope_AdminOverride D-12: edit 需要 write,scopes 只有 read+admin,
// admin 最高限额覆盖
func TestSelectScope_AdminOverride(t *testing.T) {
	scope, allowed := SelectScope([]string{"read", "admin"}, false, "edit")
	assert.Equal(t, "admin", scope)
	assert.True(t, allowed)
}

// TestSelectScope_FailClosed D-12: scopes 不含 required scope 且无 admin → 拒绝,
// 无 fallback 到 default
func TestSelectScope_FailClosed(t *testing.T) {
	scope, allowed := SelectScope([]string{"read"}, false, "edit")
	assert.Equal(t, "", scope)
	assert.False(t, allowed)
}

// TestSelectScope_EmptyScopes D-12: 空 scopes → fail-closed
func TestSelectScope_EmptyScopes(t *testing.T) {
	scope, allowed := SelectScope([]string{}, false, "list")
	assert.Equal(t, "", scope)
	assert.False(t, allowed)
}

// TestSelectScope_DefaultReadFallback D-11: 未知 action → 默认 read 兜底
func TestSelectScope_DefaultReadFallback(t *testing.T) {
	scope, allowed := SelectScope([]string{"read"}, false, "unknown_action")
	assert.Equal(t, "read", scope)
	assert.True(t, allowed)
}

// TestSelectScope_MultiScopeNotFirst D-12 核心: 多 scope 时选 action 匹配的
// (list → read),而非任意 scopes[0]=admin — 替代原 getScopeFromContext 的 scopes[0] 语义
func TestSelectScope_MultiScopeNotFirst(t *testing.T) {
	scope, allowed := SelectScope([]string{"admin", "write", "read"}, false, "list")
	assert.Equal(t, "read", scope)
	assert.True(t, allowed)
}

// TestSelectScope_MultiScopeWrite D-12: 多 scope 时 edit → write 精确匹配优先于 admin
func TestSelectScope_MultiScopeWrite(t *testing.T) {
	scope, allowed := SelectScope([]string{"admin", "write", "read"}, false, "edit")
	assert.Equal(t, "write", scope)
	assert.True(t, allowed)
}
