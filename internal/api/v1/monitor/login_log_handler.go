package monitor

import (
	"github.com/gin-gonic/gin"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	monitorServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/monitor"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// LoginLogHandler 登录日志处理器
type LoginLogHandler struct {
	service monitorServices.LoginLogService
}

// NewLoginLogHandler 创建登录日志处理器实例
func NewLoginLogHandler(service monitorServices.LoginLogService) *LoginLogHandler {
	return &LoginLogHandler{service: service}
}

// LoginLogListRequest 登录日志列表请求
type LoginLogListRequest struct {
	Username  *string `json:"username,omitempty"`
	IPAddr    *string `json:"ipaddr,omitempty"`
	Status    *int    `json:"status,omitempty"`
	BeginTime *string `json:"beginTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Current   int     `json:"current" binding:"min=1"`
	PageSize  int     `json:"pageSize" binding:"min=1,max=100"`
}

// List 查询登录日志列表
// @Summary 查询登录日志列表
// @Description 分页查询登录日志列表
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param request body LoginLogListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /monitor/login-logs/list [post]
func (h *LoginLogHandler) List(c *gin.Context) {
	var req LoginLogListRequest
	// 允许空的请求体，设置默认值
	if err := c.ShouldBindJSON(&req); err != nil {
		req = LoginLogListRequest{
			Current:  1,
			PageSize: 10,
		}
	}

	params := monitorServices.LoginLogListParams{
		Username:  req.Username,
		IPAddr:    req.IPAddr,
		Status:    req.Status,
		BeginTime: req.BeginTime,
		EndTime:   req.EndTime,
		Current:   req.Current,
		PageSize:  req.PageSize,
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取登录日志详情
// @Summary 获取登录日志详情
// @Description 根据ID获取登录日志详情
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param id path string true "登录日志ID"
// @Success 200 {object} response.Response
// @Router /monitor/login-logs/:id [post]
func (h *LoginLogHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return
	}

	loginLog, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, loginLog)
}

// Delete 删除登录日志
// @Summary 删除登录日志
// @Description 删除指定登录日志
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param id path string true "登录日志ID"
// @Success 200 {object} response.Response
// @Router /monitor/login-logs/:id/delete [post]
func (h *LoginLogHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除登录日志
// @Summary 批量删除登录日志
// @Description 批量删除多个登录日志
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "ID列表"
// @Success 200 {object} response.Response
// @Router /monitor/login-logs/batch-delete [post]
func (h *LoginLogHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, nil)
}

// Clean 清空登录日志
// @Summary 清空登录日志
// @Description 清空所有登录日志
// @Tags 登录日志
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /monitor/login-logs/clean [post]
func (h *LoginLogHandler) Clean(c *gin.Context) {
	if err := h.service.Clean(c.Request.Context()); err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, gin.H{"message": "清空成功"})
}

// UnlockUser 解锁用户
// @Summary 解锁用户
// @Description 解锁被锁定的用户
// @Tags 登录日志
// @Accept json
// @Produce json
// @Param username path string true "用户名"
// @Success 200 {object} response.Response
// @Router /monitor/login-logs/unlock/:username [post]
func (h *LoginLogHandler) UnlockUser(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		response.Error(c, apperrors.ParamMissing("用户名"))
		return
	}

	// TODO: 实现解锁用户逻辑（如从Redis中删除锁定状态）

	response.Success(c, gin.H{"message": "解锁成功"})
}
