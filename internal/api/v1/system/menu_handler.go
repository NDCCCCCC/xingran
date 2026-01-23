package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// MenuHandler 菜单处理器
type MenuHandler struct {
	service systemServices.MenuService
}

// NewMenuHandler 创建菜单处理器实例
func NewMenuHandler(service systemServices.MenuService) *MenuHandler {
	return &MenuHandler{service: service}
}

// GetTree 获取菜单树
// @Summary 获取菜单树
// @Description 获取菜单树形结构
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Router /system/menus/tree [post]
func (h *MenuHandler) GetTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
// @Param request body systemServices.MenuListParams true "查询条件"
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Router /system/menus/list [post]
func (h *MenuHandler) List(c *gin.Context) {
	var req systemServices.MenuListParams
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求
		req = systemServices.MenuListParams{}
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
// @Success 200 {object} response.Response{data=models.Menu}
// @Router /system/menus/:id [post]
func (h *MenuHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "菜单ID不能为空")
		return
	}

	menu, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
// @Param request body systemServices.MenuCreateRequest true "菜单信息"
// @Success 200 {object} response.Response{data=models.Menu}
// @Router /system/menus [post]
func (h *MenuHandler) Create(c *gin.Context) {
	var req systemServices.MenuCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

// Update 更新菜单
// @Summary 更新菜单
// @Description 更新菜单信息
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Param request body systemServices.MenuUpdateRequest true "菜单信息"
// @Success 200 {object} response.Response
// @Router /system/menus/:id/update [post]
func (h *MenuHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "菜单ID不能为空")
		return
	}

	var req systemServices.MenuUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除菜单
// @Summary 删除菜单
// @Description 删除指定菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Success 200 {object} response.Response
// @Router /system/menus/:id/delete [post]
func (h *MenuHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "菜单ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除菜单
// @Summary 批量删除菜单
// @Description 批量删除多个菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "菜单ID列表"
// @Success 200 {object} response.Response
// @Router /system/menus/batch [post]
func (h *MenuHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

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
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "菜单ID不能为空")
		return
	}

	var req struct {
		Status int `json:"status" binding:"min=0,max=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "状态更新成功"})
}

// GetUserMenus 获取当前用户的菜单列表
// @Summary 获取用户菜单
// @Description 获取当前登录用户的菜单树（用于前端渲染导航菜单）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Router /user-menus [get]
func (h *MenuHandler) GetUserMenus(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	menus, err := h.service.GetUserMenus(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Router /system/my-menus/all [post]
func (h *MenuHandler) GetAllUserMenus(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	menus, err := h.service.GetAllUserMenus(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
// @Router /user-permissions [get]
func (h *MenuHandler) GetUserPermissions(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	permissions, err := h.service.GetUserPermissions(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
		response.Error(c, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	// 查询角色已关联的菜单ID
	var menuIDs []string
	if err := h.service.GetRoleMenuIDs(c.Request.Context(), roleID, &menuIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, "查询角色菜单关联失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"checkedKeys": menuIDs,
	})
}
