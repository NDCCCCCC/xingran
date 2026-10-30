package migrations

import (
	"errors"
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// 经理同步两个管理页的菜单/按钮权限 seed（2026-10-22 通用经理同步方案）。
//
// 权限标识 **单一事实源**：与 internal/api/v1/system/manager_rule_router.go 的
// RequirePermissions 参数、前端 managerApi.ts 消费面三处逐字一致——
// 任一处漂移都会让普通角色恒 403（migration_218 头注释记载的历史坑）。
const (
	managerRuleListPerms   = "ops:manager:rule:list"
	managerRuleEditPerms   = "ops:manager:rule:edit"
	managerRuleDeletePerms = "ops:manager:rule:delete"
	managerOrgListPerms    = "ops:manager:org:list"
	managerOrgEditPerms    = "ops:manager:org:edit"
	managerOrgDeletePerms  = "ops:manager:org:delete"

	// managerSyncSystemRootFallbackName 系统管理根菜单名兜底（ID 用 migration_207 的
	// canonicalSystemRootID 常量，同包直接复用，不重复声明）。
	managerSyncSystemRootFallbackName = "系统管理"
)

// managerSyncMenuSeed 一条 C 菜单 + 其 F 按钮的定义。
type managerSyncMenuSeed struct {
	menuName  string
	path      string
	component string
	listPerms string
	orderNum  int
	buttons   []managerSyncButtonSeed
}

type managerSyncButtonSeed struct {
	menuName string
	perms    string
	orderNum int
}

// Migrate221ManagerSyncMenus 经理同步「规则维护」「机构维护」菜单 + 按钮权限。
//
// 设计：
//   - 两个 C 型路由菜单挂在「系统管理」根下（component 与前端
//     src/pages/system/manager-{rules,orgs}/index.tsx 对应，
//     componentLoader.tsx 按 import.meta.glob 懒加载）。
//   - 每页 3 个 F 按钮（list 权限落在 C 菜单 perms 上；edit/delete 独立 F 行），
//     与 manager_rule_router.go 的 RequirePermissions 分组一一对应。
//   - 幂等：按 menu_name count-then-insert（migration_195 风格），
//     F 行按 perms 先查后建（migration_218 风格）。
//   - 精准授权（migration_218 风格）：只授予已关联「系统管理」根菜单的角色，
//     NOT EXISTS 兜底可重复执行；根菜单缺失 → log warning 非阻断降级
//     （admin 走超管旁路立即可见，普通角色由管理员手动授权）。
func Migrate221ManagerSyncMenus(db *gorm.DB) error {
	log.Println("Running migration 221: manager sync menus + button permissions")

	// 1) 定位「系统管理」根菜单：先按 migration_207 规范 ID，再按名字兜底
	var parent models.Menu
	err := db.Where("id = ?", canonicalSystemRootID).First(&parent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Where("menu_name = ? AND menu_type = ?", managerSyncSystemRootFallbackName, "M").First(&parent).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Warning: 未找到「系统管理」根菜单, migration_221 跳过菜单与授权 "+
				"(部署清单须管理员手动创建菜单并授权): %v", err)
			return nil
		}
		return fmt.Errorf("查询系统管理根菜单失败: %w", err)
	}
	parentID := parent.ID

	seeds := []managerSyncMenuSeed{
		{
			menuName:  "经理同步规则",
			path:      "manager-rules",
			component: "system/manager-rules/index",
			listPerms: managerRuleListPerms,
			orderNum:  90,
			buttons: []managerSyncButtonSeed{
				{menuName: "规则新增/编辑", perms: managerRuleEditPerms, orderNum: 1},
				{menuName: "规则删除", perms: managerRuleDeletePerms, orderNum: 2},
			},
		},
		{
			menuName:  "经理同步机构",
			path:      "manager-orgs",
			component: "system/manager-orgs/index",
			listPerms: managerOrgListPerms,
			orderNum:  91,
			buttons: []managerSyncButtonSeed{
				{menuName: "机构新增/编辑", perms: managerOrgEditPerms, orderNum: 1},
				{menuName: "机构删除", perms: managerOrgDeletePerms, orderNum: 2},
			},
		},
	}

	for _, seed := range seeds {
		menuID, err := ensureManagerSyncMenu(db, seed, parentID)
		if err != nil {
			return err
		}
		// 精准授权 C 菜单
		if err := grantMenuToParentRoles(db, menuID, parentID); err != nil {
			return err
		}
		for _, btn := range seed.buttons {
			btnID, err := ensureManagerSyncButton(db, btn, menuID)
			if err != nil {
				return err
			}
			if err := grantMenuToParentRoles(db, btnID, parentID); err != nil {
				return err
			}
		}
	}

	applogger.Infof("[迁移] 经理同步菜单 seed 完成 (parent=%s)", parentID)
	log.Println("Migration 221 completed: manager sync menus ready")
	return nil
}

// ensureManagerSyncMenu 幂等创建/复用一个 C 菜单，返回菜单 ID。
func ensureManagerSyncMenu(db *gorm.DB, seed managerSyncMenuSeed, parentID string) (string, error) {
	var count int64
	if err := db.Table("sys_menu").Where("menu_name = ?", seed.menuName).Count(&count).Error; err != nil {
		return "", fmt.Errorf("查询菜单 %s 失败: %w", seed.menuName, err)
	}
	if count > 0 {
		var existing models.Menu
		if err := db.Where("menu_name = ?", seed.menuName).First(&existing).Error; err != nil {
			return "", fmt.Errorf("读取既有菜单 %s 失败: %w", seed.menuName, err)
		}
		return existing.ID, nil
	}

	path := seed.path
	component := seed.component
	perms := seed.listPerms
	icon := "#"
	menu := &models.Menu{
		MenuName:  seed.menuName,
		ParentID:  &parentID,
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeMenu, // 'C' 路由菜单
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
		Icon:      &icon,
		OrderNum:  seed.orderNum,
		Remark:    "通用经理同步（部门联系人/岗位→AD manager 属性）管理页",
	}
	if err := db.Create(menu).Error; err != nil {
		return "", fmt.Errorf("创建菜单 %s 失败: %w", seed.menuName, err)
	}
	log.Printf("Created menu: %s (ID: %s)", seed.menuName, menu.ID)
	return menu.ID, nil
}

// ensureManagerSyncButton 幂等创建/复用一个 F 按钮权限行，返回菜单 ID。
func ensureManagerSyncButton(db *gorm.DB, btn managerSyncButtonSeed, parentMenuID string) (string, error) {
	var existing models.Menu
	err := db.Where("perms = ?", btn.perms).First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("查询按钮权限 %s 失败: %w", btn.perms, err)
	}

	emptyPath := ""
	icon := "#"
	perms := btn.perms
	button := &models.Menu{
		MenuName: btn.menuName,
		ParentID: &parentMenuID,
		Path:     &emptyPath,
		MenuType: models.MenuTypeButton,
		Visible:  models.VisibleHidden,
		Status:   models.MenuStatusNormal,
		Perms:    &perms,
		Icon:     &icon,
		OrderNum: btn.orderNum,
	}
	if err := db.Create(button).Error; err != nil {
		return "", fmt.Errorf("创建按钮权限 %s 失败: %w", btn.perms, err)
	}
	log.Printf("Created button permission: %s (ID: %s)", btn.perms, button.ID)
	return button.ID, nil
}

// grantMenuToParentRoles 精准授权：只授予已关联父菜单的角色（NOT EXISTS 幂等）。
func grantMenuToParentRoles(db *gorm.DB, menuID, parentID string) error {
	if err := db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id)
		 SELECT rm.role_id, ? FROM sys_role_menu rm
		 WHERE rm.menu_id = ?
		   AND NOT EXISTS (
		       SELECT 1 FROM sys_role_menu x
		       WHERE x.role_id = rm.role_id AND x.menu_id = ?)`,
		menuID, parentID, menuID).Error; err != nil {
		return fmt.Errorf("精准授权菜单 %s 失败: %w", menuID, err)
	}
	return nil
}
