package migrations

// Migration 249 — duty/pools 菜单状态归位：消除「独立页 + Tab」双入口。
//
// 背景：值班池页面已并入「值班管理」页的「值班池管理」Tab（management/index.tsx:264），
// 但 migration_228 的 ensureMenuVisible 每次启动把 duty/pools/index 强制改成
// visible=1/status=0（runAndRecord 不检查版本是否已应用，故每次启动都执行）。
// internal/services/system/menu_service.go:497 的 /my-menus/all 只按 status=0 过滤，
// 这些 status=0 的活行会生成 /duty/pools 独立路由；而组件 pages/duty/pools/index.tsx
// 仍存在（Tab 依赖它，见 management/index.tsx:29 的 import）→ 双入口 + 侧边栏重复项。
// seed 基线 c4a5ff39 在 menu_catalog_seed.sql:27 本为 visible=0/status=1（被 228 改掉）。
//
// 用户已确认 duty/pools 应是被 Tab 复用的「隐藏停用」既有状态（页面目录必须保留）。
// 本迁移只做状态归位，三不碰：不软删/不物理删、不动 perms、不动 sys_role_menu。
//
// 处置：把所有**未软删**的 component IN ('duty/pools/index','/duty/pools/index')
// AND menu_type='C' 行统一为 visible=0 / status=1（与 seed 基线一致）。
// GORM 默认作用域自动追加 deleted_at IS NULL —— 软删行根本不会被 Find 捞出，
// 因此天然满足「只 UPDATE 活行」；逐行已是目标态则短路不产生无意义写。
//
// 幂等：count-then-act / 逐行短路；连跑两次零副作用。
// 双方言：纯 GORM 条件查询 + 通用常量，无 PG 专属语法。
// 与 228 的分工：228 已不再激活 duty/pools（只保留 duty/schedules 幂等保底），
// 存量行的归位由本迁移负责；两者每次启动重跑，顺序执行后状态稳定为 0/1。

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// 历史数据存在无前导斜杠 / 带前导斜杠两种 component 写法，全部覆盖。
var migrate249DutyPoolsComponents = []string{
	"duty/pools/index",
	"/duty/pools/index",
}

// Migrate249DutyPoolsMenuState 把所有未软删的 duty/pools C 行归位为隐藏停用（幂等 / 双方言）。
func Migrate249DutyPoolsMenuState(db *gorm.DB) error {
	log.Println("Running migration 249: normalize duty/pools menu state to hidden+stopped")

	if !db.Migrator().HasTable(&models.Menu{}) {
		log.Println("Migration 249: WARNING sys_menu 表不存在, 跳过 (非阻断)")
		return nil
	}

	// GORM 默认软删作用域：Find 只返回 deleted_at IS NULL 的活行。
	var menus []models.Menu
	if err := db.Where("component IN ?", migrate249DutyPoolsComponents).
		Where("menu_type = ?", models.MenuTypeMenu).
		Find(&menus).Error; err != nil {
		return fmt.Errorf("migration 249 查询 duty/pools 菜单失败: %w", err)
	}
	if len(menus) == 0 {
		log.Println("Migration 249: 无未软删的 duty/pools C 行, 跳过")
		return nil
	}

	changed := 0
	for i := range menus {
		menu := &menus[i]
		if menu.Visible == models.VisibleHidden && menu.Status == models.MenuStatusStop {
			continue // 已是目标态 → 幂等短路，避免无意义写
		}
		// 只 UPDATE 活行（GORM 默认软删过滤）、只改 visible/status 两列。
		if err := db.Model(&models.Menu{}).
			Where("id = ?", menu.ID).
			Updates(map[string]interface{}{
				"visible": models.VisibleHidden,
				"status":  models.MenuStatusStop,
			}).Error; err != nil {
			return fmt.Errorf("migration 249 归位 duty/pools 行(%s)失败: %w", menu.ID, err)
		}
		changed++
		log.Printf("Migration 249: 归位 duty/pools 行 %s (visible=0, status=1)", menu.ID)
	}

	log.Printf("Migration 249 completed: 共 %d 条活行归位, 共 %d 条匹配行保持幂等", changed, len(menus))
	return nil
}
