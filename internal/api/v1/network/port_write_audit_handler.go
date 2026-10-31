package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PortWriteAuditList 端口写审计分页查询 (POST /network/ports/write/audit/list)。
//
// Phase 127 前置（PWX-04）：webfe 审计明细 UI 的数据源。纯读端点，
// 不写 audit/operlog。挂在既有 write 组下（组级 network:port:write 权限）。
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
