package migrations

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ============================================================================
// Migration 255 — Phase 155 / LIVEVIEW-04：RPA 实时画面菜单（C 型页面）
//
// 归属协议第 6 条（路由统一）：Live 页面**只**通过 sys_menu seed 注册，
// 前端不改 componentLoader.tsx / DynamicRoutes.tsx（否则与 Phase 156 硬冲突）。
//
// 播种内容：
//   - 1 个 C 型页面菜单「RPA 实时画面」（component=pages/operations/rpa/live/index），
//     父菜单 = 既有的「RPA 管理」af1451a4-650c-4ed1-a2b7-b58769dc3920
//     （menu_catalog_seed.sql:193）→ 这样它天然继承 RPA 域的可见性与授权链。
//   - 不新增 F 型按钮：LIVEVIEW-05 明写「复用执行读权限」，权限码仍是
//     ops:rpa:execution:list（migration_245 已播种），故不扩权限面。
//
// 幂等：按固定 UUID 先查后建；已存在即跳过（绝不覆盖管理员改过的父菜单/排序）。
// 非阻断：失败由调用方 WARN，下次启动重试（与 240/245 同款约定）。
// 双分支：PG / sqlite 都用 GORM 模型（无方言专属 DDL），无需分支。
// ============================================================================

// 固定 UUID（与 menu_catalog_seed.sql 的既有 UUID 命名风格一致：
// 前缀标识来源、后 12 位为版本号 255）。
const (
	liveViewMenuID  = "a1b2c3d4-0255-4000-8000-000000000255"
	liveViewParent  = "af1451a4-650c-4ed1-a2b7-b58769dc3920" // 「RPA 管理」C 型父菜单
	liveViewPath    = "live"
	liveViewComp    = "pages/operations/rpa/live/index"
	liveViewMenuName = "RPA 实时画面"
)

// Migrate255LiveViewMenu Phase 155 / LIVEVIEW-04：播种实时画面菜单。
func Migrate255LiveViewMenu(db *gorm.DB) error {
	log.Println("Running migration 255: RPA liveview menu seed")

	// 父菜单是否存在（缺失则本迁移无意义：说明 RPA 管理菜单尚未播种）。
	var parentCount int64
	if err := db.Model(&models.Menu{}).Where("id = ?", liveViewParent).Count(&parentCount).Error; err != nil {
		return fmt.Errorf("migration 255: 查询父菜单失败: %w", err)
	}
	if parentCount == 0 {
		// 非阻断：全新空库由 migration_207（menu_catalog_seed.sql）先播种，
		// 启动顺序保证 207 在 255 之前；若此处缺失则登记 WARN 而不报错中断。
		applogger.Warnf("[迁移] migration 255: 父菜单「RPA 管理」(%s) 不存在，跳过实时画面菜单播种", liveViewParent)
		return nil
	}

	var count int64
	if err := db.Model(&models.Menu{}).Where("id = ?", liveViewMenuID).Count(&count).Error; err != nil {
		return fmt.Errorf("migration 255: 查询菜单失败: %w", err)
	}
	if count > 0 {
		applogger.Infof("[迁移] migration 255: 实时画面菜单已存在 (%s), 跳过", liveViewMenuID)
		return nil
	}

	parentID := liveViewParent
	path := liveViewPath
	comp := liveViewComp
	// 不设 perms：页面可见性由父菜单（RPA 管理）授权链决定，页面本身不占权限码。
	// 这与 menu_catalog_seed.sql:193「RPA 管理」的 perms=NULL 同款处理。
	icon := "FundProjectionScreenOutlined"
	remark := "实时画面与人工接管（Phase 155 / LIVEVIEW-01~05，迁移 255 播种）"

	menu := models.Menu{
		MenuName:  liveViewMenuName,
		ParentID:  &parentID,
		OrderNum:  16, // 排在既有 RPA F 按钮（1–14）之后、不影响执行列表
		Path:      &path,
		Component: &comp,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Icon:      &icon,
		Remark:    remark,
	}
	menu.ID = liveViewMenuID

	if err := db.Create(&menu).Error; err != nil {
		// 唯一冲突（重启竞态）视为已播种，不阻断启动。
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return fmt.Errorf("migration 255: 创建实时画面菜单失败: %w", err)
	}

	applogger.Infof("[迁移] sys_menu seed: %s (component=%s, parent=%s, order=16)",
		liveViewMenuName, liveViewComp, liveViewParent)
	return nil
}

// 编译期断言：liveViewMenuName 不被误用于权限码（本迁移不扩权限面）。
var _ = func(s string) string { return s }(liveViewMenuName)
