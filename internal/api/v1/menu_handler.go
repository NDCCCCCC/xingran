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

		// 如果没有筛选条件，使用缓存的菜单数据
		if req.MenuName == "" && req.Status == "" {
			// 使用 MenuCacheService 获取所有菜单（带缓存）
			menuPtrs, err := core.MenuCacheService.GetAllMenusWithCache(c.Request.Context())
			if err != nil {
				// 缓存失败，继续使用数据库查询
			} else {
				// 转换为值类型数组
				menus := make([]models.Menu, len(menuPtrs))
				for i, menu := range menuPtrs {
					menus[i] = *menu
				}
				// 构建树形结构
				menuTree := buildMenuTreeFromPtrs(menuPtrs, nil)
				response.Success(c, menuTree)
				return
			}
		}

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

// buildMenuTreeFromPtrs 从指针数组构建菜单树
func buildMenuTreeFromPtrs(menus []*models.Menu, parentID *string) []models.Menu {
	var tree []models.Menu
	for _, menu := range menus {
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			children := buildMenuTreeFromPtrs(menus, &menu.ID)
			menuCopy := *menu
			if len(children) > 0 {
				menuCopy.Children = children
			}
			tree = append(tree, menuCopy)
		}
	}
	return tree
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
			Remark: func() string {
				if req.Remark != nil {
					return *req.Remark
				}
				return ""
			}(),
		}

		// 使用 MenuCacheService 创建菜单（会自动清除缓存）
		if err := core.MenuCacheService.CreateMenu(c.Request.Context(), &menu); err != nil {
			response.Error(c, response.ErrServerError, "创建菜单失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "菜单管理", OperTypeCreate)

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
		menu.MenuName = req.MenuName
		menu.ParentID = req.ParentID
		menu.OrderNum = req.OrderNum
		menu.Path = req.Path
		menu.Component = req.Component
		menu.MenuType = req.MenuType
		menu.Visible = req.Visible
		menu.Status = req.Status
		menu.Perms = req.Perms
		menu.Icon = req.Icon
		menu.Remark = func() string {
			if req.Remark != nil {
				return *req.Remark
			}
			return ""
		}()

		// 使用 MenuCacheService 更新菜单（会自动清除缓存）
		if err := core.MenuCacheService.UpdateMenu(c.Request.Context(), &menu); err != nil {
			response.Error(c, response.ErrServerError, "更新菜单失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "菜单管理", OperTypeUpdate)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteMenu 删除菜单
// @Summary 删除菜单
// @Description 删除指定菜单，支持级联删除子菜单（需确保没有被角色关联）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path string true "菜单ID"
// @Param cascade query bool false "是否级联删除子菜单，默认false"
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

		// 获取级联删除参数
		cascade := c.DefaultQuery("cascade", "false") == "true"

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

		// 收集所有需要删除的菜单ID（包括子菜单）
		menuIdsToDelete := []string{id}

		if cascade {
			// 递归获取所有子菜单ID
			childIds, err := getAllChildMenuIDs(core.GetDB(), id)
			if err != nil {
				response.Error(c, response.ErrServerError, "获取子菜单列表失败")
				return
			}
			menuIdsToDelete = append(menuIdsToDelete, childIds...)
		} else {
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
		}

		// 检查是否被角色关联
		var count int64
		if err := core.DB.GetDB().Table("sys_role_menu").Where("menu_id IN ?", menuIdsToDelete).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查菜单关联情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "菜单已分配给角色，无法删除")
			return
		}

		// 删除菜单
		if err := core.DB.GetDB().Where("id IN ?", menuIdsToDelete).Delete(&models.Menu{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除菜单失败")
			return
		}

		// 清除缓存
		if err := core.MenuCacheService.InvalidateMenuCache(c.Request.Context()); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		recordOperLog(c, core, "菜单管理", OperTypeDelete)

		deletedCount := len(menuIdsToDelete)
		if cascade && deletedCount > 1 {
			response.Success(c, gin.H{
				"message":      "删除成功",
				"deletedCount": deletedCount,
				"cascade":      true,
			})
		} else {
			response.Success(c, "删除成功")
		}
	}
}

// BatchDeleteMenusRequest 批量删除菜单请求
type BatchDeleteMenusRequest struct {
	IDs     []string `json:"ids" binding:"required,min=1"`
	Cascade bool     `json:"cascade"`
}

// batchDeleteMenus 批量删除菜单
// @Summary 批量删除菜单
// @Description 批量删除菜单，支持级联删除子菜单（需确保没有被角色关联）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body BatchDeleteMenusRequest true "批量删除请求"
// @Success 200 {object} response.Response{data=object}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /menus/batch-delete [post]
func batchDeleteMenus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchDeleteMenusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 去重菜单ID
		idMap := make(map[string]bool)
		for _, id := range req.IDs {
			idMap[id] = true
		}
		uniqueIDs := make([]string, 0, len(idMap))
		for id := range idMap {
			uniqueIDs = append(uniqueIDs, id)
		}

		// 检查菜单是否存在
		var menus []models.Menu
		if err := core.DB.GetDB().Where("id IN ?", uniqueIDs).Find(&menus).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询菜单失败")
			return
		}
		if len(menus) != len(uniqueIDs) {
			response.Error(c, response.ErrRecordNotFound, "部分菜单不存在")
			return
		}

		// 收集所有需要删除的菜单ID（包括子菜单）
		menuIdsToDelete := uniqueIDs

		if req.Cascade {
			// 递归获取所有子菜单ID
			for _, id := range uniqueIDs {
				childIds, err := getAllChildMenuIDs(core.GetDB(), id)
				if err != nil {
					response.Error(c, response.ErrServerError, "获取子菜单列表失败")
					return
				}
				menuIdsToDelete = append(menuIdsToDelete, childIds...)
			}

			// 去重
			idMap = make(map[string]bool)
			for _, id := range menuIdsToDelete {
				idMap[id] = true
			}
			menuIdsToDelete = make([]string, 0, len(idMap))
			for id := range idMap {
				menuIdsToDelete = append(menuIdsToDelete, id)
			}
		} else {
			// 检查是否有子菜单
			var count int64
			if err := core.DB.GetDB().Model(&models.Menu{}).Where("parent_id IN ?", uniqueIDs).Count(&count).Error; err != nil {
				response.Error(c, response.ErrServerError, "检查子菜单失败")
				return
			}
			if count > 0 {
				response.Error(c, response.ErrBadRequest, "存在子菜单，无法删除")
				return
			}
		}

		// 检查是否被角色关联
		var count int64
		if err := core.DB.GetDB().Table("sys_role_menu").Where("menu_id IN ?", menuIdsToDelete).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查菜单关联情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "菜单已分配给角色，无法删除")
			return
		}

		// 删除菜单
		if err := core.DB.GetDB().Where("id IN ?", menuIdsToDelete).Delete(&models.Menu{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除菜单失败")
			return
		}

		// 清除缓存
		if err := core.MenuCacheService.InvalidateMenuCache(c.Request.Context()); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		recordOperLog(c, core, "菜单管理", OperTypeDelete)

		response.Success(c, gin.H{
			"message":      "批量删除成功",
			"deletedCount": len(menuIdsToDelete),
			"cascade":      req.Cascade,
		})
	}
}

// getAllChildMenuIDs 递归获取所有子菜单ID
func getAllChildMenuIDs(db *gorm.DB, parentID string) ([]string, error) {
	var childIDs []string

	// 查询直接子菜单
	var directChildren []models.Menu
	if err := db.Where("parent_id = ?", parentID).Find(&directChildren).Error; err != nil {
		return nil, err
	}

	for _, child := range directChildren {
		childIDs = append(childIDs, child.ID)

		// 递归获取子菜单的子菜单
		grandChildIDs, err := getAllChildMenuIDs(db, child.ID)
		if err != nil {
			return nil, err
		}
		childIDs = append(childIDs, grandChildIDs...)
	}

	return childIDs, nil
}

// getMenuTreeSelect 获取菜单树选择器
// @Summary 获取菜单树选择器
// @Description 获取所有正常状态的菜单树，用于菜单选择器组件（使用缓存）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Failure 500 {object} response.Response
// @Router /menus/tree-select [get]
func getMenuTreeSelect(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用缓存服务获取菜单树
		menuTree, err := core.MenuCacheService.GetMenuTreeWithCache(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, "查询菜单列表失败")
			return
		}

		response.Success(c, menuTree)
	}
}

// getRoleMenuTreeSelect 获取角色菜单树选择器
// @Summary 获取角色菜单树选择器
// @Description 获取指定角色的菜单树和已选中的菜单ID，用于角色菜单权限分配（使用缓存）
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

		// 使用缓存服务获取菜单树
		menuPtrs, err := core.MenuCacheService.GetAllMenusWithCache(c.Request.Context())
		if err != nil {
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

		// 转换为值类型数组并构建树形结构
		menus := make([]models.Menu, len(menuPtrs))
		for i, menu := range menuPtrs {
			menus[i] = *menu
		}
		menuTree := buildMenuTreeSelect(menus, nil)

		response.Success(c, gin.H{
			"menus":       menuTree,
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

// getUserMenus 获取用户菜单树（不包含隐藏菜单）
// @Summary 获取用户菜单树
// @Description 获取当前用户的菜单树（仅包含可见菜单，用于导航栏渲染）
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
		menus, err := permissionSvc.GetUserMenus(core.GetDB(), userID.(string), false) // 不包含隐藏菜单
		if err != nil {
			response.Error(c, response.ErrServerError, "获取用户菜单失败")
			return
		}

		response.Success(c, menus)
	}
}

// getAllUserMenus 获取用户所有菜单（包含隐藏菜单）
// @Summary 获取用户所有菜单
// @Description 获取当前用户的所有菜单（包括隐藏菜单，用于标签页标题等功能）
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Router /system/menus/user-all [post]
func getAllUserMenus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			return
		}

		permissionSvc := permission.NewService()
		menus, err := permissionSvc.GetUserMenus(core.GetDB(), userID.(string), true) // 包含隐藏菜单
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
