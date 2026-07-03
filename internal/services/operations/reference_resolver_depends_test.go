//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// TestResolveBatchWithCondition_DeviceIdScope 验证 DependsOn 的 device_id 条件
// 是否真正生效 (B+E 根因调查, 2026-07-01)。
//
// 复现场景: RS8607E 每台都有 GE5/44, sys_device_port_status 有两行
// (deviceA/portA/GE5/44) + (deviceB/portB/GE5/44)。DependsOn scope deviceA 时,
// ResolveBatchWithCondition 应只返回 portA, 不应跨设备 first-match 返回 portB。
//
// 5F003 实测: ip.device_id=aca124c8 但 port_id 指向 515f4c58 的 GE5/44 (跨设备错配)。
// 若本测试 PASS (返回 portA) → scope 生效, 5F003 错配根因在别处 (depID 错/分组 bug)。
// 若本测试 FAIL (返回 portB 或两行) → scope 失效, 修 ResolveBatchWithCondition。
func TestResolveBatchWithCondition_DeviceIdScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.Exec(`CREATE TABLE sys_device_port_status (id TEXT PRIMARY KEY, device_id TEXT, interface_name TEXT)`).Error)

	// deviceA + deviceB 都有 GE5/44 (跨设备同名, 模拟 RS8607E 多台同接口)
	assert.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('portA', 'deviceA', 'GE5/44')`).Error)
	assert.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('portB', 'deviceB', 'GE5/44')`).Error)

	resolver := NewReferenceResolver(db)
	result, err := resolver.ResolveBatchWithCondition(context.Background(),
		"sys_device_port_status.interface_name", []string{"GE5/44"},
		map[string]string{"device_id": "deviceA"})
	assert.NoError(t, err)

	// 断言: scope deviceA 时只返回 portA
	assert.Equal(t, "portA", result["GE5/44"],
		"DependsOn scope deviceA 应返回 portA; 若返回 portB 说明 device_id 条件未生效 (跨设备 first-match)")
	assert.Len(t, result, 1, "scope 后应只 1 个匹配, 不应跨设备返回多个")
}

// TestResolveBatchWithCondition_DeviceIdScope_NoMatch 验证 scope 的 device 下
// 没有 interface 时返回空 (不跨设备 fallback)。模拟 5F003 的 060e5a69 没 GE5/24。
func TestResolveBatchWithCondition_DeviceIdScope_NoMatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.Exec(`CREATE TABLE sys_device_port_status (id TEXT PRIMARY KEY, device_id TEXT, interface_name TEXT)`).Error)

	// 只有 deviceB 有 GE5/24, deviceA 没有
	assert.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('portB', 'deviceB', 'GE5/24')`).Error)

	resolver := NewReferenceResolver(db)
	result, err := resolver.ResolveBatchWithCondition(context.Background(),
		"sys_device_port_status.interface_name", []string{"GE5/24"},
		map[string]string{"device_id": "deviceA"})
	assert.NoError(t, err)

	// scope deviceA 但 deviceA 没 GE5/24 → 应返回空 (不跨设备 fallback 到 deviceB)
	assert.Empty(t, result, "scope deviceA 但无 GE5/24 时应返回空; 若返回 portB 说明跨设备 fallback (scope 失效)")
}
