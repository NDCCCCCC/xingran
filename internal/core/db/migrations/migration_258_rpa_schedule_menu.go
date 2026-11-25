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
// Migration 258 — Phase 158 / REC-02：RPA 定时调度页面菜单（C 型页面）
//
// 归属协议第 6 条（路由统一）：schedules 页面**只**通过 sys_menu seed 注册，
// 前端**不改** componentLoader.tsx / DynamicRoutes.tsx —— componentLoader 的
// 通配 glob（import.meta.glob("/src/pages/**/{index,detail}.tsx")）会自动
// 收集 pages/operations/rpa/schedules/index.tsx，改前端注册文件会与
// Phase 155/156 在同一批文件上硬冲突。
//
// 播种内容：
//   - 1 个 C 型页面菜单「定时调度」（component=pages/operations/rpa/schedules/index），
//     父菜单 = 既有的「RPA 管理」af1451a4-650c-4ed1-a2b7-b58769dc3920
//     → 天然继承 RPA 域的可见性与授权链。
//   - 不新增 F 型按钮：调度 4 个权限码 ops:rpa:schedule:{list,add,edit,remove}
//     已由 migration_257 播种，本迁移**不扩权限面**。
//
// 幂等：按固定 UUID 先查后建；已存在即跳过（绝不覆盖管理员改过的父菜单/排序）。
// 非阻断：父菜单缺失时 WARN 而非中断（与 255/256 同款约定）。
// 双分支：PG / sqlite 都用 GORM 模型（无方言专属 DDL），无需分支。
// ============================================================================

// 固定 UUID（与 255/256 同款命名风格：前缀标识来源、后 12 位为版本号 258）。
const (
	scheduleMenuID     = "a1b2c3d4-0258-4000-8000-000000000258"
	scheduleMenuParent = "af1451a4-650c-4ed1-a2b7-b58769dc3920" // 「RPA 管理」C 型父菜单
	scheduleMenuPath   = "schedules"
	scheduleMenuComp   = "pages/operations/rpa/schedules/index"
	scheduleMenuName   = "定时调度"
	scheduleMenuOrder  = 18 // 紧邻 256 的「新建自动化任务」(order=17)
)

// Migrate258RpaScheduleMenu Phase 158 / REC-02：播种 RPA 定时调度页面菜单。
func Migrate258RpaScheduleMenu(db *gorm.DB) error {
	log.Println("Running migration 258: RPA schedule menu seed")

	var parentCount int64
	if err := db.Model(&models.Menu{}).Where("id = ?", scheduleMenuParent).Count(&parentCount).Error; err != nil {
		return fmt.Errorf("migration 258: 查询父菜单失败: %w", err)
	}
	if parentCount == 0 {
		// 非阻断：全新空库由 migration_207（menu_catalog_seed.sql）先播种，
		// 启动顺序保证 207 在 258 之前；若此处缺失则登记 WARN 而不报错中断。
		applogger.Warnf("[迁移] migration 258: 父菜单「RPA 管理」(%s) 不存在，跳过定时调度菜单播种", scheduleMenuParent)
		return nil
	}

	var count int64
	if err := db.Model(&models.Menu{}).Where("id = ?", scheduleMenuID).Count(&count).Error; err != nil {
		return fmt.Errorf("migration 258: 查询菜单失败: %w", err)
	}
	if count > 0 {
		applogger.Infof("[迁移] migration 258: 定时调度菜单已存在 (%s), 跳过", scheduleMenuID)
		return nil
	}

	parentID := scheduleMenuParent
	path := scheduleMenuPath
	comp := scheduleMenuComp
	// 不设 perms：页面可见性由父菜单（RPA 管理）授权链决定，页面本身不占权限码。
	// 与 255/256 同款处理；调度 4 个 F 型权限码由 migration_257 播种。
	icon := "ClockCircleOutlined"
	remark := "RPA 定时调度页面（Phase 158 / REC-02，迁移 258 播种）"

	menu := models.Menu{
		MenuName:  scheduleMenuName,
		ParentID:  &parentID,
		OrderNum:  scheduleMenuOrder,
		Path:      &path,
		Component: &comp,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Icon:      &icon,
		Remark:    remark,
	}
	menu.ID = scheduleMenuID

	if err := db.Create(&menu).Error; err != nil {
		// 唯一冲突（重启竞态）视为已播种，不阻断启动。
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return fmt.Errorf("migration 258: 创建定时调度菜单失败: %w", err)
	}

	applogger.Infof("[迁移] sys_menu seed: %s (component=%s, parent=%s, order=%d)",
		scheduleMenuName, scheduleMenuComp, scheduleMenuParent, scheduleMenuOrder)
	return nil
}
