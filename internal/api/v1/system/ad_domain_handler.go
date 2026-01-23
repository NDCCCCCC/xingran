package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	addomainServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ADDomainHandler AD域管理处理器
type ADDomainHandler struct {
	service *addomainServices.ADDomainService
	core    *core.Core
}

// NewADDomainHandler 创建AD域管理处理器实例
func NewADDomainHandler(service *addomainServices.ADDomainService, core *core.Core) *ADDomainHandler {
	return &ADDomainHandler{
		service: service,
		core:    core,
	}
}

// ListConfigs 查询AD配置列表
func (h *ADDomainHandler) ListConfigs(c *gin.Context) {
	var req struct {
		Status   *int `json:"status,omitempty"`
		Current  int  `json:"current"`
		PageSize int  `json:"pageSize"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req = struct {
			Status   *int `json:"status,omitempty"`
			Current  int  `json:"current"`
			PageSize int  `json:"pageSize"`
		}{}
	}

	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	list, total, err := h.service.GetADConfigList(c.Request.Context(), req.Status, req.Current, req.PageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// CreateConfig 创建AD配置
func (h *ADDomainHandler) CreateConfig(c *gin.Context) {
	var req addomainServices.ADConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	creatorID := c.GetString("user_id")

	config, err := h.service.CreateADConfig(c.Request.Context(), &req, creatorID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "AD域配置", OperTypeCreate)

	// 隐藏密码
	config.AdminPassword = ""

	response.Success(c, config)
}

// GetConfig 获取AD配置详情
func (h *ADDomainHandler) GetConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "AD配置ID不能为空")
		return
	}

	config, err := h.service.GetADConfigByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "AD配置不存在")
		return
	}

	// 隐藏密码
	config.AdminPassword = ""

	response.Success(c, config)
}

// UpdateConfig 更新AD配置
func (h *ADDomainHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "AD配置ID不能为空")
		return
	}

	var req addomainServices.ADConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	req.ID = id
	updaterID := c.GetString("user_id")

	if err := h.service.UpdateADConfig(c.Request.Context(), &req, updaterID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "AD域配置", OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteConfig 删除AD配置
func (h *ADDomainHandler) DeleteConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "AD配置ID不能为空")
		return
	}

	if err := h.service.DeleteADConfig(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "AD域配置", OperTypeDelete)
	response.Success(c, nil)
}

// TestConnection 测试AD连接
func (h *ADDomainHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "AD配置ID不能为空")
		return
	}

	err := h.service.TestADConnection(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
		"message": "连接成功",
	})
}

// SyncData 同步AD数据
func (h *ADDomainHandler) SyncData(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "AD配置ID不能为空")
		return
	}

	var req struct {
		SyncType string `json:"syncType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SyncType = "full"
	}

	if req.SyncType == "" {
		req.SyncType = "full"
	}

	result, err := h.service.SyncADData(c.Request.Context(), id, req.SyncType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "AD域配置", OperTypeOther)
	response.Success(c, result)
}

// GetOUTree 获取OU树形结构
func (h *ADDomainHandler) GetOUTree(c *gin.Context) {
	var req struct {
		ConfigID string `json:"configId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	tree, err := h.service.GetOUTree(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, tree)
}

// ListGroups 查询AD用户组列表
func (h *ADDomainHandler) ListGroups(c *gin.Context) {
	var req addomainServices.ADGroupListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = addomainServices.ADGroupListRequest{}
	}

	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	list, total, err := h.service.GetADGroupList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetGroupDetail 获取AD用户组详情
func (h *ADDomainHandler) GetGroupDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户组ID不能为空")
		return
	}

	response.Error(c, http.StatusNotImplemented, "功能开发中")
}

// UpdateGroup 更新AD用户组
func (h *ADDomainHandler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户组ID不能为空")
		return
	}

	response.Error(c, http.StatusNotImplemented, "功能暂未实现")
}

// ListUsers 查询AD用户列表
func (h *ADDomainHandler) ListUsers(c *gin.Context) {
	var req addomainServices.ADUserListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = addomainServices.ADUserListRequest{}
	}

	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	list, total, err := h.service.GetADUserList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetUserDetail 获取AD用户详情
func (h *ADDomainHandler) GetUserDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	response.Error(c, http.StatusNotImplemented, "功能开发中")
}

// ListSyncLogs 查询AD同步日志列表
func (h *ADDomainHandler) ListSyncLogs(c *gin.Context) {
	var req struct {
		ConfigID string `json:"configId,omitempty"`
		Current  int    `json:"current"`
		PageSize int    `json:"pageSize"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req = struct {
			ConfigID string `json:"configId,omitempty"`
			Current  int    `json:"current"`
			PageSize int    `json:"pageSize"`
		}{}
	}

	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	list, total, err := h.service.GetADSyncLogList(c.Request.Context(), req.ConfigID, req.Current, req.PageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}
