package system

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ManagerRuleHandler 经理同步规则维护（列表/CRUD）。
// 服务层校验与翻译见 addomain.ManagerRuleService。
type ManagerRuleHandler struct {
	core *core.Core
	svc  *addomainServices.ManagerRuleService
}

// NewManagerRuleHandler 创建规则处理器。
func NewManagerRuleHandler(core *core.Core, svc *addomainServices.ManagerRuleService) *ManagerRuleHandler {
	return &ManagerRuleHandler{core: core, svc: svc}
}

// ManagerOrgHandler 经理同步机构维护（列表/CRUD）。
type ManagerOrgHandler struct {
	core *core.Core
	svc  *addomainServices.ManagerOrgService
}

// NewManagerOrgHandler 创建机构处理器。
func NewManagerOrgHandler(core *core.Core, svc *addomainServices.ManagerOrgService) *ManagerOrgHandler {
	return &ManagerOrgHandler{core: core, svc: svc}
}

// parsePageParams 解析分页参数（current 从 1 开始；越界回落默认）。
func parsePageParams(c *gin.Context) (current, pageSize int) {
	current, _ = strconv.Atoi(c.DefaultPostForm("current", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultPostForm("pageSize", "10"))
	if current < 1 {
		current = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return current, pageSize
}

// ==================== 规则 ====================

// ListRules 规则分页列表
// POST /system/manager-rules/list
func (h *ManagerRuleHandler) ListRules(c *gin.Context) {
	var req requests.ManagerRuleListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = requests.DefaultManagerRuleListRequest()
	}
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	page, err := h.svc.ListRules(c.Request.Context(), req.Current, req.PageSize, req.Keyword, req.Enabled)
	if err != nil {
		response.Error(c, apperrors.ServerError(err))
		return
	}
	response.Success(c, page)
}

// ManagerPreviewRequest 经理预览请求体。
type ManagerPreviewRequest struct {
	UserIDs []string `json:"userIds" binding:"required,min=1,max=200"`
}

// ManagerPreview 批量预览用户的经理（用户列表「经理」列展示用，纯计算不落库）
// POST /system/manager-rules/manager-preview
func (h *ManagerRuleHandler) ManagerPreview(c *gin.Context) {
	var req ManagerPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.ParamInvalid("userIds"))
		return
	}
	items, err := h.svc.PreviewManagers(c.Request.Context(), req.UserIDs)
	if err != nil {
		response.Error(c, apperrors.ServerError(err))
		return
	}
	// 查询类端点，不记 operlog；失败由前端降级为不显示经理列。
	response.Success(c, items)
}

// CreateRule 新增规则
// POST /system/manager-rules
func (h *ManagerRuleHandler) CreateRule(c *gin.Context) {
	var req addomainServices.ManagerRuleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
	rule, err := h.svc.CreateRule(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "经理同步规则", operlog.OperTypeCreate)
	response.Success(c, rule)
}

// UpdateRule 修改规则
// POST /system/manager-rules/:id/update
func (h *ManagerRuleHandler) UpdateRule(c *gin.Context) {
	var req addomainServices.ManagerRuleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
	if err := h.svc.UpdateRule(c.Request.Context(), c.Param("id"), &req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "经理同步规则", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "修改成功"})
}

// DeleteRule 删除规则
// POST /system/manager-rules/:id/delete
func (h *ManagerRuleHandler) DeleteRule(c *gin.Context) {
	if err := h.svc.DeleteRule(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "经理同步规则", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// ==================== 机构 ====================

// ListOrgs 机构分页列表
// POST /system/manager-orgs/list
func (h *ManagerOrgHandler) ListOrgs(c *gin.Context) {
	var req requests.ManagerOrgListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = requests.DefaultManagerOrgListRequest()
	}
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	page, err := h.svc.ListOrgs(c.Request.Context(), req.Current, req.PageSize, req.Keyword, req.Enabled)
	if err != nil {
		response.Error(c, apperrors.ServerError(err))
		return
	}
	response.Success(c, page)
}

// CreateOrg 新增机构
// POST /system/manager-orgs
func (h *ManagerOrgHandler) CreateOrg(c *gin.Context) {
	var req addomainServices.ManagerOrgCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
	org, err := h.svc.CreateOrg(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "经理同步机构", operlog.OperTypeCreate)
	response.Success(c, org)
}

// UpdateOrg 修改机构
// POST /system/manager-orgs/:id/update
func (h *ManagerOrgHandler) UpdateOrg(c *gin.Context) {
	var req addomainServices.ManagerOrgCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
	if err := h.svc.UpdateOrg(c.Request.Context(), c.Param("id"), &req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "经理同步机构", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "修改成功"})
}

// DeleteOrg 删除机构
// POST /system/manager-orgs/:id/delete
func (h *ManagerOrgHandler) DeleteOrg(c *gin.Context) {
	if err := h.svc.DeleteOrg(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "经理同步机构", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// GenerateOrgsFromDept 从部门架构自动生成机构映射
// POST /system/manager-orgs/generate-from-dept
func (h *ManagerOrgHandler) GenerateOrgsFromDept(c *gin.Context) {
	result, err := h.svc.GenerateOrgsFromDept(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.ServerError(err))
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "经理同步机构", operlog.OperTypeCreate)
	response.Success(c, result)
}
