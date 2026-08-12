package system

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// AD域单分组同步超时时间
const adSingleGroupSyncTimeout = 2 * time.Minute

// adFullSyncTimeout AD域全量同步超时
const adFullSyncTimeout = 30 * time.Minute

// adGroupSyncTimeout AD域分组同步超时
const adGroupSyncTimeout = 10 * time.Minute

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
// @Summary 查询AD配置列表
// @Description 查询AD域配置列表，支持按状态筛选和分页
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{status=int,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs/list [post]
func (h *ADDomainHandler) ListConfigs(c *gin.Context) {
	var req struct {
		Status        *int    `json:"status,omitempty"`
		Current       int     `json:"current"`
		PageSize      int     `json:"pageSize"`
		OrderByColumn string  `json:"orderByColumn,omitempty"`
		IsAsc         *bool   `json:"isAsc,omitempty"`
	}
	_ = c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADConfigList(c.Request.Context(), req.Status, req.Current, req.PageSize, req.OrderByColumn, req.IsAsc)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// CreateConfig 创建AD配置
// @Summary 创建AD配置
// @Description 创建新的AD域配置
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configName=string,ldapServer=string,ldapPort=int,adminDN=string,adminPassword=string,useSSL=bool,useStartTLS=bool,baseDN=string,userSearchBase=string,userSearchFilter=string,groupSearchBase=string,groupSearchFilter=string,syncInterval=int,autoSync=bool,status=int} true "AD配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs [post]
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

	// D-04: 单管理员字段已废弃，响应清空（账号管理收敛到账号池 Tab）
	config.AdminUsername = ""
	config.AdminPassword = ""

	response.Success(c, config)
}

// GetConfig 获取AD配置详情
// @Summary 获取AD配置详情
// @Description 根据配置ID获取AD域配置详细信息
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs/:id [get]
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

	config.AdminUsername = ""
	config.AdminPassword = ""
	response.Success(c, config)
}

// UpdateConfig 更新AD配置
// @Summary 更新AD配置
// @Description 更新AD域配置信息
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body object{configName=string,ldapServer=string,ldapPort=int,adminDN=string,adminPassword=string,useSSL=bool,useStartTLS=bool,baseDN=string,userSearchBase=string,userSearchFilter=string,groupSearchBase=string,groupSearchFilter=string,syncInterval=int,autoSync=bool,status=int} true "AD配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs/:id/update [post]
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
// @Summary 删除AD配置
// @Description 删除指定的AD域配置
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs/:id/delete [post]
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
// @Summary 测试AD连接
// @Description 测试AD域服务器连接是否正常
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs/:id/test [post]
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
// @Summary 同步AD数据
// @Description 从AD域同步用户、用户组等数据到系统
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body object{syncType=string} true "同步类型(full/incremental)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs/:id/sync [post]
func (h *ADDomainHandler) SyncData(c *gin.Context) {
	id := c.Param("id")
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, err)
		return
	}

	var req struct {
		SyncType string `json:"syncType"`
	}
	_ = c.ShouldBindJSON(&req)

	if req.SyncType == "" {
		req.SyncType = "full"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), adFullSyncTimeout)
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
// @Summary 获取OU树形结构
// @Description 获取AD域的组织单元(OU)树形结构
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string} true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/ou/tree [post]
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
// @Summary 查询AD用户组列表
// @Description 查询AD域用户组列表，支持分页
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string,groupName=string,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/groups/list [post]
func (h *ADDomainHandler) ListGroups(c *gin.Context) {
	var req addomainServices.ADGroupListRequest
	_ = c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADGroupList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetGroupDetail 获取AD用户组详情
// @Summary 获取AD用户组详情
// @Description 根据用户组DN获取AD用户组详细信息
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户组DN"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/groups/:id [get]
func (h *ADDomainHandler) GetGroupDetail(c *gin.Context) {
	response.Error(c, apperrors.NotImplemented())
}

// UpdateGroup 更新AD用户组
// @Summary 更新AD用户组
// @Description 更新AD域用户组信息
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户组DN"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/groups/:id/update [post]
func (h *ADDomainHandler) UpdateGroup(c *gin.Context) {
	response.Error(c, apperrors.NotImplemented())
}

// GetGroupMembers 获取AD用户组成员
// @Summary 获取AD用户组成员
// @Description 获取AD域用户组的成员列表
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户组DN"
// @Param request body object{configId=string,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/groups/:id/members [post]
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
	_ = c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	members, total, err := h.service.GetADGroupMembers(c.Request.Context(), req.ConfigID, groupDN, req.Current, req.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, members, total, req.Current, req.PageSize)
}

// ListUsers 查询AD用户列表
// @Summary 查询AD用户列表
// @Description 查询AD域用户列表，支持分页
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string,username=string,email=string,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/users/list [post]
func (h *ADDomainHandler) ListUsers(c *gin.Context) {
	var req addomainServices.ADUserListRequest
	_ = c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADUserList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetUserDetail 获取AD用户详情
// @Summary 获取AD用户详情
// @Description 根据用户ID获取AD域用户详细信息
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/users/:id [get]
func (h *ADDomainHandler) GetUserDetail(c *gin.Context) {
	response.Error(c, apperrors.NotImplemented())
}

// UpdateUser 更新AD用户
// @Summary 更新AD用户
// @Description 更新AD域用户信息
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{configId=string,update=object{email=string,phone=string,displayName=string}} true "更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/users/:id/update [post]
func (h *ADDomainHandler) UpdateUser(c *gin.Context) {
	// 路由格式: /users/:id/update，其中:id是用户ID（数据库中的ID）
	userID := c.Param("id")
	if userID == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req struct {
		ConfigID string                               `json:"configId" binding:"required"`
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
// @Summary 移动AD用户到新OU
// @Description 将AD域用户移动到新的组织单元(OU)
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{configId=string,move=object{newOuDn=string}} true "移动信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/users/:id/move [post]
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
// @Summary 启用AD用户
// @Description 启用AD域用户账号
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{configId=string} true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/users/:id/enable [post]
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
// @Summary 禁用AD用户
// @Description 禁用AD域用户账号
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{configId=string} true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/users/:id/disable [post]
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
// @Summary 查询AD同步日志列表
// @Description 查询AD域数据同步日志列表
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/sync-logs/list [post]
func (h *ADDomainHandler) ListSyncLogs(c *gin.Context) {
	var req struct {
		ConfigID      string `json:"configId,omitempty"`
		Current       int    `json:"current"`
		PageSize      int    `json:"pageSize"`
		OrderByColumn string `json:"orderByColumn,omitempty"`
		IsAsc         *bool  `json:"isAsc,omitempty"`
	}
	_ = c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADSyncLogList(c.Request.Context(), req.ConfigID, req.Current, req.PageSize, req.OrderByColumn, req.IsAsc)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// ListComputers 查询AD电脑设备列表
// @Summary 查询AD电脑设备列表
// @Description 查询AD域电脑设备列表，支持分页
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string,computerName=string,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/computers/list [post]
func (h *ADDomainHandler) ListComputers(c *gin.Context) {
	var req addomainServices.ComputerListRequest
	_ = c.ShouldBindJSON(&req)

	h.setDefaultPagination(&req.Current, &req.PageSize)

	list, total, err := h.service.GetADComputerList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// GetComputerDetail 获取AD电脑设备详情
// @Summary 获取AD电脑设备详情
// @Description 根据电脑DN获取AD域电脑设备详细信息
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string,computerDn=string} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/computers/detail [post]
func (h *ADDomainHandler) GetComputerDetail(c *gin.Context) {
	var req struct {
		ConfigID   string `json:"configId" binding:"required"`
		ComputerDN string `json:"computerDn" binding:"required"`
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

// SyncGroups triggers group sync for a specific AD configuration
// @Summary 同步AD用户组
// @Description 从AD域同步用户组数据和成员关系
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/configs/:id/sync-groups [post]
func (h *ADDomainHandler) SyncGroups(c *gin.Context) {
	id := c.Param("id")
	if err := h.requireConfigID(id); err != nil {
		response.Error(c, err)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), adGroupSyncTimeout)
	defer cancel()

	result, err := h.service.GroupSync.SyncGroupsByConfig(ctx, id)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			response.Error(c, apperrors.New(apperrors.CodeServerError, "组同步超时，请稍后重试"))
			return
		}
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD用户组同步", OperTypeOther)
	response.Success(c, result)
}

// SyncSingleGroup triggers sync for a single group
// @Summary 同步单个AD用户组
// @Description 从AD域同步指定用户组及其成员关系
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string,groupDn=string} true "同步参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/groups/sync-single [post]
func (h *ADDomainHandler) SyncSingleGroup(c *gin.Context) {
	var req struct {
		ConfigID string `json:"configId" binding:"required"`
		GroupDN  string `json:"groupDn" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), adSingleGroupSyncTimeout)
	defer cancel()

	if err := h.service.GroupSync.SyncSingleGroup(ctx, req.ConfigID, req.GroupDN); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "AD单组同步", OperTypeOther)
	response.Success(c, gin.H{"message": "同步成功"})
}

// GetGroupSyncStatus returns sync status for groups under a config
// @Summary 获取用户组同步状态
// @Description 获取指定AD配置下的用户组同步状态统计
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string} true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/groups/sync-status [post]
func (h *ADDomainHandler) GetGroupSyncStatus(c *gin.Context) {
	var req struct {
		ConfigID string `json:"configId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	status, err := h.service.GroupSync.GetGroupSyncStatus(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, status)
}

// GetADUserIds 获取AD用户ID列表
// @Summary 获取AD用户ID列表
// @Description 根据筛选条件获取所有匹配的AD用户ID（用于全选功能）
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body object{configId=string,ouDn=string,username=string,isEnabled=bool} true "筛选条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/ad-domain/users/ids [post]
func (h *ADDomainHandler) GetADUserIds(c *gin.Context) {
	var req addomainServices.ADUserListRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.requireConfigID(req.ConfigID); err != nil {
		response.Error(c, err)
		return
	}

	userIds, err := h.service.GetADUserIds(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, userIds)
}
