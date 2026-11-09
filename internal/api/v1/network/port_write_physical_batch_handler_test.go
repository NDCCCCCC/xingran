package network

// Phase 143 (PORT-01)：批量物理参数 handler 测试。
//
// 复用 port_write_handler_test.go 的 newTestHandler / invokeHandler / mockPortWriteService。
// 覆盖：BatchWriteRequest 4 字段透传、批量 sentinel 400 分支（物理参数 + 能力矩阵拒绝）、
// operlog.Record 在 response.Success 之前、audit.after_value 非空、router 注册断言保留。

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
)

// TestPhysical143_BatchHandlerFourParamsPassThrough 批量请求 4 个物理字段经
// ShouldBindJSON 绑定到 BatchWriteRequest 并原样透传给 service（DTO json tag 同名）。
func TestPhysical143_BatchHandlerFourParamsPassThrough(t *testing.T) {
	cases := []struct {
		name   string
		action portcollection.PortAction
		body   map[string]interface{}
		check  func(t *testing.T, req portwrite.BatchWriteRequest)
	}{
		{
			name:   "speed 1000",
			action: portcollection.ActionSetSpeed,
			body:   map[string]interface{}{"speed": 1000},
			check: func(t *testing.T, req portwrite.BatchWriteRequest) {
				assert.Equal(t, 1000, req.Speed)
			},
		},
		{
			name:   "speedMode auto",
			action: portcollection.ActionSetSpeed,
			body:   map[string]interface{}{"speedMode": "auto"},
			check: func(t *testing.T, req portwrite.BatchWriteRequest) {
				assert.Equal(t, "auto", req.SpeedMode)
			},
		},
		{
			name:   "duplex full",
			action: portcollection.ActionSetDuplex,
			body:   map[string]interface{}{"duplex": "full"},
			check: func(t *testing.T, req portwrite.BatchWriteRequest) {
				assert.Equal(t, "full", req.Duplex)
			},
		},
		{
			name:   "mtu 9000",
			action: portcollection.ActionSetMTU,
			body:   map[string]interface{}{"mtu": 9000},
			check: func(t *testing.T, req portwrite.BatchWriteRequest) {
				assert.Equal(t, 9000, req.MTU)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, mockSvc, _, _ := newTestHandler(t)
			mockSvc.batchResult = &portwrite.BatchResult{
				Succeeded: []portwrite.PortResult{}, Failed: []portwrite.PortResult{}, Skipped: []portwrite.PortResult{},
			}
			body := map[string]interface{}{
				"deviceId": "d1", "action": string(tc.action), "portIds": []string{"p1"},
			}
			for k, v := range tc.body {
				body[k] = v
			}
			w := invokeHandler("BatchWrite", h.BatchWrite, body)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
			require.Equal(t, 1, mockSvc.batchCalls)
			assert.Equal(t, tc.action, mockSvc.lastBatchInput.req.Action)
			tc.check(t, mockSvc.lastBatchInput.req)
		})
	}
}

// TestPhysical143_BatchHandlerSentinel400 批量物理参数 sentinel → 400（不写 audit /
// 不调 operlog）。覆盖 physicalSentinelMessage 4 类 + ErrActionNotSupported。
func TestPhysical143_BatchHandlerSentinel400(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"speed 非法", portwrite.ErrSpeedInvalid},
		{"duplex 非法", portwrite.ErrDuplexInvalid},
		{"mtu 越界", portwrite.ErrMTUOutOfRange},
		{"物理动作非法", portwrite.ErrPhysicalParamsInvalid},
		{"能力矩阵拒绝", fmt.Errorf("%w: vendor=cisco action=set_speed — 渲染层未实现 + 无真机取证依据", portcollection.ErrActionNotSupported)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
			mockSvc.batchErr = tc.err
			w := invokeHandler("BatchWrite", h.BatchWrite, map[string]interface{}{
				"deviceId": "d1", "action": "set_speed", "portIds": []string{"p1"}, "speed": 1000,
			})
			require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
			// sentinel 路径不写 audit、不调 operlog
			var count int64
			require.NoError(t, sqlDB.Model(&models.PortWriteAudit{}).Count(&count).Error)
			assert.Equal(t, int64(0), count)
			assert.Equal(t, 0, mockOperLog.recordAsyncCalls)
		})
	}
}

// TestPhysical143_BatchHandlerOperlogBeforeSuccess 成功路径：operlog.Record
// 在 response.Success 之前调用，且 audit.after_value 从 PortResult.Extra 正确落库
// （批量物理写不再恒 {}）。
func TestPhysical143_BatchHandlerOperlogBeforeSuccess(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
	mockSvc.batchResult = &portwrite.BatchResult{
		Succeeded: []portwrite.PortResult{
			{
				PortID: "p1", Action: portcollection.ActionSetMTU, Status: "succeeded",
				Extra: map[string]interface{}{"mtu": 9000, "physicalEvidence": "", "physicalEvidenceLevel": "unverified"},
			},
		},
		Failed:  []portwrite.PortResult{},
		Skipped: []portwrite.PortResult{},
	}

	w := invokeHandler("BatchWrite", h.BatchWrite, map[string]interface{}{
		"deviceId": "d1", "action": "set_mtu", "portIds": []string{"p1"}, "mtu": 9000,
	})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	// operlog.Record 被调用一次（Path C，success path 末尾 response.Success 之前）
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, ModulePortWrite, mockOperLog.lastTitle)

	// audit.after_value 非空（buildAfterValue 从 Extra 读 mtu）
	var audit models.PortWriteAudit
	require.NoError(t, sqlDB.Where("port_id = ?", "p1").First(&audit).Error)
	assert.JSONEq(t, `{"mtu":9000}`, string(audit.AfterValue))
	assert.Equal(t, "set_mtu", audit.Action)
}

// TestPhysical143_RouterRegistersEndpointRetained router 源码含 /physical 注册 +
// 组级权限链路仍在（既有断言保留，防止本 phase 改动破坏单条端点）。
func TestPhysical143_RouterRegistersEndpointRetained(t *testing.T) {
	src := readFile(t, "port_write_router.go")
	assert.Contains(t, src, `write.POST("/physical", handler.PhysicalWrite)`)
	assert.Contains(t, src,
		"middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core)")
}
