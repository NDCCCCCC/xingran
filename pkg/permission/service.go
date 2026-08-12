package permission

import (
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Service 权限服务接口
type Service interface {
	InitDefaultRolesAndMenus(db *gorm.DB) error
	GetUserMenus(db *gorm.DB, userID string, includeHidden bool) ([]models.Menu, error)
	GetUserPermissions(db *gorm.DB, userID string) ([]string, error)
	GetRoleMenus(db *gorm.DB, roleID string) ([]string, error)
	UpdateRoleMenus(db *gorm.DB, roleID string, menuIDs []string) error
	GetRoleDepts(db *gorm.DB, roleID string) ([]string, error)
	UpdateRoleDepts(db *gorm.DB, roleID string, deptIDs []string) error
}

// service 权限服务实现
type service struct{}

// NewService 创建权限服务
func NewService() Service {
	return &service{}
}

// InitDefaultRolesAndMenus 初始化默认角色和菜单
func (s *service) InitDefaultRolesAndMenus(db *gorm.DB) error {
	// 创建默认超级管理员角色
	if err := s.createDefaultAdminRole(db); err != nil {
		return fmt.Errorf("创建默认管理员角色失败: %w", err)
	}

	// 为超级管理员角色分配所有菜单权限
	if err := s.assignAllMenusToAdmin(db); err != nil {
		return fmt.Errorf("为管理员分配菜单权限失败: %w", err)
	}

	return nil
}

// createDefaultAdminRole 创建默认管理员角色
func (s *service) createDefaultAdminRole(db *gorm.DB) error {
	var count int64
	db.Model(&models.Role{}).Where("role_key = ?", "admin").Count(&count)

	if count == 0 {
		adminRole := models.Role{
			RoleName:          "超级管理员",
			RoleKey:           "admin",
			RoleSort:          1,
			DataScope:         models.DataScopeAll,
			MenuCheckStrictly: true,
			DeptCheckStrictly: true,
			Status:            models.RoleStatusEnabled,
			Remark:            "超级管理员",
		}

		if err := db.Create(&adminRole).Error; err != nil {
			return err
		}
	}

	return nil
}

// assignAllMenusToAdmin 为管理员分配所有菜单权限
func (s *service) assignAllMenusToAdmin(db *gorm.DB) error {
	var adminRole models.Role
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var allMenus []models.Menu
	if err := db.Find(&allMenus).Error; err != nil {
		return err
	}

	// 清除现有的角色菜单关联
	db.Where("role_id = ?", adminRole.ID).Delete(&models.RoleMenu{})

	// 创建新的角色菜单关联
	for _, menu := range allMenus {
		roleMenu := models.RoleMenu{
			RoleID: adminRole.ID,
			MenuID: menu.ID,
		}
		if err := db.Create(&roleMenu).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetRoleMenus 获取角色的菜单权限
func (s *service) GetRoleMenus(db *gorm.DB, roleID string) ([]string, error) {
	var menuIDs []string
	err := db.Model(&models.RoleMenu{}).
		Where("role_id = ?", roleID).
		Pluck("menu_id", &menuIDs).Error

	return menuIDs, err
}

// UpdateRoleMenus 更新角色菜单权限
func (s *service) UpdateRoleMenus(db *gorm.DB, roleID string, menuIDs []string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 删除现有的角色菜单关联
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
			return err
		}

		// 创建新的角色菜单关联
		for _, menuID := range menuIDs {
			roleMenu := models.RoleMenu{
				RoleID: roleID,
				MenuID: menuID,
			}
			if err := tx.Create(&roleMenu).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetRoleDepts 获取角色的数据权限部门
func (s *service) GetRoleDepts(db *gorm.DB, roleID string) ([]string, error) {
	var deptIDs []string
	err := db.Model(&models.RoleDept{}).
		Where("role_id = ?", roleID).
		Pluck("dept_id", &deptIDs).Error

	return deptIDs, err
}

// UpdateRoleDepts 更新角色的数据权限部门
func (s *service) UpdateRoleDepts(db *gorm.DB, roleID string, deptIDs []string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 删除现有的角色部门关联
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleDept{}).Error; err != nil {
			return err
		}

		// 创建新的角色部门关联
		for _, deptID := range deptIDs {
			roleDept := models.RoleDept{
				RoleID: roleID,
				DeptID: deptID,
			}
			if err := tx.Create(&roleDept).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetUserMenus 获取用户的菜单树
// 设计原则：
// - visible=0（隐藏）：不显示在导航栏，但权限有效，可通过直接URL访问
// - status=1（停用）：显示在导航栏但为灰色，点击后提示"功能已停用"，权限无效
//
// 参数：
// - userID: 用户ID
// - includeHidden: 是否包含隐藏菜单（false=只返回visible=1的菜单，true=返回所有菜单）
func (s *service) GetUserMenus(db *gorm.DB, userID string, includeHidden bool) ([]models.Menu, error) {
	// 一次性获取所有可能的菜单（包括用户菜单及其祖先）
	// 使用递归CTE或JOIN避免N+1查询
	var menus []models.Menu

	// 首先获取用户直接有权限的菜单
	var querySQL string
	var queryParams []interface{}

	if includeHidden {
		// 包含隐藏菜单的SQL
		querySQL = `
			WITH RECURSIVE menu_tree AS (
				-- 基础：用户有权限的菜单
				SELECT DISTINCT m.*
				FROM sys_menu m
				INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
				INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
				WHERE ur.user_id = ?
				AND m.menu_type IN ('M', 'C')
				AND m.deleted_at IS NULL

				UNION ALL

				-- 递归：获取所有祖先菜单
				SELECT m.*
				FROM sys_menu m
				INNER JOIN menu_tree mt ON m.id = mt.parent_id
				WHERE m.deleted_at IS NULL
			)
			SELECT DISTINCT * FROM menu_tree
			ORDER BY order_num
		`
		queryParams = []interface{}{userID}
	} else {
		// 不包含隐藏菜单的SQL（原有逻辑）
		querySQL = `
			WITH RECURSIVE menu_tree AS (
				-- 基础：用户有权限的菜单
				SELECT DISTINCT m.*
				FROM sys_menu m
				INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
				INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
				WHERE ur.user_id = ?
				AND m.visible = ?
				AND m.menu_type IN ('M', 'C')
				AND m.deleted_at IS NULL

				UNION ALL

				-- 递归：获取所有祖先菜单
				SELECT m.*
				FROM sys_menu m
				INNER JOIN menu_tree mt ON m.id = mt.parent_id
				WHERE m.visible = ?
				AND m.deleted_at IS NULL
			)
			SELECT DISTINCT * FROM menu_tree
			ORDER BY order_num
		`
		queryParams = []interface{}{userID, models.VisibleShow, models.VisibleShow}
	}

	err := db.Raw(querySQL, queryParams...).Scan(&menus).Error

	if err != nil {
		return nil, err
	}

	// 构建菜单树
	return s.buildMenuTree(menus, ""), nil
}

// buildMenuTree 构建菜单树
func (s *service) buildMenuTree(menus []models.Menu, parentID string) []models.Menu {
	var tree []models.Menu

	for _, menu := range menus {
		var menuParentID string
		if menu.ParentID != nil {
			menuParentID = *menu.ParentID
		}

		if menuParentID == parentID {
			// 递归构建子菜单
			children := s.buildMenuTree(menus, menu.ID)
			if len(children) > 0 {
				menu.Children = children
			}
			tree = append(tree, menu)
		}
	}

	return tree
}

// GetUserPermissions 获取用户的所有权限
// 注意：只检查 status（停用），不检查 visible（隐藏）
// - 隐藏的菜单（visible=0）：不显示在导航栏，但权限仍然有效，可通过直接URL访问
// - 停用的菜单（status=1）：完全不可用，权限无效，即使知道URL也无法访问
func (s *service) GetUserPermissions(db *gorm.DB, userID string) ([]string, error) {
	var permissions []string
	err := db.Raw(`
		SELECT DISTINCT m.perms
		FROM sys_menu m
		INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
		INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
		WHERE ur.user_id = ?
		AND m.perms IS NOT NULL
		AND m.perms != ''
		AND m.status = ?
	`, userID, models.MenuStatusNormal).Scan(&permissions).Error

	return permissions, err
}
