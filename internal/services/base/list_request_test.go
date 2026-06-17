package base

import (
	"reflect"
	"testing"
)

// resolve 用反射访问 resolveSort 的返回值以避免内部命名变更。
// 直接调用 resolveSort 即可（与 ApplySort 在同一包）。

func TestResolveSort_EmptyColumn_ReturnsFalse(t *testing.T) {
	allowed := map[string]string{"createdAt": "created_at"}
	col, dir, ok := ResolveSort(BaseListRequest{OrderByColumn: ""}, allowed)
	if ok {
		t.Fatalf("expected ok=false for empty OrderByColumn, got col=%q dir=%q", col, dir)
	}
}

func TestResolveSort_NilAllowed_ReturnsFalse(t *testing.T) {
	col, dir, ok := ResolveSort(BaseListRequest{OrderByColumn: "x"}, nil)
	if ok {
		t.Fatalf("expected ok=false for nil allowed, got col=%q dir=%q", col, dir)
	}
}

func TestResolveSort_ValidField_AscByDefault(t *testing.T) {
	allowed := map[string]string{
		"createdAt": "sys_user.created_at",
		"username":  "sys_user.username",
	}
	col, dir, ok := ResolveSort(BaseListRequest{OrderByColumn: "createdAt"}, allowed)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if col != "sys_user.created_at" {
		t.Errorf("col=%q, want sys_user.created_at", col)
	}
	if dir != "ASC" {
		t.Errorf("dir=%q, want ASC (nil IsAsc defaults to asc)", dir)
	}
}

func TestResolveSort_IsAscNil_DefaultsToAsc(t *testing.T) {
	allowed := map[string]string{"name": "name"}
	_, dir, ok := ResolveSort(BaseListRequest{OrderByColumn: "name", IsAsc: nil}, allowed)
	if !ok || dir != "ASC" {
		t.Errorf("expected ok=true, dir=ASC, got ok=%v dir=%q", ok, dir)
	}
}

func TestResolveSort_IsAscTrue_AppliesAsc(t *testing.T) {
	asc := true
	allowed := map[string]string{"name": "name"}
	_, dir, ok := ResolveSort(BaseListRequest{OrderByColumn: "name", IsAsc: &asc}, allowed)
	if !ok || dir != "ASC" {
		t.Errorf("expected ok=true, dir=ASC, got ok=%v dir=%q", ok, dir)
	}
}

func TestResolveSort_IsAscFalse_AppliesDesc(t *testing.T) {
	desc := false
	allowed := map[string]string{"name": "name"}
	_, dir, ok := ResolveSort(BaseListRequest{OrderByColumn: "name", IsAsc: &desc}, allowed)
	if !ok || dir != "DESC" {
		t.Errorf("expected ok=true, dir=DESC, got ok=%v dir=%q", ok, dir)
	}
}

func TestResolveSort_InvalidField_Ignored(t *testing.T) {
	allowed := map[string]string{"name": "name"}
	_, _, ok := ResolveSort(BaseListRequest{OrderByColumn: "password"}, allowed)
	if ok {
		t.Fatalf("expected ok=false for non-whitelisted field 'password'")
	}
}

// TestResolveSort_SQLInjectionAttempts 关键安全测试：所有 SQL 注入尝试都必须被静默拒绝。
// 失败此测试意味着白名单失效,绝不能通过。
func TestResolveSort_SQLInjectionAttempts(t *testing.T) {
	allowed := map[string]string{"id": "id", "name": "name"}
	injections := []string{
		"id; DROP TABLE users;--",
		"id OR 1=1",
		"' OR '1'='1",
		"id; UPDATE users SET admin=1",
		"id/**/UNION/**/SELECT",
		"id; SELECT pg_sleep(10)",
		"name', created_at, 'x",
		"id ASC; DROP TABLE foo",
		"id -- comment",
		"1; DELETE FROM sys_user",
		"pg_sleep(60)",
	}
	for _, inj := range injections {
		_, _, ok := ResolveSort(BaseListRequest{OrderByColumn: inj}, allowed)
		if ok {
			t.Errorf("SQL injection should be rejected: %q", inj)
		}
	}
}

func TestResolveSort_NestedField_Ignored(t *testing.T) {
	allowed := map[string]string{"name": "name"}
	_, _, ok := ResolveSort(BaseListRequest{OrderByColumn: "pool.pool_name"}, allowed)
	if ok {
		t.Fatalf("nested field notation should not be accepted as map key")
	}
}

func TestResolveSort_EmptyAllowedMap(t *testing.T) {
	_, _, ok := ResolveSort(BaseListRequest{OrderByColumn: "anything"}, map[string]string{})
	if ok {
		t.Fatalf("empty allowed map should reject any field")
	}
}

func TestResolveSort_ExactKeyMatch_CaseSensitive(t *testing.T) {
	allowed := map[string]string{"createdAt": "created_at"}
	// "createdat" (lowercase) 不应匹配 "createdAt" (camelCase)
	_, _, ok := ResolveSort(BaseListRequest{OrderByColumn: "createdat"}, allowed)
	if ok {
		t.Fatalf("case-sensitive match expected: 'createdat' should not match 'createdAt'")
	}
	// 正确匹配
	_, _, ok = ResolveSort(BaseListRequest{OrderByColumn: "createdAt"}, allowed)
	if !ok {
		t.Fatalf("'createdAt' should match 'createdAt'")
	}
}

func TestSortedKeys_StableOrder(t *testing.T) {
	// 日志输出需要稳定顺序,避免重复日志 diff
	allowed := map[string]string{"z": "z", "a": "a", "m": "m"}
	got := sortedKeys(allowed)
	want := []string{"a", "m", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortedKeys: got %v, want %v", got, want)
	}
}

func TestSortedKeys_EmptyMap(t *testing.T) {
	got := sortedKeys(map[string]string{})
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}
