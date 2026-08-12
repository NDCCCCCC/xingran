package vdi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	vdiServices "github.com/xingran-next/xingran-go-backend/internal/services/vdi"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// VMHandler 虚拟机处理器
type VMHandler struct {
	vmService vdiServices.VMService
	db        *gorm.DB
	core      *core.Core
}

// NewVMHandler 创建虚拟机处理器
func NewVMHandler(vmService vdiServices.VMService, db *gorm.DB) *VMHandler {
	return &VMHandler{vmService: vmService, db: db}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *VMHandler) WithCore(core *core.Core) *VMHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// Create 创建虚拟机
// @Summary 创建虚拟机
// @Description 创建新的虚拟机，调用VDI API
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body vdiServices.CreateVMServiceRequest true "虚拟机信息"
// @Success 200 {object} response.Response{data=vdiServices.VDIVMDTO}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm [post]
func (h *VMHandler) Create(c *gin.Context) {
	var req vdiServices.CreateVMServiceRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	vm, err := h.vmService.CreateVM(c.Request.Context(), &req)
	if !handleServiceError(c, err, "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeCreate)
	response.Success(c, vm)
}

// List 查询虚拟机列表
// @Summary 查询虚拟机列表
// @Description 分页查询虚拟机列表，支持按条件筛选
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body vdiServices.ListVMRequest true "查询参数"
// @Success 200 {object} response.Response{data=vdiServices.PageResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/list [post]
func (h *VMHandler) List(c *gin.Context) {
	var req vdiServices.ListVMRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// Extract data scope info from Gin context (set by DataScopePermission middleware)
	var userID string
	var dataScope models.DataScope
	if uid, exists := c.Get("user_id"); exists {
		if uidStr, ok := uid.(string); ok {
			userID = uidStr
		}
	}
	if ds, exists := c.Get("data_scope"); exists {
		if dsVal, ok := ds.(models.DataScope); ok {
			dataScope = dsVal
		}
	}

	result, err := h.vmService.ListVMs(c.Request.Context(), &req, userID, dataScope)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}

// GetByID 获取虚拟机详情
// @Summary 获取虚拟机详情
// @Description 根据ID获取虚拟机详细信息
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "虚拟机ID"
// @Success 200 {object} response.Response{data=vdiServices.VDIVMDTO}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /vdi/vm/{id} [post]
func (h *VMHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "虚拟机ID不能为空")
		return
	}

	vm, err := h.vmService.GetVM(c.Request.Context(), id)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, vm)
}

// Update 更新虚拟机
// @Summary 更新虚拟机
// @Description 更新虚拟机信息
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "虚拟机ID"
// @Param request body vdiServices.UpdateVMRequest true "虚拟机信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/{id}/update [post]
func (h *VMHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "虚拟机ID不能为空")
		return
	}

	var req vdiServices.UpdateVMRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.vmService.UpdateVM(c.Request.Context(), id, &req), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除虚拟机
// @Summary 删除虚拟机
// @Description 删除指定的虚拟机，调用VDI API
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "虚拟机ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/{id}/delete [post]
func (h *VMHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "虚拟机ID不能为空")
		return
	}

	if !handleServiceError(c, h.vmService.DeleteVM(c.Request.Context(), []string{id}), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// Operate 操作虚拟机
// @Summary 操作虚拟机
// @Description 批量开机、关机、重启、休眠虚拟机，调用VDI API
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body vdiServices.VMOperateRequest true "操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/operate [post]
func (h *VMHandler) Operate(c *gin.Context) {
	var req vdiServices.VMOperateRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.vmService.OperateVM(c.Request.Context(), &req), "操作") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeStatus)
	response.Success(c, nil)
}
// StartVM 启动虚拟机
// @Summary 启动虚拟机
// @Description 批量启动指定的虚拟机
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body vdiServices.VMOperateRequest true "虚拟机ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vms/start [post]
func (h *VMHandler) StartVM(c *gin.Context) {
	var req vdiServices.VMOperateRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	// 固定动作为 start
	req.Action = vdiServices.VMPowerOn

	if !handleServiceError(c, h.vmService.OperateVM(c.Request.Context(), &req), "启动虚拟机") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeStatus)
	response.Success(c, nil)
}

// StopVM 关闭虚拟机
// @Summary 关闭虚拟机
// @Description 批量关闭指定的虚拟机
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body vdiServices.VMOperateRequest true "虚拟机ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vms/stop [post]
func (h *VMHandler) StopVM(c *gin.Context) {
	var req vdiServices.VMOperateRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	// 固定动作为 stop
	req.Action = vdiServices.VMPowerOff

	if !handleServiceError(c, h.vmService.OperateVM(c.Request.Context(), &req), "关闭虚拟机") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeStatus)
	response.Success(c, nil)
}

// RestartVM 重启虚拟机
// @Summary 重启虚拟机
// @Description 批量重启指定的虚拟机
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body vdiServices.VMOperateRequest true "虚拟机ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vms/restart [post]
func (h *VMHandler) RestartVM(c *gin.Context) {
	var req vdiServices.VMOperateRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	// 固定动作为 restart
	req.Action = vdiServices.VMPowerRestart

	if !handleServiceError(c, h.vmService.OperateVM(c.Request.Context(), &req), "重启虚拟机") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeStatus)
	response.Success(c, nil)
}


// BindUser 绑定用户到虚拟机
// @Summary 绑定用户
// @Description 将虚拟机关联到指定用户，调用VDI API
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "虚拟机ID"
// @Param request body vdiServices.BindUserServiceRequest true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/{id}/bind_user [post]
func (h *VMHandler) BindUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "虚拟机ID不能为空")
		return
	}

	var req vdiServices.BindUserServiceRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.vmService.BindUser(c.Request.Context(), id, &req), "绑定用户") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeGrant)
	response.Success(c, nil)
}

// UnbindUser 解绑用户
// @Summary 解绑用户
// @Description 解除虚拟机与用户的关联
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "虚拟机ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/{id}/unbind_user [post]
func (h *VMHandler) UnbindUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "虚拟机ID不能为空")
		return
	}

	if !handleServiceError(c, h.vmService.UnbindUser(c.Request.Context(), id), "解绑用户") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeStatus)
	response.Success(c, nil)
}

// SyncFromVDI 从VDI同步虚拟机状态
// @Summary 同步虚拟机状态
// @Description 从VDI服务器同步虚拟机最新状态
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "虚拟机ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/{id}/sync [post]
func (h *VMHandler) SyncFromVDI(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "虚拟机ID不能为空")
		return
	}

	if !handleServiceError(c, h.vmService.SyncVMFromVDI(c.Request.Context(), id), "同步") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeSync)
	response.Success(c, nil)
}

// ListResourceGroups 查询资源组列表
// @Summary 查询资源组列表
// @Description 查询VDI资源组列表，支持按VDI服务器过滤
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} response.Response{data=[]vdiServices.VDIResourceGroupDTO}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/resource-groups [post]
func (h *VMHandler) ListResourceGroups(c *gin.Context) {
	var req struct {
		VdiServerID string `json:"vdi_server_id"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	groups, err := h.vmService.ListResourceGroups(c.Request.Context(), req.VdiServerID)
	if !handleServiceError(c, err, "查询资源组") {
		return
	}

	response.Success(c, groups)
}

// ListResources 查询资源列表
// @Summary 查询资源列表
// @Description 查询指定资源组下的计算资源列表
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} response.Response{data=[]vdiServices.VDIResourceDTO}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/resources [post]
func (h *VMHandler) ListResources(c *gin.Context) {
	var req struct {
		VdiServerID string `json:"vdi_server_id"`
		GroupID     string `json:"group_id"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	if req.GroupID == "" {
		response.Error(c, http.StatusBadRequest, "资源组ID不能为空")
		return
	}

	resources, err := h.vmService.ListResources(c.Request.Context(), req.VdiServerID, req.GroupID)
	if !handleServiceError(c, err, "查询资源") {
		return
	}

	response.Success(c, resources)
}

// SyncAll 同步所有虚拟机数据
// @Summary 同步所有虚拟机
// @Description 从VDI服务器同步所有虚拟机数据到本地数据库
// @Tags VDI管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/sync-all [post]
func (h *VMHandler) SyncAll(c *gin.Context) {
	// 获取第一个启用的VDI服务器ID
	var req struct {
		ServerID string `json:"server_id"`
	}
	_ = c.ShouldBindJSON(&req)

	// 如果没有指定服务器ID，使用默认的（服务层会自动查找第一个启用的）
	serverID := req.ServerID
	if serverID == "" {
		serverID = "auto"
	}

	if !handleServiceError(c, h.vmService.SyncAllVMs(c.Request.Context(), serverID), "同步") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeSync)
	response.Success(c, map[string]interface{}{
		"message": "同步任务已提交",
		"server_id": serverID,
	})
}

// ListVTPPlatforms 查询VTP平台列表
// @Summary 查询VTP平台列表
// @Description 查询VDI服务器的VTP平台列表，用于虚拟机创建
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} response.Response{data=[]vdi.VDIPlatform}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/vtp-platforms [post]
func (h *VMHandler) ListVTPPlatforms(c *gin.Context) {
	var req struct {
		VdiServerID string `json:"vdi_server_id"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	if !ensureVDIServer(c, h.db, req.VdiServerID) {
		return
	}

	// 获取VDI客户端
	client := vdiServices.NewVDIClientFromDB(h.db, req.VdiServerID)

	platforms, err := client.GetVTPPlatforms(c.Request.Context())
	if !handleServiceError(c, err, "查询VTP平台") {
		return
	}

	response.Success(c, platforms)
}

// ListRunPositions 查询运行位置列表
// @Summary 查询运行位置列表
// @Description 查询指定VTP平台下的运行位置列表
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} response.Response{data=[]vdi.RunPosition}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/run-positions [post]
func (h *VMHandler) ListRunPositions(c *gin.Context) {
	var req struct {
		VdiServerID string `json:"vdi_server_id"`
		VTPID       int    `json:"vtp_id"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	if !ensureVDIServer(c, h.db, req.VdiServerID) {
		return
	}
	if req.VTPID <= 0 {
		response.Error(c, http.StatusBadRequest, "VTP平台ID不能为空")
		return
	}

	// 获取VDI客户端
	client := vdiServices.NewVDIClientFromDB(h.db, req.VdiServerID)

	positions, err := client.GetRunPositions(c.Request.Context(), req.VTPID)
	if !handleServiceError(c, err, "查询运行位置") {
		return
	}

	response.Success(c, positions)
}

// ListStorages 查询存储位置列表
// @Summary 查询存储位置列表
// @Description 查询指定VTP平台下的存储位置列表
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} response.Response{data=[]vdi.Storage}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/storages [post]
func (h *VMHandler) ListStorages(c *gin.Context) {
	var req struct {
		VdiServerID string `json:"vdi_server_id"`
		VTPID       int    `json:"vtp_id"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	if !ensureVDIServer(c, h.db, req.VdiServerID) {
		return
	}
	if req.VTPID <= 0 {
		response.Error(c, http.StatusBadRequest, "VTP平台ID不能为空")
		return
	}

	// 获取VDI客户端
	client := vdiServices.NewVDIClientFromDB(h.db, req.VdiServerID)

	storages, err := client.GetStorages(c.Request.Context(), req.VTPID)
	if !handleServiceError(c, err, "查询存储位置") {
		return
	}

	response.Success(c, storages)
}

// ListNetworks 查询网络接口列表
// @Summary 查询网络接口列表
// @Description 查询指定VTP平台下的网络接口列表
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} response.Response{data=[]vdi.Network}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/networks [post]
func (h *VMHandler) ListNetworks(c *gin.Context) {
	var req struct {
		VdiServerID string `json:"vdi_server_id"`
		VTPID       int    `json:"vtp_id"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	if !ensureVDIServer(c, h.db, req.VdiServerID) {
		return
	}
	if req.VTPID <= 0 {
		response.Error(c, http.StatusBadRequest, "VTP平台ID不能为空")
		return
	}

	// 获取VDI客户端
	client := vdiServices.NewVDIClientFromDB(h.db, req.VdiServerID)

	networks, err := client.GetNetworks(c.Request.Context(), req.VTPID)
	if !handleServiceError(c, err, "查询网络接口") {
		return
	}

	response.Success(c, networks)
}
