package system

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	addomainServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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

// setDefaultPagination 设置默认分页参数
func (h *ADDomainHandler) setDefaultPagination(current, pageSize *int) {
	if *current == 0 {
		*current = 1
	}
	if *pageSize == 0 {
		*pageSize = 10
	}
}

// requireConfigID 验证并返回配置ID
func (h *ADDomainHandler) requireConfigID(id string) error {
	if id == "" {
		return errors.New("AD配置ID不能为空")
	}
	return nil
}

// ListConfigs 查询AD配置列表
func (h *ADDomainHandler) ListConfigs(c *gin.Context) {
	var req struct {
		Status   *int `json:"status,omitempty"`
		Current  int  `json:"current"`
		PageSize int  `json:"pageSize"`
	}
	c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

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
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	config, err := h.service.GetADConfigByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "AD配置不存在")
		return
	}

	config.AdminPassword = ""
	response.Success(c, config)
}

// UpdateConfig 更新AD配置
func (h *ADDomainHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
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
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
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
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
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
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var req struct {
		SyncType string `json:"syncType"`
	}
	c.ShouldBindJSON(&req)

	if req.SyncType == "" {
		req.SyncType = "full"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	result, err := h.service.SyncADData(ctx, id, req.SyncType)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			response.Error(c, http.StatusRequestTimeout, "同步超时，请稍后重试或联系管理员")
			return
		}
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

	// 调试日志
	applogger.Infof("GetOUTree: configId=%s", req.ConfigID)

	tree, err := h.service.GetOUTree(c.Request.Context(), req.ConfigID)
	if err != nil {
		applogger.Errorf("GetOUTree failed: %v", err)
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	applogger.Infof("GetOUTree success: returned %d OUs", len(tree))
	response.Success(c, tree)
}

// ListGroups 查询AD用户组列表
func (h *ADDomainHandler) ListGroups(c *gin.Context) {
	var req addomainServices.ADGroupListRequest
	c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADGroupList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetGroupDetail 获取AD用户组详情
func (h *ADDomainHandler) GetGroupDetail(c *gin.Context) {
	response.Error(c, http.StatusNotImplemented, "功能开发中")
}

// UpdateGroup 更新AD用户组
func (h *ADDomainHandler) UpdateGroup(c *gin.Context) {
	response.Error(c, http.StatusNotImplemented, "功能暂未实现")
}

// GetGroupMembers 获取AD用户组成员
func (h *ADDomainHandler) GetGroupMembers(c *gin.Context) {
	groupDN := c.Param("id")
	if groupDN == "" {
		response.Error(c, http.StatusBadRequest, "用户组DN不能为空")
		return
	}

	var req struct {
		ConfigID string `json:"configId" binding:"required"`
		Current  int    `json:"current"`
		PageSize int    `json:"pageSize"`
	}
	c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	members, total, err := h.service.GetADGroupMembers(c.Request.Context(), req.ConfigID, groupDN, req.Current, req.PageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, members, total, req.Current, req.PageSize)
}

// ListUsers 查询AD用户列表
func (h *ADDomainHandler) ListUsers(c *gin.Context) {
	var req addomainServices.ADUserListRequest
	c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADUserList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetUserDetail 获取AD用户详情
func (h *ADDomainHandler) GetUserDetail(c *gin.Context) {
	response.Error(c, http.StatusNotImplemented, "功能开发中")
}

// ListSyncLogs 查询AD同步日志列表
func (h *ADDomainHandler) ListSyncLogs(c *gin.Context) {
	var req struct {
		ConfigID string `json:"configId,omitempty"`
		Current  int    `json:"current"`
		PageSize int    `json:"pageSize"`
	}
	c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADSyncLogList(c.Request.Context(), req.ConfigID, req.Current, req.PageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// ListComputers 查询AD电脑设备列表
func (h *ADDomainHandler) ListComputers(c *gin.Context) {
	var req addomainServices.ComputerListRequest
	c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADComputerList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetComputerDetail 获取AD电脑设备详情
func (h *ADDomainHandler) GetComputerDetail(c *gin.Context) {
	var req struct {
		ConfigID    string `json:"configId" binding:"required"`
		ComputerDN  string `json:"computerDn" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	detail, err := h.service.GetADComputerByDN(c.Request.Context(), req.ConfigID, req.ComputerDN)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, detail)
}
