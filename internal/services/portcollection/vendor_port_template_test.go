package portcollection

import (
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestRenderCommand_VendorActionMatrix 覆盖 3 厂商 × 5 操作 = 15 个 (vendor, action) 模板。
// 子测试命名约定：{vendor}_{action}（snake_case），便于 grep 失败定位。
func TestRenderCommand_VendorActionMatrix(t *testing.T) {
	tests := []struct {
		name     string
		vendor   models.DeviceVendor
		action   PortAction
		params   PortTemplateParams
		expected []string
	}{
		// Huawei VRP (5) — shutdown/undo/dot1x 必须显式进 interface view
		{
			name:     "huawei_shutdown",
			vendor:   models.VendorHuawei,
			action:   ActionShutdown,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "shutdown"},
		},
		{
			name:     "huawei_undo_shutdown",
			vendor:   models.VendorHuawei,
			action:   ActionUndoShutdown,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "undo shutdown"},
		},
		{
			name:     "huawei_description",
			vendor:   models.VendorHuawei,
			action:   ActionDescription,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", Description: "uplink"},
			expected: []string{"interface GE0/0/1", "description uplink"},
		},
		{
			name:     "huawei_dot1x_enable",
			vendor:   models.VendorHuawei,
			action:   ActionDot1xEnable,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "authentication-profile dot1x"},
		},
		{
			name:     "huawei_dot1x_disable",
			vendor:   models.VendorHuawei,
			action:   ActionDot1xDisable,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "undo authentication-profile dot1x"},
		},

		// H3C Comware (5) — 与 Huawei VRP 命令同源 (D-08)，shutdown/undo/dot1x 进 interface view
		{
			name:     "h3c_shutdown",
			vendor:   models.VendorH3C,
			action:   ActionShutdown,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "shutdown"},
		},
		{
			name:     "h3c_undo_shutdown",
			vendor:   models.VendorH3C,
			action:   ActionUndoShutdown,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "undo shutdown"},
		},
		{
			name:     "h3c_description",
			vendor:   models.VendorH3C,
			action:   ActionDescription,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", Description: "uplink"},
			expected: []string{"interface GE0/0/1", "description uplink"},
		},
		{
			name:     "h3c_dot1x_enable",
			vendor:   models.VendorH3C,
			action:   ActionDot1xEnable,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "dot1x enable"},
		},
		{
			name:     "h3c_dot1x_disable",
			vendor:   models.VendorH3C,
			action:   ActionDot1xDisable,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1"},
			expected: []string{"interface GE0/0/1", "undo dot1x enable"},
		},

		// Ruijie RGOS (5) — shutdown/undo 同样需进 interface view（dot1x/description 已有前缀）
		{
			name:     "ruijie_shutdown",
			vendor:   models.VendorRuijie,
			action:   ActionShutdown,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"},
			expected: []string{"interface GigabitEthernet0/0/1", "shutdown"},
		},
		{
			name:     "ruijie_undo_shutdown",
			vendor:   models.VendorRuijie,
			action:   ActionUndoShutdown,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"},
			expected: []string{"interface GigabitEthernet0/0/1", "no shutdown"},
		},
		{
			name:     "ruijie_description",
			vendor:   models.VendorRuijie,
			action:   ActionDescription,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1", Description: "uplink"},
			expected: []string{"interface GigabitEthernet0/0/1", "description uplink"},
		},
		{
			name:     "ruijie_dot1x_enable",
			vendor:   models.VendorRuijie,
			action:   ActionDot1xEnable,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"},
			expected: []string{"interface GigabitEthernet0/0/1", "dot1x port-control auto", "dot1x default-user-limit 1"},
		},
		{
			// 采集缓存 dot1x_user_limit=2 → 下发 default-user-limit 2
			// 关键回归: 非 1 端口不再被硬编码 1 错置
			name:   "ruijie_dot1x_enable_with_limit_2",
			vendor: models.VendorRuijie,
			action: ActionDot1xEnable,
			params: PortTemplateParams{
				InterfaceName:  "GigabitEthernet0/0/1",
				Dot1xUserLimit: intPtr(2),
			},
			expected: []string{"interface GigabitEthernet0/0/1", "dot1x port-control auto", "dot1x default-user-limit 2"},
		},
		{
			// nil (设备 "unlimited" 或未采集) → 兜底 1,完全向后兼容历史硬编码行为
			name:   "ruijie_dot1x_enable_limit_nil_fallback",
			vendor: models.VendorRuijie,
			action: ActionDot1xEnable,
			params: PortTemplateParams{
				InterfaceName:  "GigabitEthernet0/0/1",
				Dot1xUserLimit: nil,
			},
			expected: []string{"interface GigabitEthernet0/0/1", "dot1x port-control auto", "dot1x default-user-limit 1"},
		},
		{
			name:     "ruijie_dot1x_disable",
			vendor:   models.VendorRuijie,
			action:   ActionDot1xDisable,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"},
			expected: []string{"interface GigabitEthernet0/0/1", "no dot1x port-control auto"},
		},

		// Phase 56 v1.20.1 新增 — set_access_vlan (3 vendors)
		// RISK-03 关键字分歧：Huawei=`port default vlan`，H3C=`port access vlan`，Ruijie=`switchport access vlan`。
		{
			name:     "huawei_set_access_vlan",
			vendor:   models.VendorHuawei,
			action:   ActionSetAccessVLAN,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", VLANID: 100},
			expected: []string{"interface GE0/0/1", "port link-type access", "port default vlan 100"},
		},
		{
			name:     "h3c_set_access_vlan",
			vendor:   models.VendorH3C,
			action:   ActionSetAccessVLAN,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", VLANID: 100},
			expected: []string{"interface GE0/0/1", "port link-type access", "port access vlan 100"},
		},
		{
			name:     "ruijie_set_access_vlan",
			vendor:   models.VendorRuijie,
			action:   ActionSetAccessVLAN,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1", VLANID: 100},
			expected: []string{"interface GigabitEthernet0/0/1", "switchport mode access", "switchport access vlan 100"},
		},

		// Phase 56 v1.20.1 新增 — port_binding (3 vendors x add/remove × with/without MAC)
		// RISK-01: H3C 省略 `static`；RISK-02: Ruijie 用 Cisco 风格 MAC `aabb.ccdd.eeff`。
		{
			name:     "huawei_port_binding_add",
			vendor:   models.VendorHuawei,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", BindOp: "add", IPAddress: "10.62.25.5"},
			expected: []string{"interface GE0/0/1", "user-bind static ip-address 10.62.25.5"},
		},
		{
			name:     "huawei_port_binding_add_with_mac",
			vendor:   models.VendorHuawei,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", BindOp: "add", IPAddress: "10.62.25.5", MACAddress: "AA:BB:CC:DD:EE:FF"},
			expected: []string{"interface GE0/0/1", "user-bind static ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF"},
		},
		{
			name:     "huawei_port_binding_remove",
			vendor:   models.VendorHuawei,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", BindOp: "remove", IPAddress: "10.62.25.5", MACAddress: "AA:BB:CC:DD:EE:FF"},
			expected: []string{"interface GE0/0/1", "undo user-bind static ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF"},
		},
		{
			name:     "h3c_port_binding_add",
			vendor:   models.VendorH3C,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", BindOp: "add", IPAddress: "10.62.25.5"},
			expected: []string{"interface GE0/0/1", "user-bind ip-address 10.62.25.5"},
		},
		{
			name:     "h3c_port_binding_remove",
			vendor:   models.VendorH3C,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", BindOp: "remove", IPAddress: "10.62.25.5"},
			expected: []string{"interface GE0/0/1", "undo user-bind ip-address 10.62.25.5"},
		},
		{
			name:     "h3c_port_binding_remove_with_mac",
			vendor:   models.VendorH3C,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GE0/0/1", BindOp: "remove", IPAddress: "10.62.25.5", MACAddress: "AA:BB:CC:DD:EE:FF"},
			expected: []string{"interface GE0/0/1", "undo user-bind ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF"},
		},
		{
			name:     "ruijie_port_binding_add",
			vendor:   models.VendorRuijie,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1", BindOp: "add", IPAddress: "10.62.25.5", MACAddress: "AA:BB:CC:DD:EE:FF"},
			expected: []string{"interface GigabitEthernet0/0/1", "switchport port-security binding aabb.ccdd.eeff 10.62.25.5"},
		},
		{
			name:     "ruijie_port_binding_add_no_mac",
			vendor:   models.VendorRuijie,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1", BindOp: "add", IPAddress: "10.62.25.5"},
			expected: []string{"interface GigabitEthernet0/0/1", "switchport port-security binding 10.62.25.5"},
		},
		{
			name:     "ruijie_port_binding_remove",
			vendor:   models.VendorRuijie,
			action:   ActionPortBinding,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1", BindOp: "remove", IPAddress: "10.62.25.5", MACAddress: "AA:BB:CC:DD:EE:FF"},
			expected: []string{"interface GigabitEthernet0/0/1", "no switchport port-security binding aabb.ccdd.eeff 10.62.25.5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderCommand(tt.vendor, tt.action, tt.params)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestRenderCommand_EmptyInterfaceName 所有 action 通用前置校验。
func TestRenderCommand_EmptyInterfaceName(t *testing.T) {
	_, err := RenderCommand(models.VendorHuawei, ActionShutdown, PortTemplateParams{InterfaceName: ""})
	assert.ErrorIs(t, err, ErrEmptyInterfaceName)
}

// TestRenderCommand_UnsupportedVendor 非 Huawei/H3C/Ruijie 厂商一律拒绝。
// Cisco 显式 OUT-OF-SCOPE（REQUIREMENTS.md）。
func TestRenderCommand_UnsupportedVendor(t *testing.T) {
	_, err := RenderCommand(models.DeviceVendor("cisco"), ActionShutdown, PortTemplateParams{InterfaceName: "GE0/0/1"})
	assert.ErrorIs(t, err, ErrUnsupportedVendor)
}

// TestRenderCommand_UnknownAction 未知 action 字符串返回 ErrUnknownAction。
func TestRenderCommand_UnknownAction(t *testing.T) {
	_, err := RenderCommand(models.VendorHuawei, PortAction("bogus_action"), PortTemplateParams{InterfaceName: "GE0/0/1"})
	assert.ErrorIs(t, err, ErrUnknownAction)
}

// TestRenderCommand_DescriptionEmpty Description 为空在 description action 上必须拦截。
func TestRenderCommand_DescriptionEmpty(t *testing.T) {
	_, err := RenderCommand(models.VendorHuawei, ActionDescription, PortTemplateParams{InterfaceName: "GE0/0/1", Description: ""})
	assert.ErrorIs(t, err, ErrDescriptionEmpty)
}

// TestRenderCommand_DescriptionTooLong 长度 > 80 字符拦截（与 device_port_status.Description
// size:500 中预留的 80 UI 字符约定对齐）。
func TestRenderCommand_DescriptionTooLong(t *testing.T) {
	longDesc := strings.Repeat("x", 81)
	_, err := RenderCommand(models.VendorHuawei, ActionDescription, PortTemplateParams{InterfaceName: "GE0/0/1", Description: longDesc})
	assert.ErrorIs(t, err, ErrDescriptionTooLong)
}

// intPtr 取地址简写 —— 测试夹具中构造 *int 字面量的快捷方式。
func intPtr(v int) *int { return &v }