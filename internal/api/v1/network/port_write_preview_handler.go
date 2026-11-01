package network

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PortWritePreview 端口写 dry-run 命令预览 (POST /network/ports/write/preview)。
//
// Phase 121 (v1.51 PORTW-01)。dry-run 不下发任何设备命令（纯模板渲染层），
// 因此：不写 audit 行、不调 operlog.Record、不触发写后端口采集。
//
// sentinel error 翻译与 execSinglePort / BatchWrite 共享同一语义表（4xx 不写审计）。
// 非 sentinel error → 500。
func (h *PortWriteHandler) PortWritePreview(c *gin.Context) {
	var req portwrite.PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	req.DryRun = true

	result, err := h.service.PreviewPortWrite(c.Request.Context(), req, utils.GetUsername(c))
	if err != nil {
		switch {
		case errors.Is(err, portwrite.ErrPortNotFound):
			response.Error(c, http.StatusNotFound, "端口不存在")
		case errors.Is(err, portwrite.ErrDeviceNotFound):
			response.Error(c, http.StatusNotFound, "设备不存在")
		case errors.Is(err, portwrite.ErrVlanIdOutOfRange):
			response.Error(c, http.StatusBadRequest, "VLAN ID 必须在 1-4094 之间")
		case errors.Is(err, portwrite.ErrBindOpInvalid):
			response.Error(c, http.StatusBadRequest, "绑定操作必须是 add 或 remove")
		case errors.Is(err, portwrite.ErrIPAddressInvalid):
			response.Error(c, http.StatusBadRequest, "IP 地址格式不合法")
		case errors.Is(err, portwrite.ErrMACAddressInvalid):
			response.Error(c, http.StatusBadRequest, "MAC 地址格式不合法")
		default:
			// 渲染层错误（不支持厂商 / 未知 action / 描述为空或超长）同为入参问题 → 400
			response.Error(c, http.StatusBadRequest, err.Error())
		}
		return
	}
	response.Success(c, result)
}
