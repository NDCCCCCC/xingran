package system

import (
	"github.com/gin-gonic/gin"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// ADLoginHistoryHandler AD 设备登录历史处理器(quick 261016-ad-login-history)
// 只读端点,不记录 operlog(operlog 仅写操作)。
type ADLoginHistoryHandler struct {
	svc addomainServices.ADLoginHistoryService
}

// NewADLoginHistoryHandler 创建登录历史处理器(内部构造 service)
func NewADLoginHistoryHandler(db *gorm.DB) *ADLoginHistoryHandler {
	return &ADLoginHistoryHandler{
		svc: addomainServices.NewADLoginHistoryService(db),
	}
}

// ListComputerLoginHistory 查询AD设备登录历史
// @Summary 查询AD设备登录历史
// @Description 按设备ID查询登录历史(后端固定 login_time 倒序,服务端分页)
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body addomainServices.ADLoginHistoryQuery true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/computers/login-history [post]
func (h *ADLoginHistoryHandler) ListComputerLoginHistory(c *gin.Context) {
	var req addomainServices.ADLoginHistoryQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	list, total, err := h.svc.ListComputerLoginHistory(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}
