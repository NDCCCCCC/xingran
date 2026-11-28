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
// Migration 256 — Phase 156 / NOCODE-04：新建任务向导页面菜单（C 型页面）
//
// 归属协议第 6 条（路由统一）：wizard 页面**只**通过 sys_menu seed 注册，
// 前端**不改** componentLoader.tsx / DynamicRoutes.tsx（否则与 Phase 155 在
// 同一批文件上硬冲突）。
//
// 播种内容：
//   - 1 个 C 型页面菜单「新建自动化任务」（component=pages/operations/rpa/wizard/index），
//     父菜单 = 既有的「RPA 管理」af1451a4-650c-4ed1-a2b7-b58769dc3920
//     → 天然继承 RPA 域的可见性与授权链。
//   - 不新增 F 型按钮：向导保存/执行复用既有 ops:rpa:task:add / ops:rpa:task:execute
//     （migration_245 已播种），故不扩权限面。
//
// 幂等：按固定 UUID 先查后建；已存在即跳过（绝不覆盖管理员改过的父菜单/排序）。
// 非阻断：父菜单缺失时 WARN 而非中断（与 255 同款约定）。
// 双分支：PG / sqlite 都用 GORM 模型（无方言专属 DDL），无需分支。
// ============================================================================

// 固定 UUID（与 255 同款命名风格：前缀标识来源、后 12 位为版本号 256）。
const (
	wizardMenuID     = "a1b2c3d4-0256-4000-8000-000000000256"
	wizardParent     = "af1451a4-650c-4ed1-a2b7-b58769dc3920" // 「RPA 管理」C 型父菜单
	wizardPath       = "wizard"
	wizardComp       = "pages/operations/rpa/wizard/index"
	wizardMenuName   = "新建自动化任务"
	wizardMenuOrder  = 17 // 紧邻 155 的「RPA 实时画面」(order=16)，在任务列表之后
)

// Migrate256NocodeWizardMenu Phase 156 / NOCODE-04：播种新建任务向导菜单。
func Migrate256NocodeWizardMenu(db *gorm.DB) error {
	log.Println("Running migration 256: RPA nocode wizard menu seed")

	var parentCount int64
	if err := db.Model(&models.Menu{}).Where("id = ?", wizardParent).Count(&parentCount).Error; err != nil {
		return fmt.Errorf("migration 256: 查询父菜单失败: %w", err)
	}
	if parentCount == 0 {
		// 非阻断：全新空库由 migration_207（menu_catalog_seed.sql）先播种，
		// 启动顺序保证 207 在 256 之前；若此处缺失则登记 WARN 而不报错中断。
		applogger.Warnf("[迁移] migration 256: 父菜单「RPA 管理」(%s) 不存在，跳过向导菜单播种", wizardParent)
		return nil
	}

	var count int64
	if err := db.Model(&models.Menu{}).Where("id = ?", wizardMenuID).Count(&count).Error; err != nil {
		return fmt.Errorf("migration 256: 查询菜单失败: %w", err)
	}
	if count > 0 {
		applogger.Infof("[迁移] migration 256: 向导菜单已存在 (%s), 跳过", wizardMenuID)
		return nil
	}

	parentID := wizardParent
	path := wizardPath
	comp := wizardComp
	// 不设 perms：页面可见性由父菜单（RPA 管理）授权链决定，页面本身不占权限码。
	// 与 255 的「RPA 实时画面」同款处理。
	icon := "PlusCircleOutlined"
	remark := "新建自动化任务三步向导（Phase 156 / NOCODE-01~04，迁移 256 播种）"

	menu := models.Menu{
		MenuName:  wizardMenuName,
		ParentID:  &parentID,
		OrderNum:  wizardMenuOrder,
		Path:      &path,
		Component: &comp,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Icon:      &icon,
		Remark:    remark,
	}
	menu.ID = wizardMenuID

	if err := db.Create(&menu).Error; err != nil {
		// 唯一冲突（重启竞态）视为已播种，不阻断启动。
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return fmt.Errorf("migration 256: 创建向导菜单失败: %w", err)
	}

	applogger.Infof("[迁移] sys_menu seed: %s (component=%s, parent=%s, order=%d)",
		wizardMenuName, wizardComp, wizardParent, wizardMenuOrder)
	return nil
}
