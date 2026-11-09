package network

// FIX-04（v1.53 Phase 138）：物理参数「速率 / 双工 / MTU 调整」HTTP 端点。
//
// 沿用 v1.20.1 端口写基建：
//   - 组级 RBAC（port_write_router.go 的 /write 组 network:port:write，一处覆盖）
//   - operlog（success path 末尾 response.Success 之前 OperTypeUpdate）
//   - 审计 after_value（走 execSinglePort → buildAuditRow → buildAfterValue，
//     从 PortResult.Extra 读 speed/duplex/mtu）
//
// 与 PortSecWrite / TrunkWrite 同构：单端点三动作，service 层 validator 为真相源。

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PhysicalWriteRequest 物理参数写请求（FIX-04；单端点三动作）。
//
//   - set_speed：speedMode=="auto" 或 speed ∈ {10,100,1000,10000}
//   - set_duplex：duplex ∈ {half, full, auto}
//   - set_mtu：mtu ∈ [1280, 9216]
//
// binding 只做形状兜底，枚举/范围真相源在 service 层（validatePhysicalRequest）。
type PhysicalWriteRequest struct {
	PortID    string `json:"portId" binding:"required"`
	Action    string `json:"action" binding:"required,oneof=set_speed set_duplex set_mtu"`
	Speed     int    `json:"speed,omitempty"`
	SpeedMode string `json:"speedMode,omitempty"`
	Duplex    string `json:"duplex,omitempty"`
	MTU       int    `json:"mtu,omitempty"`
	// PORTW-06 管理员强制跳过不可达预检
	SkipPreflight bool `json:"skipPreflight,omitempty"`
	// UI-02 操作原因
	Reason string `json:"reason,omitempty"`
}

// PhysicalWrite 物理参数端口写 (POST /network/ports/write/physical)。
//
// OperType=Update(2)：修改现有端口物理配置（与 SetAccessVlan/description 同类）。
// 复用 execSinglePort 拿到 before_value / audit / operlog / response.Success 全套编排。
func (h *PortWriteHandler) PhysicalWrite(c *gin.Context) {
	var req PhysicalWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	params := portwrite.PhysicalParams{
		Speed:     req.Speed,
		SpeedMode: req.SpeedMode,
		Duplex:    req.Duplex,
		MTU:       req.MTU,
	}
	action := portcollection.PortAction(req.Action)
	h.execSinglePort(c, action, operlog.OperTypeUpdate, req.PortID, "",
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.PhysicalWrite(ctx, portID, action, params, operator, writeOptsFrom(req.SkipPreflight)...)
		})
}

// physicalSentinelToHTTP 把 FIX-04 的 service sentinel 追加进 execSinglePort 的翻译表。
//
// execSinglePort 是共用编排，新增动作的错误需要在其 switch 里翻译；为不动既有
// v1.19/v1.20.1 行为，这里通过独立函数导出 sentinel 判定供 execSinglePort 调用
// （见 port_write_handler.go execSinglePort 的 physical 分支）。
func physicalSentinelMessage(err error) (int, string, bool) {
	switch {
	case errors.Is(err, portwrite.ErrSpeedInvalid):
		return http.StatusBadRequest, "速率参数不合法（需 10/100/1000/10000 或 auto）", true
	case errors.Is(err, portwrite.ErrDuplexInvalid):
		return http.StatusBadRequest, "双工参数不合法（需 half/full/auto）", true
	case errors.Is(err, portwrite.ErrMTUOutOfRange):
		return http.StatusBadRequest, "MTU 必须在 1280-9216 之间", true
	case errors.Is(err, portwrite.ErrPhysicalParamsInvalid):
		return http.StatusBadRequest, "物理参数动作不合法", true
	}
	return 0, "", false
}
