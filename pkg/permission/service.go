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

// assignAllMenusToAdmin 增量幂等补全 admin 缺失菜单，不删除已有；差集为空秒过。
//
// 设计说明（原"先删后全量重插"的定时炸弹修复）：
//   - 不再执行 DELETE sys_role_menu WHERE role_id = admin，消除"已删未插"的丢权限中间窗口。
//   - 仅 Pluck id 列（而非 SELECT * 全表），GORM 对嵌入 BaseModel 的 Menu 自动追加
//     deleted_at IS NULL，软删菜单天然过滤。
//   - 差集为空 → return nil（幂等快速路径，启动加速，不触碰 sys_role_menu 表）。
//   - 差集非空 → 单一事务 CreateInBatches 补缺失，失败回滚不丢已有权限。
//
// 行为等价性差异：不再清理指向已软删菜单的陈旧 role_menu 关联。这些 menu_id 在
// sys_menu 已被软删，GetUserMenus / GetUserPermissions 的 JOIN 带 m.deleted_at IS NULL
// 在读取层天然屏蔽，不授予任何权限；admin 目标本就是"拥有全部现存菜单"，陈旧关联无害。
func (s *service) assignAllMenusToAdmin(db *gorm.DB) error {
	var adminRole models.Role
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	// 全部现存菜单 id（Pluck 仅取 id 列；Menu 嵌入 BaseModel，GORM 自动过滤软删）
	var allMenuIDs []string
	if err := db.Model(&models.Menu{}).Pluck("id", &allMenuIDs).Error; err != nil {
		return err
	}

	// admin 已有的 menu_id
	var existingIDs []string
	if err := db.Model(&models.RoleMenu{}).
		Where("role_id = ?", adminRole.ID).
		Pluck("menu_id", &existingIDs).Error; err != nil {
		return err
	}

	// 构造差集：allMenuIDs 中存在、existingIDs 中缺失的 menu_id（差集天然去重）
	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}

	var missing []models.RoleMenu
	for _, mid := range allMenuIDs {
		if _, ok := existingSet[mid]; !ok {
			missing = append(missing, models.RoleMenu{
				RoleID: adminRole.ID,
				MenuID: mid,
			})
		}
	}

	// 幂等快速路径：admin 已拥有全部现存菜单，秒过
	if len(missing) == 0 {
		return nil
	}

	// 事务包裹批量补全，失败回滚不丢已有权限
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.CreateInBatches(missing, 100).Error
	})
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
