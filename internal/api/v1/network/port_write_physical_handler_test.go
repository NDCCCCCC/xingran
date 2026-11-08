package network

// FIX-04（v1.53 Phase 138）：物理参数 HTTP 端点 /network/ports/write/physical 测试。
//
// 复用 port_write_handler_test.go 的 newTestHandler / invokeHandler / mockPortWriteService。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
)

// TestPhysical138_EndpointSuccessOperlogAudit 成功路径：200 + operlog.Record
// + audit.after_value 从 PortResult.Extra 正确落库（速度/双工/MTU 三动作）。
func TestPhysical138_EndpointSuccessOperlogAudit(t *testing.T) {
	cases := []struct {
		name       string
		body       PhysicalWriteRequest
		wantAfter  string
		wantAction portcollection.PortAction
	}{
		{
			name:       "speed 1000",
			body:       PhysicalWriteRequest{PortID: "p1", Action: "set_speed", Speed: 1000},
			wantAfter:  `{"speed":1000}`,
			wantAction: portcollection.ActionSetSpeed,
		},
		{
			name:       "speed auto",
			body:       PhysicalWriteRequest{PortID: "p1", Action: "set_speed", SpeedMode: "auto"},
			wantAfter:  `{"speed":"auto"}`,
			wantAction: portcollection.ActionSetSpeed,
		},
		{
			name:       "duplex full",
			body:       PhysicalWriteRequest{PortID: "p1", Action: "set_duplex", Duplex: "full"},
			wantAfter:  `{"duplex":"full"}`,
			wantAction: portcollection.ActionSetDuplex,
		},
		{
			name:       "mtu 9000",
			body:       PhysicalWriteRequest{PortID: "p1", Action: "set_mtu", MTU: 9000},
			wantAfter:  `{"mtu":9000}`,
			wantAction: portcollection.ActionSetMTU,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
			// 让 mock 返回带 Extra 的 PortResult（service 成功路径写入的回显）
			extra := map[string]interface{}{}
			switch tc.body.Action {
			case "set_speed":
				if tc.body.SpeedMode == "auto" {
					extra["speed"] = "auto"
				} else {
					extra["speed"] = tc.body.Speed
				}
			case "set_duplex":
				extra["duplex"] = tc.body.Duplex
			case "set_mtu":
				extra["mtu"] = tc.body.MTU
			}
			mockSvc.physicalOut = &portwrite.PortResult{
				PortID: "p1", Action: tc.wantAction, Status: "succeeded", Extra: extra,
			}

			w := invokeHandler("PhysicalWrite", h.PhysicalWrite, tc.body)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

			// operlog.Record 被调用一次，module=端口管理，OperTypeUpdate
			assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
			assert.Equal(t, ModulePortWrite, mockOperLog.lastTitle)

			// audit 行 after_value 正确
			var audit models.PortWriteAudit
			require.NoError(t, sqlDB.Where("port_id = ?", "p1").First(&audit).Error)
			assert.JSONEq(t, tc.wantAfter, string(audit.AfterValue))
			assert.Equal(t, string(tc.wantAction), audit.Action)
			assert.Equal(t, "succeeded", audit.Status)
		})
	}
}

// TestPhysical138_EndpointSentinelInvalid service sentinel → 400，不写 audit / operlog。
func TestPhysical138_EndpointSentinelInvalid(t *testing.T) {
	cases := []struct {
		name string
		body PhysicalWriteRequest
		err  error
	}{
		{"speed 非法", PhysicalWriteRequest{PortID: "p1", Action: "set_speed", Speed: 999}, portwrite.ErrSpeedInvalid},
		{"duplex 非法", PhysicalWriteRequest{PortID: "p1", Action: "set_duplex", Duplex: "x"}, portwrite.ErrDuplexInvalid},
		{"mtu 越界", PhysicalWriteRequest{PortID: "p1", Action: "set_mtu", MTU: 1}, portwrite.ErrMTUOutOfRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
			mockSvc.physicalErr = tc.err

			w := invokeHandler("PhysicalWrite", h.PhysicalWrite, tc.body)
			require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

			// sentinel 路径不写 audit、不调 operlog（by-design）
			var count int64
			require.NoError(t, sqlDB.Model(&models.PortWriteAudit{}).Count(&count).Error)
			assert.Equal(t, int64(0), count)
			assert.Equal(t, 0, mockOperLog.recordAsyncCalls)
		})
	}
}

// TestPhysical138_EndpointInvalidActionBinding 未知 action 被 gin binding oneof 拦截 400。
func TestPhysical138_EndpointInvalidActionBinding(t *testing.T) {
	h, _, _, _ := newTestHandler(t)
	w := invokeHandler("PhysicalWrite", h.PhysicalWrite, PhysicalWriteRequest{PortID: "p1", Action: "set_unknown"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestPhysical138_BuildAfterValue 直接锁 buildAfterValue 三动作分支（审计真相源）。
func TestPhysical138_BuildAfterValue(t *testing.T) {
	assert.JSONEq(t, `{"speed":1000}`, string(buildAfterValue(portcollection.ActionSetSpeed,
		&portwrite.PortResult{Extra: map[string]interface{}{"speed": 1000}})))
	assert.JSONEq(t, `{"speed":"auto"}`, string(buildAfterValue(portcollection.ActionSetSpeed,
		&portwrite.PortResult{Extra: map[string]interface{}{"speed": "auto"}})))
	assert.JSONEq(t, `{"duplex":"half"}`, string(buildAfterValue(portcollection.ActionSetDuplex,
		&portwrite.PortResult{Extra: map[string]interface{}{"duplex": "half"}})))
	assert.JSONEq(t, `{"mtu":1500}`, string(buildAfterValue(portcollection.ActionSetMTU,
		&portwrite.PortResult{Extra: map[string]interface{}{"mtu": 1500}})))
	// nil / 空 Extra 兜底 {}（向后兼容）
	assert.JSONEq(t, `{}`, string(buildAfterValue(portcollection.ActionSetMTU, nil)))
}

// TestPhysical138_RouterRegistersEndpoint router 源码含 /physical 注册（组级权限复用）。
func TestPhysical138_RouterRegistersEndpoint(t *testing.T) {
	src := readFile(t, "port_write_router.go")
	assert.Contains(t, src, `write.POST("/physical", handler.PhysicalWrite)`)
	// 组级权限链路仍在（一处覆盖新端点）
	assert.Contains(t, src,
		"middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core)")

	// 防止误用裸 json 解码：确认序列化工具可用
	_, err := json.Marshal(PhysicalWriteRequest{PortID: "p1", Action: "set_speed"})
	require.NoError(t, err)
}
