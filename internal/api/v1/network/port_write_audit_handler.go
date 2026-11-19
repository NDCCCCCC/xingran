package network

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PortWriteAuditList 端口写审计分页查询 (POST /network/ports/write/audit/list)。
//
// Phase 127 前置（PWX-04）：webfe 审计明细 UI 的数据源。纯读端点，
// 不写 audit/operlog。挂在既有 write 组下（组级 network:port:write 权限）。
//
// v1.57：返回值内嵌原审计行并附设备名称/IP、接口名/端口描述（快照优先），
// 前端不再展示裸 UUID。
func (h *PortWriteHandler) PortWriteAuditList(c *gin.Context) {
	var req portwrite.AuditListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	result, err := h.service.ListAudit(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// PortWriteAuditDeviceOptions 审计页设备选择器 (POST /network/ports/write/audit/device-options)。
//
// v1.57：审计页需要"按设备名称选设备"，但审计路由只挂了 network:port:write，
// 而 /network/devices/list 需要 network:device:* 读权限 —— 审计员会 403。
// 故在此提供只返回 id/名称/IP 的最小读面，权限沿用组级 network:port:write，
// 不扩大任何权限。纯读端点，不写 audit/operlog。
func (h *PortWriteHandler) PortWriteAuditDeviceOptions(c *gin.Context) {
	var req portwrite.AuditOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	result, err := h.service.ListDeviceOptions(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// PortWriteAuditPortOptions 审计页端口选择器 (POST /network/ports/write/audit/port-options)。
//
// v1.57：deviceId 必填（接口名跨设备重名，禁止全局裸选）；缺参返 400。
// 权限同 device-options（组级 network:port:write）。纯读端点。
func (h *PortWriteHandler) PortWriteAuditPortOptions(c *gin.Context) {
	var req portwrite.AuditOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	result, err := h.service.ListPortOptions(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, portwrite.ErrAuditPortOptionDeviceRequired) {
			response.Error(c, http.StatusBadRequest, "请先选择设备（deviceId 必填）")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}
