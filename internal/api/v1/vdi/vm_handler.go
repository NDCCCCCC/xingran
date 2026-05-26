package vdi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	vdiServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/vdi"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// VMHandler 虚拟机处理器
type VMHandler struct {
	vmService vdiServices.VMService
}

// NewVMHandler 创建虚拟机处理器
func NewVMHandler(vmService vdiServices.VMService) *VMHandler {
	return &VMHandler{vmService: vmService}
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

	result, err := h.vmService.ListVMs(c.Request.Context(), &req)
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

	response.Success(c, nil)
}

// ConfigIP 批量配置虚拟机IP
// @Summary 配置虚拟机IP
// @Description 批量设置虚拟机IP地址，调用VDI API
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param request body []vdiServices.VMIPConfigRequest true "IP配置"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/config_ip [post]
func (h *VMHandler) ConfigIP(c *gin.Context) {
	var req []vdiServices.VMIPConfigRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.vmService.BatchConfigIP(c.Request.Context(), req), "配置IP") {
		return
	}

	response.Success(c, nil)
}

// Rename 重命名虚拟机
// @Summary 重命名虚拟机
// @Description 修改虚拟机名称，调用VDI API
// @Tags VDI管理
// @Accept json
// @Produce json
// @Param id path string true "虚拟机ID"
// @Param request body vdiServices.RenameVMServiceRequest true "新名称"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /vdi/vm/{id}/rename [post]
func (h *VMHandler) Rename(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "虚拟机ID不能为空")
		return
	}

	var req vdiServices.RenameVMServiceRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.vmService.RenameVM(c.Request.Context(), id, &req), "重命名") {
		return
	}

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

	response.Success(c, nil)
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

	response.Success(c, map[string]interface{}{
		"message": "同步任务已提交",
		"server_id": serverID,
	})
}
