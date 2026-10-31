package network

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PortSecWriteRequest 端口安全写请求（Phase 125 PWX-02；单端点五动作）。
//
//   - portsec_enable / portsec_disable / portsec_sticky：无附加参数
//   - portsec_max_mac：maxMacCount 必填（1-4096）
//   - portsec_static_bind：macAddress + vlanId 必填（critical，影响合法用户接入）
//
// MAC 与接口唯一性：sys_device_port_status 无绑定表（Phase 56 Pitfall 6 同源），
// 由设备端 duplicate detection 兜底；系统侧校验 MAC 格式 + VLAN 范围。
type PortSecWriteRequest struct {
	PortID string `json:"portId" binding:"required"`
	Action string `json:"action" binding:"required,oneof=portsec_enable portsec_disable portsec_max_mac portsec_sticky portsec_static_bind"`
	// portsec_max_mac
	MaxMacCount int `json:"maxMacCount,omitempty" binding:"omitempty,min=1,max=4096"`
	// portsec_static_bind
	MACAddress string `json:"macAddress,omitempty"`
	VLANID     int    `json:"vlanId,omitempty" binding:"omitempty,min=1,max=4094"`
	// PORTW-06 管理员强制跳过不可达预检
	SkipPreflight bool `json:"skipPreflight,omitempty"`
	// UI-02 操作原因
	Reason string `json:"reason,omitempty"`
}

// PortSecWrite 端口安全端口写 (POST /network/ports/write/portsec)。
//
// OperType 分流：enable/sticky/max_mac → Update(2)；disable → Status(10)；
// static_bind → Create(1)（新增绑定记录语义）。
func (h *PortWriteHandler) PortSecWrite(c *gin.Context) {
	var req PortSecWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	operType := operlog.OperTypeUpdate
	switch req.Action {
	case "portsec_disable":
		operType = operlog.OperTypeStatus
	case "portsec_static_bind":
		operType = operlog.OperTypeCreate
	}
	params := portwrite.PortSecParams{
		MaxMACCount: req.MaxMacCount,
		MACAddress:  req.MACAddress,
		VLANID:      req.VLANID,
	}
	result, err := h.service.PortSecWrite(c.Request.Context(), req.PortID,
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
		case errors.Is(err, portwrite.ErrPortSecParamsInvalid),
			errors.Is(err, portwrite.ErrMaxMacOutOfRange),
			errors.Is(err, portwrite.ErrVlanIdOutOfRange),
			errors.Is(err, portwrite.ErrMACAddressInvalid):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusBadRequest, err.Error())
		}
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operType)
	response.Success(c, result)
}
