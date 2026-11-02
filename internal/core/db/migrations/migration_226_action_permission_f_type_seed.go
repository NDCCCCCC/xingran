package migrations

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// actionPermissionSeed 定义动作级权限种子的单条结构。
type actionPermissionSeed struct {
	name            string // 菜单显示名称（sys_menu.menu_name）
	perms           string // 权限标识（sys_menu.perms，唯一）
	parentPerms     string // 父 C 型菜单的 perms；若为空则通过 parentName + parentComponent 定位
	parentName      string // 特例父菜单名称（如资产列表）
	parentComponent string // 特例父菜单前端组件路径（进一步精确）
	orderNum        int    // 排序号
}

// ============================================================================
// F 型声明块（由 xingran-react-frontend/scripts/check-permission-codes.mjs 解析）：
// 块内所有权限码字面量按 F 型按钮权限计入 menuFType，不再只当 untyped 字面量。
// 必须与下方 actionPermissionSeeds 保持 1:1 严格一致（逐条相等且数量一致）。
// 父 perms 必须留在块外，避免被误判为 F 型按钮。
// ============================================================================
// F_TYPE_SEED_BEGIN
var actionPermissionFTypeSeedCodes = []string{
	// 1. system:role (6 条)
	"system:role:list",
	"system:role:view",
	"system:role:add",
	"system:role:edit",
	"system:role:remove",
	"system:role:export",

	// 2. system:menu (5 条)
	"system:menu:list",
	"system:menu:view",
	"system:menu:add",
	"system:menu:edit",
	"system:menu:remove",

	// 3. system:dept (7 条)
	"system:dept:list",
	"system:dept:view",
	"system:dept:add",
	"system:dept:edit",
	"system:dept:remove",
	"system:dept:import",
	"system:dept:export",

	// 4. system:post (5 条)
	"system:post:list",
	"system:post:view",
	"system:post:add",
	"system:post:edit",
	"system:post:remove",

	// 5. ops:building (2 条)
	"ops:building:import",
	"ops:building:export",

	// 6. ops:floor (2 条)
	"ops:floor:import",
	"ops:floor:export",

	// 7. ops:workstation (2 条)
	"ops:workstation:import",
	"ops:workstation:export",

	// 8. ops:asset (2 条 - 特例父菜单)
	"ops:asset:import",
	"ops:asset:export",

	// 9. ops:infopoint (2 条)
	"ops:infopoint:import",
	"ops:infopoint:export",

	// 10. ops:serverroom (2 条)
	"ops:serverroom:import",
	"ops:serverroom:export",

	// 11. ops:dedicatedline (2 条)
	"ops:dedicatedline:import",
	"ops:dedicatedline:export",

	// 12. ops:roomdevice (2 条)
	"ops:roomdevice:import",
	"ops:roomdevice:export",

	// 13. network:device (6 条)
	"network:device:list",
	"network:device:view",
	"network:device:add",
	"network:device:edit",
	"network:device:delete",
	"network:device:export",

	// 14. network:credential (6 条)
	"network:credential:list",
	"network:credential:view",
	"network:credential:add",
	"network:credential:edit",
	"network:credential:delete",
	"network:credential:export",

	// 15. network:template (6 条)
	"network:template:list",
	"network:template:view",
	"network:template:add",
	"network:template:edit",
	"network:template:delete",
	"network:template:export",

	// 16. network:command (2 条 - 多父集合)
	"network:command:view",
	"network:command:export",

	// 17. network:backup (3 条)
	"network:backup:list",
	"network:backup:delete",
	"network:backup:export",

	// 18. network:discovery (3 条)
	"network:discovery:view",
	"network:discovery:delete",
	"network:discovery:export",
}
// F_TYPE_SEED_END

// actionPermissionSeeds 动作级权限种子表（第一批 65 条）。
var actionPermissionSeeds = []actionPermissionSeed{
	// 1. system:role (6 条)
	{name: "角色查询", perms: "system:role:list", parentPerms: "system:role", orderNum: 1},
	{name: "角色详情", perms: "system:role:view", parentPerms: "system:role", orderNum: 2},
	{name: "角色新增", perms: "system:role:add", parentPerms: "system:role", orderNum: 3},
	{name: "角色修改", perms: "system:role:edit", parentPerms: "system:role", orderNum: 4},
	{name: "角色删除", perms: "system:role:remove", parentPerms: "system:role", orderNum: 5},
	{name: "角色导出", perms: "system:role:export", parentPerms: "system:role", orderNum: 6},

	// 2. system:menu (5 条)
	{name: "菜单查询", perms: "system:menu:list", parentPerms: "system:menu", orderNum: 1},
	{name: "菜单详情", perms: "system:menu:view", parentPerms: "system:menu", orderNum: 2},
	{name: "菜单新增", perms: "system:menu:add", parentPerms: "system:menu", orderNum: 3},
	{name: "菜单修改", perms: "system:menu:edit", parentPerms: "system:menu", orderNum: 4},
	{name: "菜单删除", perms: "system:menu:remove", parentPerms: "system:menu", orderNum: 5},

	// 3. system:dept (7 条)
	{name: "部门查询", perms: "system:dept:list", parentPerms: "system:dept", orderNum: 1},
	{name: "部门详情", perms: "system:dept:view", parentPerms: "system:dept", orderNum: 2},
	{name: "部门新增", perms: "system:dept:add", parentPerms: "system:dept", orderNum: 3},
	{name: "部门修改", perms: "system:dept:edit", parentPerms: "system:dept", orderNum: 4},
	{name: "部门删除", perms: "system:dept:remove", parentPerms: "system:dept", orderNum: 5},
	{name: "部门导入", perms: "system:dept:import", parentPerms: "system:dept", orderNum: 6},
	{name: "部门导出", perms: "system:dept:export", parentPerms: "system:dept", orderNum: 7},

	// 4. system:post (5 条)
	{name: "岗位查询", perms: "system:post:list", parentPerms: "system:post", orderNum: 1},
	{name: "岗位详情", perms: "system:post:view", parentPerms: "system:post", orderNum: 2},
	{name: "岗位新增", perms: "system:post:add", parentPerms: "system:post", orderNum: 3},
	{name: "岗位修改", perms: "system:post:edit", parentPerms: "system:post", orderNum: 4},
	{name: "岗位删除", perms: "system:post:remove", parentPerms: "system:post", orderNum: 5},

	// 5. ops:building (2 条)
	{name: "楼宇导入", perms: "ops:building:import", parentPerms: "ops:building:list", orderNum: 10},
	{name: "楼宇导出", perms: "ops:building:export", parentPerms: "ops:building:list", orderNum: 11},

	// 6. ops:floor (2 条)
	{name: "楼层导入", perms: "ops:floor:import", parentPerms: "ops:floor:list", orderNum: 10},
	{name: "楼层导出", perms: "ops:floor:export", parentPerms: "ops:floor:list", orderNum: 11},

	// 7. ops:workstation (2 条)
	{name: "工位导入", perms: "ops:workstation:import", parentPerms: "ops:workstation:list", orderNum: 10},
	{name: "工位导出", perms: "ops:workstation:export", parentPerms: "ops:workstation:list", orderNum: 11},

	// 8. ops:asset (2 条 - 特例父菜单，perms 为 NULL，通过 parentName + parentComponent 定位)
	{name: "资产导入", perms: "ops:asset:import", parentName: "资产列表", parentComponent: "operations/assets/index", orderNum: 10},
	{name: "资产导出", perms: "ops:asset:export", parentName: "资产列表", parentComponent: "operations/assets/index", orderNum: 11},

	// 9. ops:infopoint (2 条)
	{name: "信息点导入", perms: "ops:infopoint:import", parentPerms: "ops:infopoint:list", orderNum: 10},
	{name: "信息点导出", perms: "ops:infopoint:export", parentPerms: "ops:infopoint:list", orderNum: 11},

	// 10. ops:serverroom (2 条)
	{name: "机房导入", perms: "ops:serverroom:import", parentPerms: "ops:serverroom:list", orderNum: 10},
	{name: "机房导出", perms: "ops:serverroom:export", parentPerms: "ops:serverroom:list", orderNum: 11},

	// 11. ops:dedicatedline (2 条)
	{name: "专线导入", perms: "ops:dedicatedline:import", parentPerms: "ops:dedicatedline:list", orderNum: 10},
	{name: "专线导出", perms: "ops:dedicatedline:export", parentPerms: "ops:dedicatedline:list", orderNum: 11},

	// 12. ops:roomdevice (2 条)
	{name: "机房设备导入", perms: "ops:roomdevice:import", parentPerms: "ops:roomdevice:list", orderNum: 10},
	{name: "机房设备导出", perms: "ops:roomdevice:export", parentPerms: "ops:roomdevice:list", orderNum: 11},

	// 13. network:device (6 条)
	{name: "设备查询", perms: "network:device:list", parentPerms: "network:device", orderNum: 1},
	{name: "设备详情", perms: "network:device:view", parentPerms: "network:device", orderNum: 2},
	{name: "设备新增", perms: "network:device:add", parentPerms: "network:device", orderNum: 3},
	{name: "设备修改", perms: "network:device:edit", parentPerms: "network:device", orderNum: 4},
	{name: "设备删除", perms: "network:device:delete", parentPerms: "network:device", orderNum: 5},
	{name: "设备导出", perms: "network:device:export", parentPerms: "network:device", orderNum: 6},

	// 14. network:credential (6 条)
	{name: "凭据查询", perms: "network:credential:list", parentPerms: "network:credential", orderNum: 1},
	{name: "凭据详情", perms: "network:credential:view", parentPerms: "network:credential", orderNum: 2},
	{name: "凭据新增", perms: "network:credential:add", parentPerms: "network:credential", orderNum: 3},
	{name: "凭据修改", perms: "network:credential:edit", parentPerms: "network:credential", orderNum: 4},
	{name: "凭据删除", perms: "network:credential:delete", parentPerms: "network:credential", orderNum: 5},
	{name: "凭据导出", perms: "network:credential:export", parentPerms: "network:credential", orderNum: 6},

	// 15. network:template (6 条)
	{name: "模板查询", perms: "network:template:list", parentPerms: "network:template", orderNum: 1},
	{name: "模板详情", perms: "network:template:view", parentPerms: "network:template", orderNum: 2},
	{name: "模板新增", perms: "network:template:add", parentPerms: "network:template", orderNum: 3},
	{name: "模板修改", perms: "network:template:edit", parentPerms: "network:template", orderNum: 4},
	{name: "模板删除", perms: "network:template:delete", parentPerms: "network:template", orderNum: 5},
	{name: "模板导出", perms: "network:template:export", parentPerms: "network:template", orderNum: 6},

	// 16. network:command (2 条 - 命令分发/配置执行多父集合)
	{name: "命令查看", perms: "network:command:view", parentPerms: "network:command", orderNum: 1},
	{name: "命令导出", perms: "network:command:export", parentPerms: "network:command", orderNum: 2},

	// 17. network:backup (3 条)
	{name: "备份查询", perms: "network:backup:list", parentPerms: "network:backup", orderNum: 1},
	{name: "备份删除", perms: "network:backup:delete", parentPerms: "network:backup", orderNum: 2},
	{name: "备份导出", perms: "network:backup:export", parentPerms: "network:backup", orderNum: 3},

	// 18. network:discovery (3 条)
	{name: "发现查看", perms: "network:discovery:view", parentPerms: "network:discovery", orderNum: 1},
	{name: "发现删除", perms: "network:discovery:delete", parentPerms: "network:discovery", orderNum: 2},
	{name: "发现导出", perms: "network:discovery:export", parentPerms: "network:discovery", orderNum: 3},
}

// Migrate226ActionPermissionFTypeSeed 播种动作级 F 型菜单并精准授权给持父菜单的角色。
// 双方言（PG / sqlite 通用）、幂等、非阻断降级。
func Migrate226ActionPermissionFTypeSeed(db *gorm.DB) error {
	log.Println("Running migration 226: action-level F type permission seed")

	// 0) 入口硬断言：声明块与种子表必须条数相同且逐条相等
	if len(actionPermissionFTypeSeedCodes) != len(actionPermissionSeeds) {
		return fmt.Errorf("migration 226 致命漂移: F_TYPE_SEED 块声明数(%d) != 种子表数(%d)",
			len(actionPermissionFTypeSeedCodes), len(actionPermissionSeeds))
	}
	for i, code := range actionPermissionFTypeSeedCodes {
		if code != actionPermissionSeeds[i].perms {
			return fmt.Errorf("migration 226 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, actionPermissionSeeds[i].perms)
		}
	}

	// 逐条处理种子（单条失败仅记录日志并继续，保证非阻断）
	for _, seed := range actionPermissionSeeds {
		// 1) 解析父 C 型菜单（必须是 menu_type='C' AND status=0 的 live 菜单）
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 226: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 226: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		// 2) 幂等建立 F 型按钮行
		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 226: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		// 3) 精准授权给持有任一父 C 菜单的角色
		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 226: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 226 completed: action-level F type permissions seeded")
	return nil
}

// resolveActionPermissionParents 解析 live 状态的父 C 型菜单。
// 必须过滤 menu_type = 'C' AND status = 0，排查掉停用旧树副本（防越权）。
func resolveActionPermissionParents(db *gorm.DB, seed actionPermissionSeed) ([]models.Menu, error) {
	var parents []models.Menu
	q := db.Where("menu_type = ? AND status = ?", models.MenuTypeMenu, models.MenuStatusNormal)

	if seed.parentPerms != "" {
		q = q.Where("perms = ?", seed.parentPerms)
	} else if seed.parentName != "" {
		// 特例场景（如 ops:asset 的父「资产列表」perms 为 NULL）
		q = q.Where("menu_name = ?", seed.parentName)
		if seed.parentComponent != "" {
			q = q.Where("component = ?", seed.parentComponent)
		}
	} else {
		return nil, errors.New("种子未指定 parentPerms 也未指定 parentName 特例")
	}

	if err := q.Order("id ASC").Find(&parents).Error; err != nil {
		return nil, err
	}
	return parents, nil
}

// ensureActionPermissionMenu 幂等创建或复用 F 型按钮行。
func ensureActionPermissionMenu(db *gorm.DB, seed actionPermissionSeed, primaryParentID string) (string, error) {
	var existing models.Menu
	// 按 perms + menu_type='F' 查找，精确识别既有按钮行
	err := db.Where("perms = ? AND menu_type = ?", seed.perms, models.MenuTypeButton).First(&existing).Error
	if err == nil {
		// 既有行存在，复用 ID
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("查询既有按钮菜单失败: %w", err)
	}

	emptyPath := ""
	icon := "#"
	perms := seed.perms
	button := &models.Menu{
		MenuName: seed.name,
		ParentID: &primaryParentID,
		Path:     &emptyPath,
		MenuType: models.MenuTypeButton,
		Visible:  models.VisibleHidden,
		Status:   models.MenuStatusNormal,
		Perms:    &perms,
		Icon:     &icon,
		OrderNum: seed.orderNum,
	}

	if err := db.Create(button).Error; err != nil {
		return "", fmt.Errorf("创建按钮权限行失败: %w", err)
	}

	// GORM 在 models.Menu.Visible 带 default:1 tag 时会对零值 VisibleHidden(0) 省略插入列导致落库为 1
	// 显式 Update 强制将 visible 归零，遵循 AGENTS.md（visible 1=visible, 0=hidden）
	if err := db.Model(&models.Menu{}).Where("id = ?", button.ID).Update("visible", models.VisibleHidden).Error; err != nil {
		return "", fmt.Errorf("显式更新按钮 visible=0 失败: %w", err)
	}

	log.Printf("Created action button permission: %s (ID: %s, perms: %s)", seed.name, button.ID, seed.perms)
	return button.ID, nil
}

// grantActionPermissionToParentHolders 将 menuID 精准授予所有持有任一 parentIDs 的角色（NOT EXISTS 幂等）。
// 双方言（PG / sqlite 通用），禁止拼 SQL，禁止依赖唯一约束。
func grantActionPermissionToParentHolders(db *gorm.DB, menuID string, parentIDs []string) error {
	if len(parentIDs) == 0 {
		return nil
	}

	if len(parentIDs) == 1 {
		return db.Exec(
			`INSERT INTO sys_role_menu (role_id, menu_id)
			 SELECT rm.role_id, ? FROM sys_role_menu rm
			 WHERE rm.menu_id = ?
			   AND NOT EXISTS (
			       SELECT 1 FROM sys_role_menu x
			       WHERE x.role_id = rm.role_id AND x.menu_id = ?)`,
			menuID, parentIDs[0], menuID).Error
	}

	return db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id)
		 SELECT DISTINCT rm.role_id, ? FROM sys_role_menu rm
		 WHERE rm.menu_id IN ?
		   AND NOT EXISTS (
		       SELECT 1 FROM sys_role_menu x
		       WHERE x.role_id = rm.role_id AND x.menu_id = ?)`,
		menuID, parentIDs, menuID).Error
}
