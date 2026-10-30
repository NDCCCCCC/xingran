package migrations

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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
// F 型声明块（由 ruoyi-react-frontend/scripts/check-permission-codes.mjs 解析）：
// 块内所有权限码字面量按 F 型按钮权限计入 menuFType，不再只当 untyped 字面量。
// 必须与下方 actionPermissionSeeds 保持 1:1 严格一致。
// 父 perms 必须留在块外，避免被误判为 F 型按钮。
// ============================================================================
// F_TYPE_SEED_BEGIN
var actionPermissionFTypeSeedCodes = []string{
	// 试点码（87-01 tracer）：角色新增按钮
	"system:role:add",
}
// F_TYPE_SEED_END

// actionPermissionSeeds 动作级权限种子表（87-01 仅试点码，87-02 铺开第一批 65 条）。
var actionPermissionSeeds = []actionPermissionSeed{
	{
		name:        "角色新增",
		perms:       "system:role:add",
		parentPerms: "system:role",
		orderNum:    1,
	},
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
