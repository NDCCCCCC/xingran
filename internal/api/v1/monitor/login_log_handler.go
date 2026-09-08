package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// LoginLogHandler 登录日志处理器 — embeds MonitorLogHandler for Delete/BatchDelete.
type LoginLogHandler struct {
	*MonitorLogHandler[models.LoginLog] // shared Delete/BatchDelete
	svc                               monitorServices.LoginLogService
}

// NewLoginLogHandler creates a LoginLogHandler.
func NewLoginLogHandler(svc monitorServices.LoginLogService) *LoginLogHandler {
	h := &LoginLogHandler{svc: svc}
	h.MonitorLogHandler = NewMonitorLogHandler[models.LoginLog](
		svc, // logMutator
		"登录日志",
		operlog.OperTypeDelete,
		operlog.OperTypeBatch,
		true, // needOperlog=true: LoginLog delete calls operlog
	)
	return h
}

// WithCore injects core dependency.
func (h *LoginLogHandler) WithCore(core *core.Core) *LoginLogHandler {
	h.MonitorLogHandler.WithCore(core)
	return h
}

// LoginLogListRequest 登录日志列表请求
type LoginLogListRequest struct {
	base.BaseListRequest
	Username  *string `json:"username,omitempty"`
	IPAddr   *string `json:"ipaddr,omitempty"`
	Status   *int    `json:"status,omitempty"`
	BeginTime *string `json:"beginTime,omitempty"`
	EndTime  *string `json:"endTime,omitempty"`
}

// List 查询登录日志列表
func (h *LoginLogHandler) List(c *gin.Context) {
	var req LoginLogListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = LoginLogListRequest{
			BaseListRequest: base.BaseListRequest{
				Current:  constants.DefaultCurrent,
				PageSize: constants.DefaultPageSize,
			},
		}
	}

	params := monitorServices.LoginLogListParams{
		BaseListRequest: req.BaseListRequest,
		Username:       req.Username,
		IPAddr:         req.IPAddr,
		Status:         req.Status,
		BeginTime:      req.BeginTime,
		EndTime:        req.EndTime,
	}

	result, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取登录日志详情
func (h *LoginLogHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return
	}

	loginLog, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, loginLog)
}

// Clean 清空登录日志
func (h *LoginLogHandler) Clean(c *gin.Context) {
	if err := h.svc.Clean(c.Request.Context()); err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "登录日志", operlog.OperTypeClean)

	response.Success(c, gin.H{"message": "清空成功"})
}

// UnlockUser 解锁用户 — Phase 107 TODO-03 placeholder, not part of handler deduplication
func (h *LoginLogHandler) UnlockUser(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		response.Error(c, apperrors.ParamMissing("用户名"))
		return
	}

	// TODO: 实现解锁用户逻辑（如从Redis中删除锁定状态）

	response.Success(c, gin.H{"message": "解锁成功"})
}
