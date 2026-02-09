package system

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
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

// setDefaultPagination 设置默认分页参数
func (h *ADDomainHandler) setDefaultPagination(current, pageSize *int) {
	if *current == 0 {
		*current = 1
	}
	if *pageSize == 0 {
		*pageSize = 10
	}
}

// requireConfigID 验证配置ID是否为空
func (h *ADDomainHandler) requireConfigID(id string) error {
	if id == "" {
		return apperrors.ParamMissing("配置ID")
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
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// CreateConfig 创建AD配置
func (h *ADDomainHandler) CreateConfig(c *gin.Context) {
	var req addomainServices.ADConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	creatorID := c.GetString("user_id")

	config, err := h.service.CreateADConfig(c.Request.Context(), &req, creatorID)
	if err != nil {
		response.Error(c, err)
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
		response.Error(c, err)
		return
	}

	config, err := h.service.GetADConfigByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.ADConfigNotFound())
		return
	}

	config.AdminPassword = ""
	response.Success(c, config)
}

// UpdateConfig 更新AD配置
func (h *ADDomainHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, err)
		return
	}

	var req addomainServices.ADConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	updaterID := c.GetString("user_id")

	if err := h.service.UpdateADConfig(c.Request.Context(), &req, updaterID); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD域配置", OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteConfig 删除AD配置
func (h *ADDomainHandler) DeleteConfig(c *gin.Context) {
	id := c.Param("id")
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, err)
		return
	}

	if err := h.service.DeleteADConfig(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD域配置", OperTypeDelete)
	response.Success(c, nil)
}

// TestConnection 测试AD连接
func (h *ADDomainHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, err)
		return
	}

	if err := h.service.TestADConnection(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"success": true})
}

// SyncData 同步AD数据
func (h *ADDomainHandler) SyncData(c *gin.Context) {
	id := c.Param("id")
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, err)
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
			response.Error(c, apperrors.New(apperrors.CodeServerError, "同步超时，请稍后重试"))
			return
		}
		response.Error(c, err)
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
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	tree, err := h.service.GetOUTree(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tree)
}

// ListGroups 查询AD用户组列表
func (h *ADDomainHandler) ListGroups(c *gin.Context) {
	var req addomainServices.ADGroupListRequest
	c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADGroupList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetGroupDetail 获取AD用户组详情
func (h *ADDomainHandler) GetGroupDetail(c *gin.Context) {
	response.Error(c, apperrors.NotImplemented())
}

// UpdateGroup 更新AD用户组
func (h *ADDomainHandler) UpdateGroup(c *gin.Context) {
	response.Error(c, apperrors.NotImplemented())
}

// GetGroupMembers 获取AD用户组成员
func (h *ADDomainHandler) GetGroupMembers(c *gin.Context) {
	groupDN := c.Param("id")
	if groupDN == "" {
		response.Error(c, apperrors.ParamMissing("用户组DN"))
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
		response.Error(c, err)
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
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetUserDetail 获取AD用户详情
func (h *ADDomainHandler) GetUserDetail(c *gin.Context) {
	response.Error(c, apperrors.NotImplemented())
}

// UpdateUser 更新AD用户
func (h *ADDomainHandler) UpdateUser(c *gin.Context) {
	// 路由格式: /users/:id/update，其中:id是用户ID（数据库中的ID）
	userID := c.Param("id")
	if userID == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req struct {
		ConfigID string                                     `json:"configId" binding:"required"`
		Update   addomainServices.ADUserUpdateRequest `json:"update"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	// 根据userID获取用户的DN
	user, err := h.service.GetADUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, apperrors.UserNotFound())
		return
	}

	config, err := h.service.GetADConfigByID(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, apperrors.ADConfigNotFound())
		return
	}

	if err := h.service.UpdateADUser(c.Request.Context(), config, user.UserDN, &req.Update); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD用户", OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// MoveUser 移动AD用户到新OU
func (h *ADDomainHandler) MoveUser(c *gin.Context) {
	// 路由格式: /users/:id/move，其中:id是用户ID
	userID := c.Param("id")
	if userID == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req struct {
		ConfigID string `json:"configId" binding:"required"`
		Move     struct {
			NewOUDN string `json:"newOuDn" binding:"required"`
		} `json:"move"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	// 根据userID获取用户的DN
	user, err := h.service.GetADUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, apperrors.UserNotFound())
		return
	}

	config, err := h.service.GetADConfigByID(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, apperrors.ADConfigNotFound())
		return
	}

	if err := h.service.MoveADUser(c.Request.Context(), config, user.UserDN, req.Move.NewOUDN); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD用户", OperTypeOther)
	response.Success(c, gin.H{"message": "移动成功"})
}

// EnableUser 启用AD用户
func (h *ADDomainHandler) EnableUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req struct {
		ConfigID string `json:"configId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	user, err := h.service.GetADUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, apperrors.UserNotFound())
		return
	}

	config, err := h.service.GetADConfigByID(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, apperrors.ADConfigNotFound())
		return
	}

	if err := h.service.EnableADUser(c.Request.Context(), config, user.UserDN); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD用户", OperTypeOther)
	response.Success(c, gin.H{"message": "启用成功"})
}

// DisableUser 禁用AD用户
func (h *ADDomainHandler) DisableUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req struct {
		ConfigID string `json:"configId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	user, err := h.service.GetADUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, apperrors.UserNotFound())
		return
	}

	config, err := h.service.GetADConfigByID(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, apperrors.ADConfigNotFound())
		return
	}

	if err := h.service.DisableADUser(c.Request.Context(), config, user.UserDN); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD用户", OperTypeOther)
	response.Success(c, gin.H{"message": "禁用成功"})
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
		response.Error(c, err)
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
		response.Error(c, err)
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
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	detail, err := h.service.GetADComputerByDN(c.Request.Context(), req.ConfigID, req.ComputerDN)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, detail)
}
