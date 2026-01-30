package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// DiscoveryHandler 设备发现处理器
type DiscoveryHandler struct {
	discoveryService *services.DeviceDiscoveryService
}

// NewDiscoveryHandler 创建设备发现处理器实例
func NewDiscoveryHandler(discoveryService *services.DeviceDiscoveryService) *DiscoveryHandler {
	return &DiscoveryHandler{discoveryService: discoveryService}
}

// List 查询发现任务列表
// @Summary 查询发现任务列表
// @Description 分页查询设备发现任务列表
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int} true "分页参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/discoveries/list [post]
func (h *DiscoveryHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	current := getIntField(rawReq, "current", 1)
	pageSize := getIntField(rawReq, "pageSize", 10)

	tasks, total, err := h.discoveryService.GetDiscoveryList(c.Request.Context(), current, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	pageResp := response.PageResponse{
		List:     tasks,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}

	response.Success(c, pageResp)
}

// Create 创建发现任务
// @Summary 创建发现任务
// @Description 创建新的设备发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param request body services.DiscoveryRequest true "发现任务信息"
// @Success 200 {object} response.Response
// @Router /network/discoveries/create [post]
func (h *DiscoveryHandler) Create(c *gin.Context) {
	var req services.DiscoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认SNMP端口
	if req.SNMPPort == 0 {
		req.SNMPPort = 161
	}

	userID, _ := c.Get("user_id")
	req.CreatedBy = userID.(string)

	discoveryID, err := h.discoveryService.CreateDiscoveryTask(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"discoveryId": discoveryID})
}

// GetByID 获取发现任务详情
// @Summary 获取发现任务详情
// @Description 根据ID获取发现任务详情
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id [post]
func (h *DiscoveryHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "任务ID不能为空")
		return
	}

	task, err := h.discoveryService.GetDiscoveryByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, task)
}

// GetResults 获取发现结果
// @Summary 获取发现结果
// @Description 获取发现任务的设备发现结果
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/results [post]
func (h *DiscoveryHandler) GetResults(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "任务ID不能为空")
		return
	}

	devices, err := h.discoveryService.GetDiscoveryResults(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"devices": devices})
}

// Execute 执行发现任务
// @Summary 执行发现任务
// @Description 执行设备发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/execute [post]
func (h *DiscoveryHandler) Execute(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "任务ID不能为空")
		return
	}

	result, err := h.discoveryService.ExecuteDiscovery(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// Cancel 取消发现任务
// @Summary 取消发现任务
// @Description 取消正在执行的发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/cancel [post]
func (h *DiscoveryHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "任务ID不能为空")
		return
	}

	if err := h.discoveryService.CancelDiscovery(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "取消成功"})
}

// Delete 删除发现任务
// @Summary 删除发现任务
// @Description 删除指定的发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/delete [post]
func (h *DiscoveryHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "任务ID不能为空")
		return
	}

	if err := h.discoveryService.DeleteDiscovery(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除发现任务
// @Summary 批量删除发现任务
// @Description 批量删除多个发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param request body object{discoveryIds=[]string} true "任务ID列表"
// @Success 200 {object} response.Response
// @Router /network/discoveries/batch-delete [post]
func (h *DiscoveryHandler) BatchDelete(c *gin.Context) {
	var req struct {
		DiscoveryIDs []string `json:"discoveryIds" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.discoveryService.BatchDeleteDiscoveries(c.Request.Context(), req.DiscoveryIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.DiscoveryIDs),
	})
}

// ImportDevices 导入发现的设备
// @Summary 导入发现的设备
// @Description 将发现的设备导入系统
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/import [post]
func (h *DiscoveryHandler) ImportDevices(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "任务ID不能为空")
		return
	}

	userID, _ := c.Get("user_id")
	count, err := h.discoveryService.ImportDiscoveredDevices(c.Request.Context(), id, nil, userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"count": count})
}
