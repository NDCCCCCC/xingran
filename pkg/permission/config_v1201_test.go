package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRoutePermissionRegistry_V1201PortWriteRoutes 验证 v1.20.1 (Phase 56 W3) 新增的
// 2 个端口写命令端点出现在 route-permission registry 中，且复用 NetworkPortWrite 常量。
//
// registry 行仅为可发现性用途（design.md §5 + WR-05 deferred item）；
// 实际 RBAC 拦截由 v1.19 组级 RequirePermissions 中间件一处覆盖。
func TestRoutePermissionRegistry_V1201PortWriteRoutes(t *testing.T) {
	perms := GetRoutePermissions()

	cases := []struct {
		path        string
		method      string
		description string
	}{
		{"/network/ports/write/set-access-vlan", "POST", "修改端口 access VLAN"},
		{"/network/ports/write/port-binding", "POST", "端口绑定（IP/MAC/Port 静态绑定）"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			var found *RoutePermission
			for i := range perms {
				if perms[i].Path == tc.path && perms[i].Method == tc.method {
					found = &perms[i]
					break
				}
			}
			assert.NotNil(t, found, "registry must contain row for %s %s", tc.method, tc.path)
			if found == nil {
				return
			}
			assert.Equal(t, NetworkPortWrite, found.Permission,
				"%s must reuse NetworkPortWrite constant (no new perm constant per design.md §5)", tc.path)
			assert.Equal(t, tc.description, found.Description,
				"%s registry description mismatch", tc.path)
		})
	}
}

// TestRoutePermissionRegistry_V119PortWriteRoutesIntact 防御性回归：v1.19 6 个端点的
// registry 行未被 Phase 56 W3 改动破坏。
func TestRoutePermissionRegistry_V119PortWriteRoutesIntact(t *testing.T) {
	perms := GetRoutePermissions()

	expectedPaths := []string{
		"/network/ports/write/shutdown",
		"/network/ports/write/undo-shutdown",
		"/network/ports/write/description",
		"/network/ports/write/dot1x-enable",
		"/network/ports/write/dot1x-disable",
		"/network/ports/write/batch",
	}
	for _, p := range expectedPaths {
		var found *RoutePermission
		for i := range perms {
			if perms[i].Path == p && perms[i].Method == "POST" {
				found = &perms[i]
				break
			}
		}
		assert.NotNil(t, found, "v1.19 registry row missing: %s", p)
		if found != nil {
			assert.Equal(t, NetworkPortWrite, found.Permission, "v1.19 row %s must still use NetworkPortWrite", p)
		}
	}
}

// TestGetPermissionByPath_V1201Routes lookups via reflection-based path matcher
// return NetworkPortWrite for the 2 new routes.
func TestGetPermissionByPath_V1201Routes(t *testing.T) {
	for _, p := range []string{
		"/network/ports/write/set-access-vlan",
		"/network/ports/write/port-binding",
	} {
		assert.Equal(t, NetworkPortWrite, GetPermissionByPath(p, "POST"),
			"GetPermissionByPath(%s, POST) must return NetworkPortWrite", p)
	}
}
