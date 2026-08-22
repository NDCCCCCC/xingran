package operations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TableName 锁定测试(74-11 escalation):表名是 DB 契约,GORM AutoMigrate/
// 归档 SQL/前端列配置都依赖这些字符串,静默改动即 FATA。与
// internal/models/status_constants_test.go 同范式。
func TestOperationsTableNames(t *testing.T) {
	assert.Equal(t, "ops_buildings", OpsBuilding{}.TableName())
	assert.Equal(t, "ops_dedicated_lines", OpsDedicatedLine{}.TableName())
	assert.Equal(t, "ops_doors", Door{}.TableName())
	assert.Equal(t, "ops_floors", OpsFloor{}.TableName())
	assert.Equal(t, "ops_floor_plan_texts", FloorPlanText{}.TableName())
	assert.Equal(t, "ops_info_points", OpsInfoPoint{}.TableName())
	assert.Equal(t, "ops_room_devices", OpsRoomDevice{}.TableName())
	assert.Equal(t, "ops_room_network_devices", OpsRoomNetworkDevice{}.TableName())
	assert.Equal(t, "ops_room_photos", OpsRoomPhoto{}.TableName())
	assert.Equal(t, "ops_server_rooms", OpsServerRoom{}.TableName())
	assert.Equal(t, "ops_walls", Wall{}.TableName())
}
