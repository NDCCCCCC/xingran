package operations

// 本文件承接 internal/models/models_hooks_80_03_test.go 中原属根包
// operations.go（Floor / ServerRoom / RoomDevice）的 TableName 断言。
// 根包 internal/models/operations.go 已于 2026-11-16 整体删除（见
// .planning/notes/261113-hardcode-audit-handoff.md R6），子包是 ops_* 表
// 映射的唯一事实源；断言迁到子包后覆盖全部 12 个模型（根包当时只锁 3 个）。

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpsModelsTableName(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		// ── 原根包 operations.go 模型（3 个）──
		{"OpsFloor", OpsFloor{}.TableName(), "ops_floors"},
		{"OpsServerRoom", OpsServerRoom{}.TableName(), "ops_server_rooms"},
		{"OpsRoomDevice", OpsRoomDevice{}.TableName(), "ops_room_devices"},

		// ── 子包其余模型（补齐覆盖）──
		{"OpsBuilding", OpsBuilding{}.TableName(), "ops_buildings"},
		{"OpsDedicatedLine", OpsDedicatedLine{}.TableName(), "ops_dedicated_lines"},
		{"OpsInfoPoint", OpsInfoPoint{}.TableName(), "ops_info_points"},
		{"OpsRoomNetworkDevice", OpsRoomNetworkDevice{}.TableName(), "ops_room_network_devices"},
		{"OpsRoomPhoto", OpsRoomPhoto{}.TableName(), "ops_room_photos"},
		{"Door", Door{}.TableName(), "ops_doors"},
		{"Wall", Wall{}.TableName(), "ops_walls"},
		{"FloorPlanText", FloorPlanText{}.TableName(), "ops_floor_plan_texts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
