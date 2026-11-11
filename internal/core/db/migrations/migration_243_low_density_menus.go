package migrations

// Migrate243LowDensityMenus —— 低密度页面 Tab 化后的菜单收敛。
//
// 背景：三个「内容单薄却独占菜单」的页面已并入宿主页面作 Tab，前端独立页面目录
// 同步删除（feat-frontend-settings-consolidation）：
//
//	① pages/duty/holidays        → pages/duty/management（「节假日管理」Tab）
//	② pages/duty/config          → pages/duty/management（「值班配置」Tab）
//	③ pages/workorder/categories → pages/workorder/orders（「工单分类」Tab）
//
// 页面没了但菜单还在，用户点进去就是空白页/加载失败，故必须收敛菜单。
//
// 三件事（全部幂等；菜单不存在即跳过，绝不新建）：
//   - 隐藏上述 C 型菜单：visible=0（隐藏，前端 Sidebar 以 visible===1 过滤）、
//     status=1（停用，与 canonical seed 里 duty/pools、duty/holidays 的既有口径一致）。
//     保留行与 sys_role_menu 关联不动 —— 不丢授权历史，回滚只需把 visible/status 改回。
//   - 把这些菜单下的 F 型按钮权限改挂宿主 C 菜单（parent_id 迁移，perms 原值不变），
//     避免角色授权树里出现「父菜单已隐藏、按钮孤儿」的分支。
//   - 不触碰 meta JSONB：前端路由/侧边栏的隐藏判定均基于 visible，与 meta.hidden 无关
//     （routeGenerator.ts / Sidebar.tsx），改 meta 反而引入 PG/SQLite 方言差异。
//
// 双方言：纯 GORM 条件更新，无 ::text / ON CONFLICT / gen_random_uuid() 等 PG 专属语法。
// 非阻断：调用方 WARN，下次启动重试。

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// lowDensityMenuMigration 描述一个「被合并页面的菜单 → 宿主页面菜单」的收敛关系。
type lowDensityMenuMigration struct {
	// name 仅用于日志与错误信息（中文页面名）。
	name string
	// pageComponents 被合并页面在 sys_menu.component 里的全部可能写法
	// （历史数据存在无前导斜杠/带前导斜杠两种形态，故列全部变体）。
	pageComponents []string
	// hostComponents 宿主页面 component；为空表示无宿主可挂（只隐藏、不改挂 F）。
	hostComponents []string
}

var lowDensityMenuMigrations = []lowDensityMenuMigration{
	{
		name:           "节假日管理（并入值班管理「节假日管理」Tab）",
		pageComponents: []string{"duty/holidays/index", "/duty/holidays/index"},
		hostComponents: []string{"duty/management/index", "/duty/management/index"},
	},
	{
		name:           "值班配置（并入值班管理「值班配置」Tab）",
		pageComponents: []string{"duty/config/index", "/duty/config/index"},
		hostComponents: []string{"duty/management/index", "/duty/management/index"},
	},
	{
		name:           "工单分类（并入工单管理「工单分类」Tab）",
		pageComponents: []string{"workorder/categories/index", "/workorder/categories/index"},
		hostComponents: []string{"workorder/orders/index", "/workorder/orders/index"},
	},
}

// Migrate243LowDensityMenus 隐藏低密度页面菜单，并把其 F 型按钮权限改挂宿主菜单（双方言 / 幂等）。
func Migrate243LowDensityMenus(db *gorm.DB) error {
	log.Println("Running migration 243: low density page menus consolidation")

	if !db.Migrator().HasTable("sys_menu") {
		// 极端场景（全新库在 207 seed 之前）：无表可收敛，跳过且不算失败。
		log.Println("[迁移] migration 243 跳过：sys_menu 不存在")
		return nil
	}

	for _, item := range lowDensityMenuMigrations {
		if err := hideMergedPageMenu(db, item); err != nil {
			return fmt.Errorf("migration 243 %s: %w", item.name, err)
		}
	}

	log.Println("[迁移] migration 243 完成：低密度页面菜单已收敛")
	return nil
}

// hideMergedPageMenu 处理单个合并项：隐藏页面菜单 + F 子权限改挂宿主。
func hideMergedPageMenu(db *gorm.DB, item lowDensityMenuMigration) error {
	// 1) 定位被合并的页面菜单（C 型，未软删）
	var pageMenus []models.Menu
	if err := db.Where("component IN ?", item.pageComponents).
		Where("menu_type = ?", models.MenuTypeMenu).
		Find(&pageMenus).Error; err != nil {
		return fmt.Errorf("查询页面菜单失败: %w", err)
	}
	if len(pageMenus) == 0 {
		// 存量库可能本就没有该菜单（如从未播过），或已被人工删除 —— 跳过即幂等。
		log.Printf("[迁移] migration 243 跳过「%s」：sys_menu 中无对应页面菜单", item.name)
		return nil
	}

	// 2) 定位宿主菜单（F 权限改挂目标）；找不到就只隐藏、不改挂
	hostID, hostErr := findHostMenuID(db, item.hostComponents)
	if hostErr != nil && !errors.Is(hostErr, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询宿主菜单失败: %w", hostErr)
	}

	for i := range pageMenus {
		menu := &pageMenus[i]

		// 3) F 型按钮权限改挂宿主（perms 原值保持不变）
		if hostID != "" && hostID != menu.ID {
			moved, err := reparentButtonPermissions(db, menu.ID, hostID)
			if err != nil {
				return fmt.Errorf("改挂 F 型权限失败(menu=%s): %w", menu.ID, err)
			}
			if moved > 0 {
				log.Printf("[迁移] migration 243 改挂 F 权限 %d 条: %s → 宿主 %s（%s）",
					moved, menu.ID, hostID, item.name)
			}
		}

		// 4) 隐藏 + 停用（已是目标态则不产生 UPDATE）
		if menu.Visible == models.VisibleHidden && menu.Status == models.MenuStatusStop {
			continue
		}
		if err := db.Model(&models.Menu{}).
			Where("id = ?", menu.ID).
			Updates(map[string]interface{}{
				"visible": models.VisibleHidden,
				"status":  models.MenuStatusStop,
			}).Error; err != nil {
			return fmt.Errorf("隐藏页面菜单失败(menu=%s): %w", menu.ID, err)
		}
		log.Printf("[迁移] migration 243 隐藏菜单: %s（%s）", menu.MenuName, item.name)
	}

	return nil
}

// findHostMenuID 按 component 变体查宿主 C 型菜单 ID；未找到返回 gorm.ErrRecordNotFound。
func findHostMenuID(db *gorm.DB, components []string) (string, error) {
	if len(components) == 0 {
		return "", gorm.ErrRecordNotFound
	}
	var host models.Menu
	err := db.Where("component IN ?", components).
		Where("menu_type = ?", models.MenuTypeMenu).
		Order("created_at").
		First(&host).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[迁移] migration 243 未找到宿主菜单(component=%v)，仅隐藏不迁移 F 权限", components)
		}
		return "", err
	}
	return host.ID, nil
}

// reparentButtonPermissions 把 parentID 下的 F 型按钮权限整体改挂到 newParentID，返回影响行数。
func reparentButtonPermissions(db *gorm.DB, parentID, newParentID string) (int64, error) {
	res := db.Model(&models.Menu{}).
		Where("parent_id = ?", parentID).
		Where("menu_type = ?", models.MenuTypeButton).
		Update("parent_id", newParentID)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
