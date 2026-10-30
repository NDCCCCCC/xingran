package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// 系统设置菜单的权威记录 (menu_catalog_seed.sql 201 行):
// id 固定 308d89be-…, path=settings-page 不动, 仅 component 随前端目录合并
// (Phase 70 D-11: src/pages/system/settings-page/ 并入 src/pages/system/settings/)。
const (
	// settingsMenuID 规范目录里「系统设置」页面菜单的固定 id。
	settingsMenuID = "308d89be-e516-4556-b949-bc22bf6ab759"
	// settingsMenuOldComponent 旧目录路径 (settings-page 壳, 已删除)。
	settingsMenuOldComponent = "system/settings-page/index"
	// settingsMenuNewComponent 新目录路径 (settings/index 入口, SettingsShell 实例)。
	// 值不带 pages/ 前缀、不带 .tsx 后缀 (componentLoader.tsx glob 契约)。
	settingsMenuNewComponent = "system/settings/index"
)

// Migrate209UpdateSettingsMenuComponent 把系统设置菜单的 component 从旧目录路径
// 修正到合并后的新路径 (Phase 70 D-11 数据面闭环)。
//
// 背景:
//   - 前端 src/pages/system/settings-page/ 整目录删除, 入口并入
//     src/pages/system/settings/index.tsx (SettingsShell 系统实例)。
//   - sys_menu.component 决定前端懒加载组件路径, 不迁移则菜单点开白屏
//     (componentLoader glob 拾取不到 system/settings-page/index)。
//
// 范围 (安全边界 T-70-01/T-70-0601): 只 UPDATE component 一个字段,
// path/perms/visible/status 一律不动 —— 路由 URL 保持 /system/settings-page,
// 权限语义 (system:config:list) 零变化。UPDATE 值为固定常量, 不接受任何用户输入。
//
// 幂等: id + 旧 component 值双条件 WHERE —— 旧值已不存在时 RowsAffected=0,
// 重复执行无副作用 (Go 单测 TestMigrate209… 双条件守护用例锁定)。
//
// 双方言: 纯 GORM Update, 无 PG 专有语法, PG 与 SQLite 分支各注册一次
// (database.go 207/208 同款挂法)。注意必须排在 Migrate207 之后: 全新库由 207
// 种子出旧 component 值, 本迁移随即修正为新值。
//
// 返回 changed = RowsAffected > 0, 供 core.go 在缓存服务就绪后按标志失效
// 菜单缓存 —— 迁移执行于 db.NewDatabase (早于 c.Cache 创建), 迁移函数内
// 拿不到 cache 实例, 不能在此直接 DeleteByPattern (PATTERNS 事实 1)。
func Migrate209UpdateSettingsMenuComponent(db *gorm.DB) (bool, error) {
	log.Println("Running migration 209: sys_menu component system/settings-page/index -> system/settings/index")

	res := db.Model(&models.Menu{}).
		Where("id = ?", settingsMenuID).
		Where("component = ?", settingsMenuOldComponent).
		Update("component", settingsMenuNewComponent)
	if res.Error != nil {
		return false, fmt.Errorf("migration 209: 系统设置菜单 component 更新失败: %w", res.Error)
	}

	changed := res.RowsAffected > 0
	if changed {
		applogger.Infof("[迁移] 209 系统设置菜单组件路径已更新 (rows=%d)", res.RowsAffected)
	}
	return changed, nil
}
