//go:build integration

package tests

import "testing"

// TestNormalizeAPIKey 归一化单测：docs.go 路径参数形态混用（{id} 与 :id 并存）。
func TestNormalizeAPIKey(t *testing.T) {
	cases := []struct{ method, path, want string }{
		{"POST", "/api/v1/system/users/list", "POST /system/users/list"},
		{"POST", "/api/v1/system/users/:id/reset-password", "POST /system/users/:id/reset-password"},
		{"GET", "/system/departments/role-dept-tree-select/{roleId}", "GET /system/departments/role-dept-tree-select/:roleId"},
		{"DELETE", "/system/dashboards/{id}", "DELETE /system/dashboards/:id"},
		{"PUT", "/system/settings/preferences", "PUT /system/settings/preferences"},
		{"GET", "/ops/floor/{id}/users", "GET /ops/floor/:id/users"},
	}
	for _, c := range cases {
		if got := normalizeAPIKey(c.method, c.path); got != c.want {
			t.Errorf("normalizeAPIKey(%q,%q) = %q, want %q", c.method, c.path, got, c.want)
		}
	}
}
