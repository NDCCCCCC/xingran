package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// MenuListRequest 菜单列表请求
type MenuListRequest struct {
	MenuName string `json:"menuName,omitempty"`
	Status   string `json:"status,omitempty"`
	Current  int    `json:"current,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// MenuCreateRequest 菜单创建请求
type MenuCreateRequest struct {
	MenuName  string             `json:"menuName" binding:"required,max=50"`
	ParentID  *string            `json:"parentId,omitempty"`
	OrderNum  int                `json:"orderNum"`
	Path      *string            `json:"path,omitempty"`
	Component *string            `json:"component,omitempty"`
	MenuType  models.MenuType    `json:"menuType"`
	Visible   models.VisibleType `json:"visible"`
	Status    models.MenuStatus  `json:"status"`
	Perms     *string            `json:"perms,omitempty"`
	Icon      *string            `json:"icon,omitempty"`
	Remark    *string            `json:"remark,omitempty"`
}

// MenuUpdateRequest 菜单更新请求
type MenuUpdateRequest struct {
	ID        string             `json:"id" binding:"required"`
	MenuName  string             `json:"menuName" binding:"required,max=50"`
	ParentID  *string            `json:"parentId,omitempty"`
	OrderNum  int                `json:"orderNum"`
	Path      *string            `json:"path,omitempty"`
	Component *string            `json:"component,omitempty"`
	MenuType  models.MenuType    `json:"menuType"`
	Visible   models.VisibleType `json:"visible"`
	Status    models.MenuStatus  `json:"status"`
	Perms     *string            `json:"perms,omitempty"`
	Icon      *string            `json:"icon,omitempty"`
	Remark    *string            `json:"remark,omitempty"`
}

// listMenus 查询菜单列表
// @Summary 查询菜单列表
// @Description 查询菜单列表，返回树形结构的菜单数据，支持按菜单名称、状态筛选
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body MenuListRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Failure 500 {object} response.Response
// @Router /menus/list [post]
func listMenus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MenuListRequest
		// 允许空的请求体，所有字段都是可选的
		c.ShouldBindJSON(&req)

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.Menu{})

		// 菜单名称模糊查询
		if req.MenuName != "" {
			db = db.Where("menu_name LIKE ?", "%"+req.MenuName+"%")
		}

		// 状态查询
		if req.Status != "" {
			if status, err := strconv.Atoi(req.Status); err == nil {
				db = db.Where("status = ?", status)
			}
		}

		// 查询数据并按order_num排序
		// 注意：菜单是树形结构，不需要分页，所以忽略current和pageSize参数
		var menus []models.Menu
		if err := db.Order("order_num ASC").Find(&menus).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询菜单列表失败")
			return
		}

		// 构建树形结构
		menuTree := buildMenuTree(menus, nil)

		response.Success(c, menuTree)
	}
}

// buildMenuTree 构建菜单树
func buildMenuTree(menus []models.Menu, parentID *string) []models.Menu {
	var tree []models.Menu
	for _, menu := range menus {
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			children := buildMenuTree(menus, &menu.ID)
			if len(children) > 0 {
				menu.Children = children
			}
			tree = append(tree, menu)
		}
	}
	return tree
}

// createMenu 创建菜单
// @Summary 创建菜单
// @Description 创建新的菜单，支持设置菜单名称、父菜单、路径、组件、图标、权限标识等
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body MenuCreateRequest true "菜单创建信息"
// @Success 200 {object} response.Response{data=models.Menu}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /menus [post]
func createMenu(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MenuCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 处理空字符串的 ParentID，将其转换为 nil
		if req.ParentID != nil && *req.ParentID == "" {
			req.ParentID = nil
		}

		// 检查菜单名称是否已存在
		var count int64
		db := core.DB.GetDB().Model(&models.Menu{})
		if req.ParentID != nil {
			db = db.Where("parent_id = ? AND menu_name = ?", req.ParentID, req.MenuName)
		} else {
			db = db.Where("parent_id IS NULL AND menu_name = ?", req.MenuName)
		}
		if err := db.Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查菜单名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "菜单名称已存在")
			return
		}

		// 创建菜单
		menu := models.Menu{
			MenuName:  req.MenuName,
			ParentID:  req.ParentID,
			OrderNum:  req.OrderNum,
			Path:      req.Path,
			Component: req.Component,
			MenuType:  req.MenuType,
			Visible:   req.Visible,
			Status:    req.Status,
			Perms:     req.Perms,
			Icon:      req.Icon,
			Remark:    func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Create(&menu).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建菜单失败")
			return
		}

		response.Success(c, menu)
	}
}

// getMenu 获取菜单详情
// @Summary 获取菜单详情
// @Description 根据菜单ID获取菜单详细信息
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Success 200 {object} response.Response{data=models.Menu}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /menus/{id} [get]
func getMenu(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "菜单ID不能为空")
			return
		}

		var menu models.Menu
		if err := core.DB.GetDB().First(&menu, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "菜单不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询菜单失败")
			}
			return
		}

		response.Success(c, menu)
	}
}

// updateMenu 更新菜单
// @Summary 更新菜单
// @Description 更新菜单信息，包括菜单名称、父菜单、路径、组件、图标、权限标识等
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Param request body MenuUpdateRequest true "菜单更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /menus/{id} [put]
func updateMenu(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "菜单ID不能为空")
			return
		}

		var req MenuUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 处理空字符串的 ParentID，将其转换为 nil
		if req.ParentID != nil && *req.ParentID == "" {
			req.ParentID = nil
		}

		// 检查菜单是否存在
		var menu models.Menu
		if err := core.DB.GetDB().First(&menu, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "菜单不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询菜单失败")
			}
			return
		}

		// 检查是否将自己设置为父菜单
		if req.ParentID != nil && *req.ParentID == id {
			response.Error(c, response.ErrBadRequest, "不能将自己设置为父菜单")
			return
		}

		// 检查菜单名称是否与其他菜单重复
		var count int64
		db := core.DB.GetDB().Model(&models.Menu{}).Where("id != ?", id)
		if req.ParentID != nil {
			db = db.Where("parent_id = ? AND menu_name = ?", req.ParentID, req.MenuName)
		} else {
			db = db.Where("parent_id IS NULL AND menu_name = ?", req.MenuName)
		}
		if err := db.Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查菜单名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "菜单名称已存在")
			return
		}

		// 更新菜单信息
		updates := map[string]interface{}{
			"menu_name":  req.MenuName,
			"parent_id":  req.ParentID,
			"order_num":  req.OrderNum,
			"path":       req.Path,
			"component":  req.Component,
			"menu_type":  req.MenuType,
			"visible":    req.Visible,
			"status":     req.Status,
			"perms":      req.Perms,
			"icon":       req.Icon,
			"remark":     func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Model(&menu).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新菜单失败")
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteMenu 删除菜单
// @Summary 删除菜单
// @Description 删除指定菜单（需确保没有子菜单且没有被角色关联）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /menus/{id} [delete]
func deleteMenu(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "菜单ID不能为空")
			return
		}

		// 检查菜单是否存在
		var menu models.Menu
		if err := core.DB.GetDB().First(&menu, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "菜单不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询菜单失败")
			}
			return
		}

		// 检查是否有子菜单
		var count int64
		if err := core.DB.GetDB().Model(&models.Menu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查子菜单失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "存在子菜单，无法删除")
			return
		}

		// 检查是否被角色关联
		if err := core.DB.GetDB().Table("sys_role_menu").Where("menu_id = ?", id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查菜单关联情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "菜单已分配给角色，无法删除")
			return
		}

		// 删除菜单
		if err := core.DB.GetDB().Delete(&menu).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除菜单失败")
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// getMenuTreeSelect 获取菜单树选择器
// @Summary 获取菜单树选择器
// @Description 获取所有正常状态的菜单树，用于菜单选择器组件
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Failure 500 {object} response.Response
// @Router /menus/tree-select [get]
func getMenuTreeSelect(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var menus []models.Menu
		if err := core.DB.GetDB().Where("status = ?", models.MenuStatusNormal).
			Order("order_num ASC").Find(&menus).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询菜单列表失败")
			return
		}

		// 构建树形结构
		menuTree := buildMenuTree(menus, nil)

		response.Success(c, menuTree)
	}
}

// getRoleMenuTreeSelect 获取角色菜单树选择器
// @Summary 获取角色菜单树选择器
// @Description 获取指定角色的菜单树和已选中的菜单ID，用于角色菜单权限分配
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param roleId path string true "角色ID"
// @Success 200 {object} response.Response{data=object}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /menus/role-menu-tree-select/{roleId} [get]
func getRoleMenuTreeSelect(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleId := c.Param("roleId")
		if roleId == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		// 查询所有菜单
		var menus []models.Menu
		if err := core.DB.GetDB().Where("status = ?", models.MenuStatusNormal).
			Order("order_num ASC").Find(&menus).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询菜单列表失败")
			return
		}

		// 查询角色已关联的菜单ID
		var menuIds []string
		if err := core.DB.GetDB().Table("sys_role_menu").
			Where("role_id = ?", roleId).Pluck("menu_id", &menuIds).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询角色菜单关联失败")
			return
		}

		// 构建树形结构
		menuTree := buildMenuTreeSelect(menus, nil)

		response.Success(c, gin.H{
			"menus":    menuTree,
			"checkedKeys": menuIds,
		})
	}
}

// buildMenuTreeSelect 构建菜单树选择器
func buildMenuTreeSelect(menus []models.Menu, parentID *string) []map[string]interface{} {
	var tree []map[string]interface{}
	for _, menu := range menus {
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			children := buildMenuTreeSelect(menus, &menu.ID)
			node := map[string]interface{}{
				"id":       menu.ID,
				"key":      menu.ID,
				"title":    menu.MenuName,
				"value":    menu.ID,
				"children": children,
			}
			if len(children) == 0 {
				delete(node, "children")
			}
			tree = append(tree, node)
		}
	}
	return tree
}

// getUserMenus 获取用户菜单树
// @Summary 获取用户菜单树
// @Description 获取当前用户的菜单树
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Router /system/menus/user-menus [post]
func getUserMenus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			return
		}

		permissionSvc := permission.NewService()
		menus, err := permissionSvc.GetUserMenus(core.GetDB(), userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, "获取用户菜单失败")
			return
		}

		response.Success(c, menus)
	}
}

// getUserPermissions 获取用户权限列表
// @Summary 获取用户权限列表
// @Description 获取当前用户的所有权限
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]string}
// @Router /system/menus/user-permissions [post]
func getUserPermissions(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			return
		}

		permissions, err := middleware.GetCurrentUserPermissions(core, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, "获取用户权限失败")
			return
		}

		response.Success(c, permissions)
	}
}
