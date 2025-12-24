package permission

import (
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// Service 权限服务接口
type Service interface {
	InitDefaultRolesAndMenus(db *gorm.DB) error
	GetUserMenus(db *gorm.DB, userID string) ([]models.Menu, error)
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

	// 创建默认菜单
	if err := s.createDefaultMenus(db); err != nil {
		return fmt.Errorf("创建默认菜单失败: %w", err)
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
			RoleName:         "超级管理员",
			RoleKey:          "admin",
			RoleSort:         1,
			DataScope:        models.DataScopeAll,
			MenuCheckStrictly: true,
			DeptCheckStrictly: true,
			Status:           models.RoleStatusEnabled,
			Remark:           "超级管理员",
		}

		if err := db.Create(&adminRole).Error; err != nil {
			return err
		}
	}

	return nil
}

// createDefaultMenus 创建默认菜单
func (s *service) createDefaultMenus(db *gorm.DB) error {
	// 1. 创建仪表盘菜单（顶级菜单）
	dashboardMenu := models.Menu{
		MenuName:  "仪表盘",
		OrderNum:  0,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Path:      func() *string { s := "/dashboard"; return &s }(),
		Component: func() *string { s := "dashboard/index"; return &s }(),
		Icon:      func() *string { s := "dashboard"; return &s }(),
		Perms:     func() *string { s := string(System); return &s }(),
	}

	s.createMenuIfNotExists(db, &dashboardMenu)

	// 2. 创建系统管理菜单（父菜单）
	systemMenu := models.Menu{
		MenuName: "系统管理",
		OrderNum: 1,
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
		Icon:     func() *string { s := "setting"; return &s }(),
		Perms:    func() *string { s := string(System); return &s }(),
	}

	s.createMenuIfNotExists(db, &systemMenu)
	if systemMenu.ID == "" {
		db.Where("menu_name = ? AND path IS NULL", systemMenu.MenuName).First(&systemMenu)
	}

	// 系统管理子菜单
	systemChildMenus := []models.Menu{
		{
			MenuName:  "用户管理",
			OrderNum:  1,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &systemMenu.ID,
			Path:      func() *string { s := "/system/user"; return &s }(),
			Component: func() *string { s := "system/user/index"; return &s }(),
			Icon:      func() *string { s := "user"; return &s }(),
			Perms:     func() *string { s := string(SystemUser); return &s }(),
		},
		{
			MenuName:  "角色管理",
			OrderNum:  2,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &systemMenu.ID,
			Path:      func() *string { s := "/system/role"; return &s }(),
			Component: func() *string { s := "system/role/index"; return &s }(),
			Icon:      func() *string { s := "users"; return &s }(),
			Perms:     func() *string { s := string(SystemRole); return &s }(),
		},
		{
			MenuName:  "菜单管理",
			OrderNum:  3,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &systemMenu.ID,
			Path:      func() *string { s := "/system/menu"; return &s }(),
			Component: func() *string { s := "system/menu/index"; return &s }(),
			Icon:      func() *string { s := "menu"; return &s }(),
			Perms:     func() *string { s := string(SystemMenu); return &s }(),
		},
		{
			MenuName:  "部门管理",
			OrderNum:  4,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &systemMenu.ID,
			Path:      func() *string { s := "/system/dept"; return &s }(),
			Component: func() *string { s := "system/dept/index"; return &s }(),
			Icon:      func() *string { s := "users"; return &s }(),
			Perms:     func() *string { s := string(SystemDept); return &s }(),
		},
	}

	for _, menu := range systemChildMenus {
		s.createMenuIfNotExists(db, &menu)
	}

	// 3. 创建系统监控菜单（父菜单）
	monitorMenu := models.Menu{
		MenuName: "系统监控",
		OrderNum: 2,
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
		Icon:     func() *string { s := "monitor"; return &s }(),
		Perms:    func() *string { s := string(Monitor); return &s }(),
	}

	s.createMenuIfNotExists(db, &monitorMenu)
	if monitorMenu.ID == "" {
		db.Where("menu_name = ? AND path IS NULL", monitorMenu.MenuName).First(&monitorMenu)
	}

	// 系统监控子菜单
	monitorChildMenus := []models.Menu{
		{
			MenuName:  "监控仪表盘",
			OrderNum:  1,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &monitorMenu.ID,
			Path:      func() *string { s := "/monitor/dashboard"; return &s }(),
			Component: func() *string { s := "monitor/dashboard/index"; return &s }(),
			Icon:      func() *string { s := "dashboard"; return &s }(),
			Perms:     func() *string { s := string(Monitor); return &s }(),
		},
		{
			MenuName:  "服务监控",
			OrderNum:  2,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &monitorMenu.ID,
			Path:      func() *string { s := "/monitor/server"; return &s }(),
			Component: func() *string { s := "monitor/server/index"; return &s }(),
			Icon:      func() *string { s := "server"; return &s }(),
			Perms:     func() *string { s := string(MonitorServer); return &s }(),
		},
		{
			MenuName:  "日志管理",
			OrderNum:  3,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &monitorMenu.ID,
			Path:      func() *string { s := "/monitor/logs"; return &s }(),
			Component: func() *string { s := "monitor/logs/index"; return &s }(),
			Icon:      func() *string { s := "history"; return &s }(),
			Perms:     func() *string { s := string(MonitorLog); return &s }(),
		},
		{
			MenuName:  "定时任务",
			OrderNum:  4,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &monitorMenu.ID,
			Path:      func() *string { s := "/monitor/job"; return &s }(),
			Component: func() *string { s := "monitor/job/index"; return &s }(),
			Icon:      func() *string { s := "clock"; return &s }(),
			Perms:     func() *string { s := string(MonitorJob); return &s }(),
		},
		{
			MenuName:  "缓存管理",
			OrderNum:  5,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			ParentID:  &monitorMenu.ID,
			Path:      func() *string { s := "/monitor/cache"; return &s }(),
			Component: func() *string { s := "monitor/cache/index"; return &s }(),
			Icon:      func() *string { s := "database"; return &s }(),
			Perms:     func() *string { s := string(MonitorCache); return &s }(),
		},
	}

	for _, menu := range monitorChildMenus {
		s.createMenuIfNotExists(db, &menu)
	}

	return nil
}

// createMenuIfNotExists 如果菜单不存在则创建
func (s *service) createMenuIfNotExists(db *gorm.DB, menu *models.Menu) {
	var count int64
	db.Model(&models.Menu{}).Where("menu_name = ? AND COALESCE(path, '') = COALESCE(?, '')", menu.MenuName, menu.Path).Count(&count)

	if count == 0 {
		db.Create(menu)
	}
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
func (s *service) GetUserMenus(db *gorm.DB, userID string) ([]models.Menu, error) {
	var menus []models.Menu

	err := db.Raw(`
		SELECT DISTINCT m.*
		FROM sys_menu m
		INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
		INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
		WHERE ur.user_id = ?
		AND m.status = ?
		AND m.visible = ?
		ORDER BY m.order_num
	`, userID, models.MenuStatusNormal, models.VisibleShow).Scan(&menus).Error

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
		AND m.visible = ?
	`, userID, models.MenuStatusNormal, models.VisibleShow).Scan(&permissions).Error

	return permissions, err
}