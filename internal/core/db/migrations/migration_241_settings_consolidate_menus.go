package migrations

import (
	"errors"
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate241SettingsConsolidateMenus —— 系统设置页合并（前端 SettingsShell 单一入口）。
//
// 背景：原「经理同步规则」(manager-rules)、「经理同步机构」(manager-orgs)、
// 「数据备份与恢复」(backup) 三个独立路由页，已在前端并入
// component=system/settings/index 的「系统设置」页，成为页内分类区块。
// 若菜单栏仍保留这三个入口，用户会进入「旧壳页」而非合并后的设置页 —— 因此本迁移
// 把它们的 C 型菜单隐藏，并把它们的 F 型按钮权限整行改挂到「系统设置」菜单下，
// 保证「菜单树 / 角色授权树」结构与前端实际页面结构一致。
//
// 为什么「隐藏」而不是「删除」：
//   - F 按钮的 perms（ops:manager:*、system:backup:*）是后端 router 的
//     RequirePermissions 与前端 permissionCodes 的单一事实源，删除 C 菜单行无收益，
//     却会连带删掉 sys_role_menu 里管理员手工做过的授权，且不可回滚；
//   - visible=0 只是「菜单栏不渲染」，component 仍可被路由生成器解析、旧链接/书签
//     不会 404，属于可逆的最小破坏（再次置 1 即可恢复入口）；
//   - 隐藏不触碰 perms / status，后端鉴权语义与既有授权行完全不变。
//
// 为什么改挂 F 按钮 parent_id：
//
//	角色授权树按 parent 递归渲染按钮权限节点。父菜单若继续指向已隐藏的旧 C 菜单，
//	授权树里会出现「挂在隐藏节点下的孤儿按钮」；改挂到「系统设置」后，权限仍然
//	存在、仍然可授权，只是出现在正确的页面节点下。**权限码原值不变**，
//	也不写 sys_role_menu —— 已授权角色继续有效，无需二次授权（与 169/195/228/240
//	的「不 INSERT sys_role_menu」约定同源）。
//
// 顺序说明（有意与「先隐藏后定位」颠倒）：先定位「系统设置」菜单作为前置门禁。
// 找不到它时整体跳过（返回 nil，仅告警），避免出现「旧入口已隐藏、新入口不存在」
// 导致三个功能页面在菜单栏彻底失联的半完成状态。
//
// 幂等策略：
//   - 隐藏：UPDATE ... SET visible=0，已隐藏的行重复写同值无副作用；
//   - 改挂：UPDATE ... SET parent_id=<系统设置 id>，已挂在该父下重复写同值无副作用；
//   - 定位失败（菜单不存在）一律按「跳过」处理，不报错（存量瘦库/自定义菜单名场景）。
func Migrate241SettingsConsolidateMenus(db *gorm.DB) error {
	log.Println("Running migration 241: 系统设置页合并（隐藏旧入口菜单 + 按钮权限改挂）")

	// ---------- 1. 前置门禁：定位「系统设置」C 菜单 ----------
	// 优先 menu_name='系统设置' AND menu_type='C'（种子里的规范行，固定 id
	// 308d89be-e516-4556-b949-bc22bf6ab759，migration_209 已把它的 component
	// 修正为 system/settings/index）；回退按 component 定位（菜单被改名的情况）。
	settingsMenu, err := locateSettingsMenu241(db)
	if err != nil {
		return err
	}
	if settingsMenu == nil {
		log.Printf("Warning: 未找到「系统设置」C 菜单 (menu_name=%q / component=%q), "+
			"migration_241 跳过隐藏与改挂（菜单栏保持原状, 管理员需手动创建系统设置菜单）",
			settingsConsolidateSettingsMenuName, settingsMenuNewComponent)
		return nil
	}
	settingsID := settingsMenu.ID
	applogger.Infof("[迁移] migration_241 定位到系统设置菜单: %s (id=%s)", settingsMenu.MenuName, settingsID)

	// ---------- 2. 隐藏三个已并入的 C 型路由菜单 ----------
	managerMenus := []struct{ name, path string }{
		{"经理同步规则", "manager-rules"},
		{"经理同步机构", "manager-orgs"},
	}
	for _, m := range managerMenus {
		if err := hideConsolidatedCMenu241(db, m.name, m.path); err != nil {
			return err
		}
	}
	if err := hideBackupCMenu241(db); err != nil {
		return err
	}

	// ---------- 3. manager 4 个 + backup 3 个 F 按钮改挂到「系统设置」 ----------
	// 清单见 settingsConsolidateButtonPerms：migration_221 只播种 edit/delete 两个 F
	// （list 权限挂在各自 C 菜单的 perms 上，随 C 菜单一起隐藏后仍可用于授权，
	// 无需搬移），migration_240 播种 list/export/restore 三个 F；合计 7 行。
	// 按 perms IN (...) 批量 UPDATE：权限码原值不变，仅改归属父节点。
	// 已挂在该父下 / 行不存在的场景都不产生副作用（后者 RowsAffected 更小，仅记日志）。
	res := db.Model(&models.Menu{}).
		Where("menu_type = ?", models.MenuTypeButton).
		Where("perms IN ?", settingsConsolidateButtonPerms).
		Update("parent_id", settingsID)
	if res.Error != nil {
		return fmt.Errorf("migration 241: 改挂按钮权限父菜单失败: %w", res.Error)
	}
	applogger.Infof("[迁移] migration_241 按钮权限改挂: %d/%d 行 parent_id → 系统设置(%s)",
		res.RowsAffected, len(settingsConsolidateButtonPerms), settingsID)
	if int(res.RowsAffected) < len(settingsConsolidateButtonPerms) {
		log.Printf("Migration 241: 部分按钮权限行不存在 (匹配 %d, 期望 %d), "+
			"缺失者由 migration_221/240 在下次启动补齐后自动归位",
			res.RowsAffected, len(settingsConsolidateButtonPerms))
	}

	log.Println("Migration 241 completed: 旧入口菜单已隐藏, 按钮权限已归入系统设置页")
	return nil
}

// settingsConsolidateSettingsMenuName 合并后的唯一设置入口菜单名。
const settingsConsolidateSettingsMenuName = "系统设置"

// settingsConsolidateBackupMenuID 「数据备份与恢复」C 菜单固定 UUID，
// 与 migration_240 的 menuID 及 menu_catalog_seed.sql 的 seed 行同源。
const settingsConsolidateBackupMenuID = "a1b2c3d4-0001-4000-8000-000000000240"

// settingsConsolidateBackupMenuName 「数据备份与恢复」C 菜单名（按 id 定位失败时的回退锚点）。
const settingsConsolidateBackupMenuName = "数据备份与恢复"

// settingsConsolidateButtonPerms 需要改挂到「系统设置」下的 F 型按钮权限码。
// manager 四个复用 migration_221 的包级常量（同一事实源，避免字面量漂移）；
// backup 三个与 migration_240 / menu_catalog_seed.sql 逐字一致。
var settingsConsolidateButtonPerms = []string{
	managerRuleEditPerms,
	managerRuleDeletePerms,
	managerOrgEditPerms,
	managerOrgDeletePerms,
	"system:backup:list",
	"system:backup:export",
	"system:backup:restore",
}

// locateSettingsMenu241 定位「系统设置」C 菜单；不存在返回 (nil, nil) 由调用方降级跳过。
func locateSettingsMenu241(db *gorm.DB) (*models.Menu, error) {
	var menu models.Menu
	err := db.Where("menu_name = ? AND menu_type = ?",
		settingsConsolidateSettingsMenuName, models.MenuTypeMenu).First(&menu).Error
	if err == nil {
		return &menu, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("migration 241: 按名称查询系统设置菜单失败: %w", err)
	}

	// 回退：菜单被改名的存量库，按 component 锚定（migration_209 修正后的规范值）。
	err = db.Where("component = ?", settingsMenuNewComponent).First(&menu).Error
	if err == nil {
		return &menu, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("migration 241: 按 component 查询系统设置菜单失败: %w", err)
	}
	return nil, nil
}

// hideConsolidatedCMenu241 隐藏一个已并入系统设置页的 C 型菜单。
// 定位优先 menu_name + menu_type='C'（种子里就是按名字建的行），
// 回退按 path + menu_type='C'（菜单被改名的存量库）。已隐藏/不存在都不报错。
func hideConsolidatedCMenu241(db *gorm.DB, menuName, path string) error {
	res := db.Model(&models.Menu{}).
		Where("menu_name = ? AND menu_type = ?", menuName, models.MenuTypeMenu).
		Update("visible", models.VisibleHidden)
	if res.Error != nil {
		return fmt.Errorf("migration 241: 隐藏菜单 %s 失败: %w", menuName, res.Error)
	}
	if res.RowsAffected > 0 {
		applogger.Infof("[迁移] sys_menu 隐藏: %s (visible=0, 已并入系统设置页)", menuName)
		return nil
	}

	res = db.Model(&models.Menu{}).
		Where("path = ? AND menu_type = ?", path, models.MenuTypeMenu).
		Update("visible", models.VisibleHidden)
	if res.Error != nil {
		return fmt.Errorf("migration 241: 按 path 隐藏菜单 %s 失败: %w", path, res.Error)
	}
	if res.RowsAffected > 0 {
		applogger.Infof("[迁移] sys_menu 隐藏: path=%s (visible=0, 已并入系统设置页)", path)
		return nil
	}
	log.Printf("Migration 241: 菜单 %s (path=%s) 不存在, 跳过隐藏", menuName, path)
	return nil
}

// hideBackupCMenu241 隐藏「数据备份与恢复」C 型菜单。
// 定位优先固定 UUID（migration_240 与 seed 同源），回退 menu_name + menu_type='C'。
func hideBackupCMenu241(db *gorm.DB) error {
	res := db.Model(&models.Menu{}).
		Where("id = ?", settingsConsolidateBackupMenuID).
		Update("visible", models.VisibleHidden)
	if res.Error != nil {
		return fmt.Errorf("migration 241: 隐藏备份菜单失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		applogger.Infof("[迁移] sys_menu 隐藏: 数据备份与恢复 (id=%s, visible=0)", settingsConsolidateBackupMenuID)
		return nil
	}

	res = db.Model(&models.Menu{}).
		Where("menu_name = ? AND menu_type = ?", settingsConsolidateBackupMenuName, models.MenuTypeMenu).
		Update("visible", models.VisibleHidden)
	if res.Error != nil {
		return fmt.Errorf("migration 241: 按名称隐藏备份菜单失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		applogger.Infof("[迁移] sys_menu 隐藏: 数据备份与恢复 (按名称匹配, visible=0)")
		return nil
	}
	log.Printf("Migration 241: 菜单 %s 不存在, 跳过隐藏", settingsConsolidateBackupMenuName)
	return nil
}
