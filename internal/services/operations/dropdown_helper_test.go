package operations

import (
	"strings"
	"testing"
)

func TestBuildDropdownCacheKey(t *testing.T) {
	tests := []struct {
		name    string
		entity  string
		filters map[string]any
		want    string
		checkFn func(t *testing.T, got string)
	}{
		{
			name:    "空 filters → :all 后缀",
			entity:  "building",
			filters: nil,
			want:    "dropdown:building:all",
		},
		{
			name:    "空 map → :all 后缀",
			entity:  "floor",
			filters: map[string]interface{}{},
			want:    "dropdown:floor:all",
		},
		{
			name:    "单 filter → :h 前缀",
			entity:  "workstation",
			filters: map[string]interface{}{"name": "foo"},
			checkFn: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, "dropdown:workstation:h") {
					t.Errorf("got %q, want prefix %q", got, "dropdown:workstation:h")
				}
				// SHA1 截断 8 字节 = 16 hex chars
				if len(got) != len("dropdown:workstation:h")+16 {
					t.Errorf("got %q, hash length %d, want 16", got, len(got)-len("dropdown:workstation:h"))
				}
			},
		},
		{
			name:    "nil 值跳过 → 视为空",
			entity:  "building",
			filters: map[string]interface{}{"orgId": nil, "name": nil},
			want:    "dropdown:building:all",
		},
		{
			name:   "多 filter → key 稳定(同输入同输出)",
			entity: "floor",
			filters: map[string]interface{}{
				"buildingId": "abc-123",
				"status":     0,
			},
			checkFn: func(t *testing.T, got string) {
				// 第二次相同输入必须得到相同输出(排序保证)
				got2 := BuildDropdownCacheKey("floor", map[string]interface{}{
					"status":     0,
					"buildingId": "abc-123",
				})
				if got != got2 {
					t.Errorf("cache key 不稳定:\n第一次: %s\n第二次: %s", got, got2)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDropdownCacheKey(tt.entity, tt.filters)
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("BuildDropdownCacheKey() = %q, want %q", got, tt.want)
				}
			}
			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

func TestDropdownMaxRows(t *testing.T) {
	// 硬上限契约:防止前端误用。改了需同时改前端 opsApi.ts searchOptions 文档注释。
	if DropdownMaxRows != 50 {
		t.Errorf("DropdownMaxRows = %d, want 50 (前端契约)", DropdownMaxRows)
	}
}