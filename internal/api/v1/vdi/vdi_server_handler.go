package vdi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	vdiServices "github.com/xingran-next/xingran-go-backend/internal/services/vdi"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// VDIServerHandler VDI服务器处理器
type VDIServerHandler struct {
	serverService vdiServices.VDIServerService
	core          *core.Core
}

// NewVDIServerHandler 创建VDI服务器处理器
func NewVDIServerHandler(serverService vdiServices.VDIServerService) *VDIServerHandler {
	return &VDIServerHandler{serverService: serverService}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *VDIServerHandler) WithCore(core *core.Core) *VDIServerHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// Create 创建VDI服务器
// @Summary 创建VDI服务器
// @Description 创建新的VDI服务器配置
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body vdiServices.CreateVDIServerRequest true "VDI服务器信息"
// @Success 200 {object} response.Response{data=vdiServices.VDIServerDTO}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/servers [post]
func (h *VDIServerHandler) Create(c *gin.Context) {
	var req vdiServices.CreateVDIServerRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	server, err := h.serverService.CreateServer(c.Request.Context(), &req)
	if !handleServiceError(c, err, "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "VDI服务器", operlog.OperTypeCreate)
	response.Success(c, server)
}

// List 查询VDI服务器列表
// @Summary 查询VDI服务器列表
// @Description 分页查询VDI服务器列表
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body object{page=int,pageSize=int} true "查询参数"
// @Success 200 {object} response.Response{data=vdiServices.VDIServerPageResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/servers/list [post]
func (h *VDIServerHandler) List(c *gin.Context) {
	var params struct {
		Page          int    `json:"page"`
		PageSize      int    `json:"pageSize"`
		OrderByColumn string `json:"orderByColumn,omitempty"`
		IsAsc         *bool  `json:"isAsc,omitempty"`
	}
	if !handleJSONBinding(c, &params) {
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	result, err := h.serverService.ListServers(c.Request.Context(), params.Page, params.PageSize, params.OrderByColumn, params.IsAsc)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}

// GetByID 获取VDI服务器详情
// @Summary 获取VDI服务器详情
// @Description 根据ID获取VDI服务器详细信息
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "VDI服务器ID"
// @Success 200 {object} response.Response{data=vdiServices.VDIServerDTO}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /vdi/servers/{id} [post]
func (h *VDIServerHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "VDI服务器ID不能为空")
		return
	}

	server, err := h.serverService.GetServer(c.Request.Context(), id)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, server)
}

// Update 更新VDI服务器
// @Summary 更新VDI服务器
// @Description 更新VDI服务器配置
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "VDI服务器ID"
// @Param request body vdiServices.UpdateVDIServerRequest true "VDI服务器信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/servers/{id}/update [post]
func (h *VDIServerHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "VDI服务器ID不能为空")
		return
	}

	var req vdiServices.UpdateVDIServerRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.serverService.UpdateServer(c.Request.Context(), id, &req), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "VDI服务器", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除VDI服务器
// @Summary 删除VDI服务器
// @Description 删除指定的VDI服务器配置
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "VDI服务器ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/servers/{id}/delete [post]
func (h *VDIServerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "VDI服务器ID不能为空")
		return
	}

	if !handleServiceError(c, h.serverService.DeleteServer(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "VDI服务器", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// TestConnection 测试VDI服务器连接
// @Summary 测试连接
// @Description 测试VDI服务器连接是否正常
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "VDI服务器ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/servers/{id}/test [post]
func (h *VDIServerHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "VDI服务器ID不能为空")
		return
	}

	if !handleServiceError(c, h.serverService.TestConnection(c.Request.Context(), id), "测试连接") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "VDI服务器", operlog.OperTypeOther)
	response.Success(c, nil)
}
