package network

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// TrunkWriteRequest trunk/hybrid 写请求（Phase 124 PWX-01；单端点三动作）。
//
//   - action：trunk_set_native | trunk_set_allowed | trunk_set_hybrid
//   - trunk_set_native：nativeVlan 必填（1-4094）
//   - trunk_set_allowed：allowedVlans[] 必填 + mode（add|remove|replace，默认 replace）
//   - trunk_set_hybrid：hybridTagged[]/hybridUntagged[] 至少一项，且不可重叠
//     （华为设备侧会拒绝重叠；H3C 设备侧静默优先 untagged 不报错 —— 系统侧校验
//     是唯一防线，service 层 ErrHybridOverlap → 400）
type TrunkWriteRequest struct {
	PortID string `json:"portId" binding:"required"`
	Action string `json:"action" binding:"required,oneof=trunk_set_native trunk_set_allowed trunk_set_hybrid"`
	// trunk_set_native
	NativeVLAN int `json:"nativeVlan,omitempty" binding:"omitempty,min=1,max=4094"`
	// trunk_set_allowed
	AllowedVLANs []int  `json:"allowedVlans,omitempty"`
	AllowedMode  string `json:"mode,omitempty" binding:"omitempty,oneof=add remove replace"`
	// trunk_set_hybrid
	HybridTagged   []int `json:"hybridTagged,omitempty"`
	HybridUntagged []int `json:"hybridUntagged,omitempty"`
	// PORTW-06 管理员强制跳过不可达预检
	SkipPreflight bool `json:"skipPreflight,omitempty"`
	// UI-02 操作原因
	Reason string `json:"reason,omitempty"`
}

// TrunkWrite trunk/hybrid 端口写 (POST /network/ports/write/trunk)。
//
// OperType=Update(=2)；trunk 变更风险 critical，管线复用 Phase 121/123
// （渲染/NoOp/互斥/预检/快照回滚全链路）。
func (h *PortWriteHandler) TrunkWrite(c *gin.Context) {
	var req TrunkWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	params := portwrite.TrunkParams{
		NativeVLAN:     req.NativeVLAN,
		AllowedVLANs:   req.AllowedVLANs,
		AllowedMode:    req.AllowedMode,
		HybridTagged:   req.HybridTagged,
		HybridUntagged: req.HybridUntagged,
	}
	result, err := h.service.TrunkWrite(c.Request.Context(), req.PortID,
		portcollection.PortAction(req.Action), params, utils.GetUsername(c), writeOptsFrom(req.SkipPreflight)...)
	if err != nil {
		switch {
		case errors.Is(err, portwrite.ErrPortNotFound):
			response.Error(c, http.StatusNotFound, "端口不存在")
		case errors.Is(err, portwrite.ErrDeviceNotFound):
			response.Error(c, http.StatusNotFound, "设备不存在")
		case errors.Is(err, portwrite.ErrPortWriteLocked):
			response.Error(c, http.StatusConflict, "端口正在被其他写操作处理，请稍后重试: "+err.Error())
		case errors.Is(err, portwrite.ErrDeviceUnreachable):
			response.Error(c, http.StatusServiceUnavailable, "设备不可达（预检失败）: "+err.Error())
		case errors.Is(err, portwrite.ErrHybridOverlap),
			errors.Is(err, portwrite.ErrAllowedModeInvalid),
			errors.Is(err, portwrite.ErrTrunkParamsInvalid),
			errors.Is(err, portwrite.ErrVlanIdOutOfRange):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			// 渲染层错误（不支持厂商/未知 action）同为入参问题 → 400
			response.Error(c, http.StatusBadRequest, err.Error())
		}
		return
	}
	response.Success(c, result)
}
