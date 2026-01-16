package v1

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// RoleListRequest 角色列表请求
type RoleListRequest struct {
	RoleName string `json:"roleName,omitempty"`
	RoleKey  string `json:"roleKey,omitempty"`
	Status   string `json:"status,omitempty"`
	Current  int    `json:"current,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// RoleCreateRequest 角色创建请求
type RoleCreateRequest struct {
	RoleName          string            `json:"roleName" binding:"required,max=50"`
	RoleKey           string            `json:"roleKey" binding:"required,max=50"`
	RoleSort          int               `json:"roleSort" binding:"min=0"`
	DataScope         models.DataScope  `json:"dataScope"`
	MenuCheckStrictly *bool             `json:"menuCheckStrictly"`
	DeptCheckStrictly *bool             `json:"deptCheckStrictly"`
	Status            models.RoleStatus `json:"status"`
	Remark            *string           `json:"remark,omitempty"`
	MenuIds           []string          `json:"menuIds,omitempty"`
	DeptIds           []string          `json:"deptIds,omitempty"`
}

// RoleUpdateRequest 角色更新请求
type RoleUpdateRequest struct {
	ID                string            `json:"id" binding:"required"`
	RoleName          string            `json:"roleName" binding:"required,max=50"`
	RoleKey           string            `json:"roleKey" binding:"required,max=50"`
	RoleSort          int               `json:"roleSort" binding:"min=0"`
	DataScope         models.DataScope  `json:"dataScope"`
	MenuCheckStrictly *bool             `json:"menuCheckStrictly"`
	DeptCheckStrictly *bool             `json:"deptCheckStrictly"`
	Status            models.RoleStatus `json:"status"`
	Remark            *string           `json:"remark,omitempty"`
	MenuIds           []string          `json:"menuIds,omitempty"`
	DeptIds           []string          `json:"deptIds,omitempty"`
}

// RoleStatusRequest 角色状态请求
type RoleStatusRequest struct {
	Status models.RoleStatus `json:"status" binding:"required"`
}

// RoleMenusRequest 角色菜单权限请求
type RoleMenusRequest struct {
	MenuIds []string `json:"menuIds" binding:"required"`
}

// listRoles 查询角色列表
// @Summary 查询角色列表
// @Description 分页查询角色列表，支持按角色名称、权限字符、状态筛选
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body RoleListRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.Role}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/list [post]
func listRoles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RoleListRequest
		// 允许空的请求体，所有字段都是可选的
		if err := c.ShouldBindJSON(&req); err != nil {
			// 如果是空请求体，初始化默认值
			req.Current = 1
			req.PageSize = 10
		}

		// 设置分页默认值
		current := req.Current
		pageSize := req.PageSize
		if current <= 0 {
			current = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.Role{})

		// 角色名称模糊查询
		if req.RoleName != "" {
			db = db.Where("role_name LIKE ?", "%"+req.RoleName+"%")
		}

		// 权限字符模糊查询
		if req.RoleKey != "" {
			db = db.Where("role_key LIKE ?", "%"+req.RoleKey+"%")
		}

		// 状态查询
		if req.Status != "" {
			if status, err := strconv.Atoi(req.Status); err == nil {
				db = db.Where("status = ?", status)
			}
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询角色总数失败")
			return
		}

		// 查询分页数据
		var roles []models.Role
		offset := (current - 1) * pageSize
		if err := db.Offset(offset).Limit(pageSize).Order("role_sort ASC, created_at DESC").Find(&roles).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询角色列表失败")
			return
		}

		response.Page(c, roles, total, current, pageSize)
	}
}

// getAllRoles 获取所有启用角色（用于下拉选择）
// @Summary 获取所有启用角色
// @Description 获取所有启用状态的角色，用于下拉选择
// @Tags 角色管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Role}
// @Router /roles/all [post]
func getAllRoles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 RoleCacheService 获取所有启用的角色（带缓存）
		roles, err := core.RoleCacheService.GetAllRolesWithCache(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, "查询角色列表失败")
			return
		}

		// 只返回必要的字段
		var simplifiedRoles []map[string]interface{}
		for _, role := range roles {
			simplifiedRoles = append(simplifiedRoles, map[string]interface{}{
				"id":       role.ID,
				"roleName": role.RoleName,
				"roleKey":  role.RoleKey,
			})
		}

		response.Success(c, simplifiedRoles)
	}
}

// createRole 创建角色
// @Summary 创建角色
// @Description 创建新的角色，支持设置角色名称、权限字符、数据权限、菜单权限、部门权限等
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body RoleCreateRequest true "角色创建信息"
// @Success 200 {object} response.Response{data=models.Role}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles [post]
func createRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, ok := bindAndParseRoleRequest(c)
		if !ok {
			return
		}

		// 检查角色名称是否已存在
		if exists, err := checkRoleNameExists(core.DB.GetDB(), req.RoleName, ""); err != nil {
			response.Error(c, response.ErrServerError, "检查角色名称失败")
			return
		} else if exists {
			response.Error(c, response.ErrRecordExists, "角色名称已存在")
			return
		}

		// 检查权限字符是否已存在
		if exists, err := checkRoleKeyExists(core.DB.GetDB(), req.RoleKey, ""); err != nil {
			response.Error(c, response.ErrServerError, "检查权限字符失败")
			return
		} else if exists {
			response.Error(c, response.ErrRecordExists, "权限字符已存在")
			return
		}

		var role models.Role
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			newRole := models.Role{
				RoleName:          req.RoleName,
				RoleKey:           req.RoleKey,
				RoleSort:          req.RoleSort,
				DataScope:         models.DataScope(req.DataScope),
				MenuCheckStrictly: true,
				DeptCheckStrictly: true,
				Status:            models.RoleStatus(req.Status),
				Remark:            derefString(req.Remark),
			}

			if err := tx.Create(&newRole).Error; err != nil {
				return fmt.Errorf("创建角色失败: %w", err)
			}

			if err := assignRoleMenusAndDepts(tx, newRole.ID, req.MenuIds, req.DeptIds); err != nil {
				return err
			}

			role = newRole
			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		recordOperLog(c, core, "角色管理", OperTypeCreate)
		response.Success(c, role)
	}
}

// getRole 获取角色详情
// @Summary 获取角色详情
// @Description 根据角色ID获取角色详细信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} response.Response{data=models.Role}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id} [get]
func getRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		response.Success(c, role)
	}
}

// updateRole 更新角色
// @Summary 更新角色
// @Description 更新角色信息，包括角色名称、权限字符、数据权限、菜单权限、部门权限等
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body RoleUpdateRequest true "角色更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id} [put]
func updateRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := getRequiredParam(c, "id", "角色ID")
		if !ok {
			return
		}

		req, ok := bindAndParseRoleRequest(c)
		if !ok {
			return
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 检查角色名称是否与其他角色重复
		if exists, err := checkRoleNameExists(core.DB.GetDB(), req.RoleName, id); err != nil {
			response.Error(c, response.ErrServerError, "检查角色名称失败")
			return
		} else if exists {
			response.Error(c, response.ErrRecordExists, "角色名称已存在")
			return
		}

		// 检查权限字符是否与其他角色重复
		if exists, err := checkRoleKeyExists(core.DB.GetDB(), req.RoleKey, id); err != nil {
			response.Error(c, response.ErrServerError, "检查权限字符失败")
			return
		} else if exists {
			response.Error(c, response.ErrRecordExists, "权限字符已存在")
			return
		}

		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			updates := map[string]interface{}{
				"role_name":           req.RoleName,
				"role_key":            req.RoleKey,
				"role_sort":           req.RoleSort,
				"data_scope":          models.DataScope(req.DataScope),
				"menu_check_strictly": true,
				"dept_check_strictly": true,
				"status":              models.RoleStatus(req.Status),
				"remark":              derefString(req.Remark),
			}

			if err := tx.Model(&role).Updates(updates).Error; err != nil {
				return fmt.Errorf("更新角色失败: %w", err)
			}

			return updateRoleMenusAndDepts(tx, id, req.MenuIds, req.DeptIds)
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		recordOperLog(c, core, "角色管理", OperTypeUpdate)
		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteRole 删除角色
// @Summary 删除角色
// @Description 删除指定角色（需确保没有用户使用该角色）
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id} [delete]
func deleteRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 检查是否有用户使用该角色
		var count int64
		if err := core.DB.GetDB().Table("sys_user_roles").Where("role_id = ?", id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查角色使用情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "该角色已分配给用户，无法删除")
			return
		}

		// 使用事务
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			// 删除角色菜单关联
			if err := tx.Where("role_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
				return fmt.Errorf("删除角色菜单关联失败: %w", err)
			}

			// 删除角色部门关联
			if err := tx.Where("role_id = ?", id).Delete(&models.RoleDept{}).Error; err != nil {
				return fmt.Errorf("删除角色部门关联失败: %w", err)
			}

			// 删除角色
			if err := tx.Delete(&role).Error; err != nil {
				return fmt.Errorf("删除角色失败: %w", err)
			}

			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "角色管理", OperTypeDelete)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// batchDeleteRoles 批量删除角色
// @Summary 批量删除角色
// @Description 批量删除多个角色（需确保没有用户使用这些角色）
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "角色ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/batch [delete]
func batchDeleteRoles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			IDs []string `json:"ids" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if len(req.IDs) == 0 {
			response.Error(c, response.ErrBadRequest, "请选择要删除的角色")
			return
		}

		// 检查是否有用户使用这些角色
		var count int64
		if err := core.DB.GetDB().Table("sys_user_roles").Where("role_id IN ?", req.IDs).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查角色使用情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "存在已分配给用户的角色，无法删除")
			return
		}

		// 使用事务
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			// 删除角色菜单关联
			if err := tx.Where("role_id IN ?", req.IDs).Delete(&models.RoleMenu{}).Error; err != nil {
				return fmt.Errorf("删除角色菜单关联失败: %w", err)
			}

			// 删除角色部门关联
			if err := tx.Where("role_id IN ?", req.IDs).Delete(&models.RoleDept{}).Error; err != nil {
				return fmt.Errorf("删除角色部门关联失败: %w", err)
			}

			// 删除角色
			if err := tx.Where("id IN ?", req.IDs).Delete(&models.Role{}).Error; err != nil {
				return fmt.Errorf("删除角色失败: %w", err)
			}

			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "角色管理", OperTypeDelete)

		response.Success(c, gin.H{"message": "批量删除成功"})
	}
}

// updateRoleStatus 更改角色状态
// @Summary 更改角色状态
// @Description 启用或禁用指定角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body RoleStatusRequest true "角色状态"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id}/status [put]
func updateRoleStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		var req struct {
			Status int `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// 尝试从原始请求中读取
			var rawReq map[string]interface{}
			if err2 := c.ShouldBindJSON(&rawReq); err2 != nil {
				response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
				return
			}
			// 使用辅助函数转换status值
			if statusVal, ok := rawReq["status"]; ok {
				req.Status = parseStatusToInt(statusVal)
			} else {
				response.Error(c, response.ErrBadRequest, "缺少status字段")
				return
			}
		}

		// 验证status的值
		if req.Status != 0 && req.Status != 1 {
			response.Error(c, response.ErrBadRequest, "状态值无效，必须是0(启用)或1(禁用)")
			return
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 更新状态
		if err := core.DB.GetDB().Model(&role).Update("status", models.RoleStatus(req.Status)).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新角色状态失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "角色管理", OperTypeUpdate)

		response.Success(c, gin.H{"message": "更新状态成功"})
	}
}

// updateRoleMenus 更新角色菜单权限
// @Summary 更新角色菜单权限
// @Description 更新角色的菜单权限，支持分配或取消分配菜单权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body RoleMenusRequest true "菜单ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id}/menus [put]
func updateRoleMenus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := getRequiredParam(c, "id", "角色ID")
		if !ok {
			return
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 绑定菜单ID列表
		var req struct {
			MenuIds []string `json:"menuIds"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			var rawReq map[string]interface{}
			if err2 := c.ShouldBindJSON(&rawReq); err2 != nil {
				response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
				return
			}
			if menuIds, ok := rawReq["menuIds"]; ok {
				req.MenuIds = toStringSlice(menuIds)
			}
		}

		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			return updateRoleMenusAndDepts(tx, id, req.MenuIds, nil)
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		recordOperLog(c, core, "角色管理", OperTypeGrant)
		response.Success(c, gin.H{"message": "菜单权限分配成功"})
	}
}
