package commands

import (
	"strings"
	"testing"
)

func TestMapEndpoint(t *testing.T) {
	cases := []struct {
		method, path                      string
		wantModule                        string
		wantRes                           []string
		wantAction                        string
		wantArgs                          int
	}{
		{"POST", "/system/users/list", "system", []string{"user"}, "list", 0},
		{"POST", "/system/users", "system", []string{"user"}, "create", 0},
		{"POST", "/system/users/:id", "system", []string{"user"}, "get", 1},
		{"POST", "/system/users/:id/update", "system", []string{"user"}, "update", 1},
		{"POST", "/system/users/:id/delete", "system", []string{"user"}, "delete", 1},
		{"POST", "/system/users/:id/reset-password", "system", []string{"user"}, "reset-password", 1},
		{"POST", "/system/roles/list", "system", []string{"role"}, "list", 0},
		{"POST", "/system/departments/tree", "system", []string{"department"}, "tree", 0},
		{"POST", "/system/ad-domain/configs/list", "system", []string{"ad-domain", "config"}, "list", 0},
		{"POST", "/system/ad-domain/computers/list", "system", []string{"ad-domain", "computer"}, "list", 0},
		{"POST", "/system/ad-domain/groups/:id/members", "system", []string{"ad-domain", "group"}, "members", 1},
		{"POST", "/system/dashboards/templates/:id/create", "system", []string{"dashboard", "template"}, "create", 1},
		{"GET", "/system/auth/public-key", "system", []string{"auth"}, "public-key", 0},
		{"POST", "/system/dict-data/list", "system", []string{"dict-data"}, "list", 0},
		{"POST", "/system/departments/{id}/users", "system", []string{"department"}, "users", 1},
		{"POST", "/system/user/unlock", "system", []string{"user"}, "unlock", 0},
		{"POST", "/system/dashboards/:id/versions", "system", []string{"dashboard"}, "versions", 1},
		{"POST", "/system/dashboards/:id/versions/:versionId/restore", "system", []string{"dashboard", "version"}, "restore", 2},
		{"POST", "/system/configs/key/:configKey", "system", []string{"config", "key"}, "get", 1},
		{"POST", "/system/notice/:id/channels", "system", []string{"notice"}, "channels", 1},
		{"POST", "/system/ad-domain/users/:id", "system", []string{"ad-domain", "user"}, "get", 1},
		{"POST", "/system/departments/:id/status", "system", []string{"department"}, "status", 1},
	}
	for _, c := range cases {
		got := MapEndpoint(c.method, c.path)
		if got.Module != c.wantModule || got.Action != c.wantAction {
			t.Errorf("MapEndpoint(%s %s) = %v/%s, want %v/%s",
				c.method, c.path, got.ResourcePath, got.Action, c.wantRes, c.wantAction)
			continue
		}
		if strings.Join(got.ResourcePath, "/") != strings.Join(c.wantRes, "/") {
			t.Errorf("MapEndpoint(%s %s) ResourcePath = %v, want %v", c.method, c.path, got.ResourcePath, c.wantRes)
		}
		if len(got.PathArgs) != c.wantArgs {
			t.Errorf("MapEndpoint(%s %s) PathArgs = %v, want %d 个", c.method, c.path, got.PathArgs, c.wantArgs)
		}
	}
}

func TestSingular(t *testing.T) {
	cases := map[string]string{
		"users":       "user",
		"roles":       "role",
		"depts":       "dept",
		"menus":       "menu",
		"ad-domain":   "ad-domain",
		"dict-data":   "dict-data",
		"business":    "business",
		"status":      "status",
		"my-notices":  "my-notice",
		"departments": "department",
	}
	for in, want := range cases {
		if got := Singular(in); got != want {
			t.Errorf("Singular(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderPath(t *testing.T) {
	got := RenderPath("/api/v1/system/users/:id", []string{"id"}, []string{"abc-123"})
	if got != "/api/v1/system/users/abc-123" {
		t.Fatalf("got %q", got)
	}
	got = RenderPath("/api/v1/system/departments/{id}/users", []string{"id"}, []string{"u1"})
	if got != "/api/v1/system/departments/u1/users" {
		t.Fatalf("got %q", got)
	}
	// 缺省时保留占位
	got = RenderPath("/api/v1/system/users/:id", []string{"id"}, nil)
	if got != "/api/v1/system/users/<id>" {
		t.Fatalf("got %q", got)
	}
}

func TestSingularIrregular(t *testing.T) {
	cases := map[string]string{
		"categories":  "category",
		"discoveries": "discovery",
	}
	for in, want := range cases {
		if got := Singular(in); got != want {
			t.Errorf("Singular(%q) = %q, want %q", in, got, want)
		}
	}
}
