package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// 常量定义
const (
	ErrMsgMenuIDRequired = "菜单ID不能为空"
	ErrMsgRoleIDRequired = "角色ID不能为空"
)

// MenuHandler 菜单处理器
type MenuHandler struct {
	service systemServices.MenuService
	core    *core.Core
}

// NewMenuHandler 创建菜单处理器实例
func NewMenuHandler(service systemServices.MenuService) *MenuHandler {
	return &MenuHandler{service: service}
}

// WithCore 注入 core 依赖（Phase 34 操作日志记录所需）。
func (h *MenuHandler) WithCore(core *core.Core) *MenuHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// ==================== 辅助方法 ====================

// requireID 获取并验证路径参数ID
func (h *MenuHandler) requireID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("菜单ID"))
		return "", false
	}
	return id, true
}

// getUserID 从上下文获取用户ID
func (h *MenuHandler) getUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return "", false
	}
	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, apperrors.Unauthorized())
		return "", false
	}
	return userIDStr, true
}

// bindAndValidate 绑定并验证请求参数
func (h *MenuHandler) bindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return false
	}
	return true
}

// GetTree 获取菜单树
// @Summary 获取菜单树
// @Description 获取菜单树形结构
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/menus/tree [post]
func (h *MenuHandler) GetTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tree)
}

// List 查询菜单列表
// @Summary 查询菜单列表
// @Description 查询菜单列表
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body requests.MenuListParams true "查询条件"
// @Success 200 {object} response.Response
// @Router /system/menus/list [post]
func (h *MenuHandler) List(c *gin.Context) {
	var req requests.MenuListParams
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetByID 获取菜单详情
// @Summary 获取菜单详情
// @Description 根据菜单ID获取菜单详细信息
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Success 200 {object} response.Response
// @Router /system/menus/:id [post]
func (h *MenuHandler) GetByID(c *gin.Context) {
	id, ok := h.requireID(c)
	if !ok {
		return
	}

	menu, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, menu)
}

// Create 创建菜单
// @Summary 创建菜单
// @Description 创建新的菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body requests.MenuCreateRequest true "菜单信息"
// @Success 200 {object} response.Response
// @Router /system/menus [post]
func (h *MenuHandler) Create(c *gin.Context) {
	var req requests.MenuCreateRequest
	if !h.bindAndValidate(c, &req) {
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "菜单管理", operlog.OperTypeCreate)
	response.Success(c, gin.H{"message": "创建成功"})
}

// Update 更新菜单
// @Summary 更新菜单
// @Description 更新菜单信息
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Param request body requests.MenuUpdateRequest true "菜单信息"
// @Success 200 {object} response.Response
// @Router /system/menus/:id/update [post]
func (h *MenuHandler) Update(c *gin.Context) {
	id, ok := h.requireID(c)
	if !ok {
		return
	}

	var req requests.MenuUpdateRequest
	if !h.bindAndValidate(c, &req) {
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "菜单管理", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除菜单
// @Summary 删除菜单
// @Description 删除指定菜单，支持级联删除子菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Param cascade query bool false "是否级联删除子菜单" default(false)
// @Success 200 {object} response.Response
// @Router /system/menus/:id/delete [post]
func (h *MenuHandler) Delete(c *gin.Context) {
	id, ok := h.requireID(c)
	if !ok {
		return
	}

	// 读取 cascade 查询参数，默认为 false
	cascade := c.DefaultQuery("cascade", "false") == "true"

	if err := h.service.Delete(c.Request.Context(), id, cascade); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "菜单管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchDelete 批量删除菜单
// @Summary 批量删除菜单
// @Description 批量删除多个菜单，支持级联删除子菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string,cascade=bool} true "菜单ID列表"
// @Success 200 {object} response.Response
// @Router /system/menus/batch [post]
func (h *MenuHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs     []string `json:"ids" binding:"required,min=1"`
		Cascade bool     `json:"cascade"`
	}

	if !h.bindAndValidate(c, &req) {
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs, req.Cascade); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "菜单管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}

// UpdateStatus 更新菜单状态
// @Summary 更新菜单状态
// @Description 更新菜单的启用/停用状态
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Param request body object{status=int} true "状态"
// @Success 200 {object} response.Response
// @Router /system/menus/:id/status [post]
func (h *MenuHandler) UpdateStatus(c *gin.Context) {
	id, ok := h.requireID(c)
	if !ok {
		return
	}

	var req struct {
		Status int `json:"status" binding:"min=0,max=1"`
	}
	if !h.bindAndValidate(c, &req) {
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "菜单管理", operlog.OperTypeStatus)
	response.Success(c, gin.H{"message": "状态更新成功"})
}

// GetUserMenus 获取当前用户的菜单列表
// @Summary 获取用户菜单
// @Description 获取当前登录用户的菜单树（用于前端渲染导航菜单）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/my-menus [post]
func (h *MenuHandler) GetUserMenus(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	menus, err := h.service.GetUserMenus(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, menus)
}

// GetAllUserMenus 获取当前用户的所有菜单（包含隐藏菜单）
// @Summary 获取用户所有菜单
// @Description 获取当前登录用户的菜单树（包含隐藏菜单，用于标签页标题）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/my-menus/all [post]
func (h *MenuHandler) GetAllUserMenus(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	menus, err := h.service.GetAllUserMenus(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, menus)
}

// GetUserPermissions 获取当前用户的权限列表
// @Summary 获取用户权限
// @Description 获取当前登录用户的权限标识列表
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]string}
// @Router /system/my-menus/permissions [post]
func (h *MenuHandler) GetUserPermissions(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	permissions, err := h.service.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, permissions)
}

// RoleMenuTreeSelect 获取角色菜单树选择器
// @Summary 获取角色菜单树选择器
// @Description 获取指定角色的菜单树和已选中的菜单ID，用于角色菜单权限分配
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param roleId path string true "角色ID"
// @Success 200 {object} response.Response{data=object{checkedKeys=[]string}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/menus/role-menu-tree-select/{roleId} [post]
func (h *MenuHandler) RoleMenuTreeSelect(c *gin.Context) {
	roleID := c.Param("roleId")
	if roleID == "" {
		response.Error(c, apperrors.ParamMissing("角色ID"))
		return
	}

	// 查询角色已关联的菜单ID
	var menuIDs []string
	if err := h.service.GetRoleMenuIDs(c.Request.Context(), roleID, &menuIDs); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, "查询角色菜单关联失败"))
		return
	}

	response.Success(c, gin.H{
		"checkedKeys": menuIDs,
	})
}
