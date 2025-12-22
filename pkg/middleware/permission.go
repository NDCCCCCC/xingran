package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// Permission 权限中间件
func Permission(permission string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 检查用户权限
		if !checkUserPermission(core, userID.(string), permission) {
			response.Error(c, response.ErrForbidden, "没有访问权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkUserPermission 检查用户权限
func checkUserPermission(core *core.Core, userID, permission string) bool {
	// 超级管理员直接通过
	if isSuperAdmin(core, userID) {
		return true
	}

	// 查询用户所有角色权限
	var count int64
	err := core.DB.GetDB().Raw(`
		SELECT COUNT(DISTINCT rm.menu_id)
		FROM sys_user_role ur
		INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
		INNER JOIN sys_menu m ON rm.menu_id = m.id
		WHERE ur.user_id = ? AND m.perms = ? AND m.status = ? AND m.visible = ?
	`, userID, permission, models.MenuStatusNormal, models.VisibleShow).Scan(&count).Error

	if err != nil {
		return false
	}

	return count > 0
}

// isSuperAdmin 检查是否是超级管理员
func isSuperAdmin(core *core.Core, userID string) bool {
	// 检查用户是否有超级管理员角色
	var count int64
	err := core.DB.GetDB().Raw(`
		SELECT COUNT(*)
		FROM sys_user_role ur
		INNER JOIN sys_role r ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.role_key = ?
	`, userID, "admin").Scan(&count).Error

	if err != nil {
		return false
	}

	return count > 0
}

// GetCurrentUserPermissions 获取当前用户所有权限
func GetCurrentUserPermissions(core *core.Core, userID string) ([]string, error) {
	permissionSvc := permission.NewService()
	return permissionSvc.GetUserPermissions(core.GetDB(), userID)
}

// RequirePermissions 需要多个权限中的任意一个
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 超级管理员直接通过
		if isSuperAdmin(core, userID.(string)) {
			c.Next()
			return
		}

		// 检查是否有任意一个权限
		for _, permission := range permissions {
			if checkUserPermission(core, userID.(string), permission) {
				c.Next()
				return
			}
		}

		response.Error(c, response.ErrForbidden, "没有访问权限")
		c.Abort()
	}
}

// RequireAllPermissions 需要所有权限
func RequireAllPermissions(permissions []string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 超级管理员直接通过
		if isSuperAdmin(core, userID.(string)) {
			c.Next()
			return
		}

		// 检查是否拥有所有权限
		for _, permission := range permissions {
			if !checkUserPermission(core, userID.(string), permission) {
				response.Error(c, response.ErrForbidden, "没有访问权限")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// DataScopePermission 数据权限中间件
func DataScopePermission(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 超级管理员直接通过
		if isSuperAdmin(core, userID.(string)) {
			c.Next()
			return
		}

		// 获取用户最大数据权限范围
		dataScope, err := getUserMaxDataScope(core, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, "获取数据权限失败")
			c.Abort()
			return
		}

		// 将数据权限信息存储到上下文
		c.Set("data_scope", dataScope)
		c.Set("user_id", userID)

		c.Next()
	}
}

// getUserMaxDataScope 获取用户最大数据权限范围
func getUserMaxDataScope(core *core.Core, userID string) (models.DataScope, error) {
	var dataScope models.DataScope

	err := core.DB.GetDB().Raw(`
		SELECT COALESCE(MAX(r.data_scope), ?) as data_scope
		FROM sys_user_roles ur
		INNER JOIN sys_role r ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.status = ?
	`, models.DataScopeSelf, userID, models.RoleStatusEnabled).Scan(&dataScope).Error

	return dataScope, err
}

// ApplyDataScope 应用数据权限过滤
func ApplyDataScope(db *gorm.DB, userID string, dataScope models.DataScope, core *core.Core, deptField string) *gorm.DB {
	switch dataScope {
	case models.DataScopeAll:
		// 全部数据权限，不做过滤
		return db
	case models.DataScopeCustom:
		// 自定义数据权限，查询用户可访问的部门
		var deptIds []string
		core.DB.GetDB().Raw(`
			SELECT DISTINCT rd.dept_id
			FROM sys_user_roles ur
			INNER JOIN sys_role_dept rd ON ur.role_id = rd.role_id
			WHERE ur.user_id = ?
		`, userID).Scan(&deptIds)

		if len(deptIds) > 0 {
			return db.Where(fmt.Sprintf("%s IN ?", deptField), deptIds)
		}
		return db.Where("1=0") // 没有权限访问任何数据
	case models.DataScopeDept:
		// 本部门数据权限，查询用户所属部门
		var deptId string
		core.DB.GetDB().Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId)
		if deptId != "" {
			return db.Where(fmt.Sprintf("%s = ?", deptField), deptId)
		}
		return db.Where("1=0")
	case models.DataScopeDeptChild:
		// 本部门及子部门数据权限
		var deptId string
		core.DB.GetDB().Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId)
		if deptId != "" {
			// 查询本部门及所有子部门
			var childDeptIds []string
			childDeptIds = append(childDeptIds, deptId)
			getChildDepts(core, deptId, &childDeptIds)

			return db.Where(fmt.Sprintf("%s IN ?", deptField), childDeptIds)
		}
		return db.Where("1=0")
	case models.DataScopeSelf:
		// 仅本人数据权限
		return db.Where(fmt.Sprintf("%s = ?", deptField), userID)
	default:
		return db.Where("1=0") // 默认无权限
	}
}

// getChildDepts 递归获取子部门
func getChildDepts(core *core.Core, parentId string, deptIds *[]string) {
	var childDepts []struct {
		ID string `gorm:"column:id"`
	}

	core.DB.GetDB().Table("sys_dept").Where("parent_id = ?", parentId).Select("id").Scan(&childDepts)

	for _, dept := range childDepts {
		*deptIds = append(*deptIds, dept.ID)
		getChildDepts(core, dept.ID, deptIds)
	}
}