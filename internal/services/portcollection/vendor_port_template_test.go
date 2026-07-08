package portcollection

import (
	"strings"
	"testing"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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
			expected: []string{"interface GigabitEthernet0/0/1", "dot1x port-control auto"},
		},
		{
			name:     "ruijie_dot1x_disable",
			vendor:   models.VendorRuijie,
			action:   ActionDot1xDisable,
			params:   PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"},
			expected: []string{"interface GigabitEthernet0/0/1", "no dot1x port-control"},
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