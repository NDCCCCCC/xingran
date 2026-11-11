package migrations

// migration_242 · 同步漂移告警并入「AD 域控配置」页 —— 菜单侧迁移。
//
// 背景：漂移告警是 AD 域控同步的附属监控（internal/services/addomain/
// ad_sync_drift_alert_service.go 监控 sys_ad_group_member 计数漂移，
// internal/scheduler/ad_sync_tasks.go 每小时检测）。原 migration_238 为它播种了
// 独立 C 型页面菜单「同步漂移告警」；前端已并入 ad-domain/configs/index.tsx 的
// 第三个 Tab（配置列表 / 对账 / 漂移告警），不再需要独立菜单。
//
// 本迁移只做菜单/权限，三件事（双方言 · 幂等）：
//  ① 隐藏 migration_238 播种的 C 型菜单「同步漂移告警」
//     （visible=0 隐藏 + status=1 停用；行保留，旧直达链接的 perms 语义不变）；
//  ② 4 个 F 权限码 system:adDriftAlert:{list,config,edit,trigger} 原值不变，
//     父菜单改挂「AD域控配置」C 型菜单（component = ad-domain/configs/index）；
//  ③ 按「持有 AD域控配置 C 菜单」的角色精准授权这 4 个 F 行（NOT EXISTS 幂等），
//     与 migration_163（服务账号池按钮权限挂宿主菜单、不建独立菜单）同一先例。
//
// 为什么**不**在这里重复声明 F_TYPE_SEED 块：check-permission-codes.mjs 会把
// 每个 `// F_TYPE_SEED_BEGIN ... END` 块解析成 F 型码集合，重复声明同一批码会让
// 校验侧出现重复项。这里直接复用同包 migration_238 的 adDriftAlertFTypeSeeds。
//
// 幂等：hide 是幂等 UPDATE；ensureActionPermissionMenu 按 perms+menu_type 复用既有行
// （只补 parent_id）；授权走 NOT EXISTS 子查询。连跑两次零重复。
//
// 与 migration_238 每次启动重跑的关系：238 的 F 播种走 resolveActionPermissionParents
// （过滤 status=0），旧 C 菜单被本迁移停用后 238 找不到父 → log warning 跳过，
// 不会把 F 行父菜单改回旧 C；ensureADDriftAlertPageMenu 只复用既有 C 行、不改其状态。

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// 宿主菜单定位键：component 最稳定（menu_name 历史上出现过「AD域控配置」与「域控配置」两种）。
const (
	adConfigPageMenuComponent = "ad-domain/configs/index"
	adConfigPageMenuPath      = "configs"
)

// 旧漂移告警 C 菜单定位键（migration_238 播种值）。
const (
	adDriftAlertLegacyMenuName  = "同步漂移告警"
	adDriftAlertLegacyComponent = "system/ad-drift-alert/index"
)

// Migrate242ADDriftAlertMergeMenu 把漂移告警菜单/权限收敛到 AD 域控配置页（幂等）。
func Migrate242ADDriftAlertMergeMenu(db *gorm.DB) error {
	log.Println("Running migration 242: merge AD drift alert into AD config page (menu migration)")

	if !db.Migrator().HasTable(&models.Menu{}) {
		log.Println("Migration 242: WARNING sys_menu 表不存在, 跳过 (非阻断)")
		return nil
	}

	// 先隐藏旧 C 菜单：即使后续挂载失败，也不会同时存在两个可见入口。
	if err := hideADDriftAlertLegacyMenu(db); err != nil {
		return err
	}

	cfgMenu, err := findADConfigPageMenu(db)
	if err != nil {
		return err
	}
	if cfgMenu == nil {
		// 宿主菜单缺失时优雅降级（admin 走超管旁路；下次启动重试）。
		log.Println("Migration 242: WARNING 未找到「AD域控配置」C 型菜单, 跳过 F 权限挂载与授权 (非阻断)")
		return nil
	}

	if len(adDriftAlertFTypeSeeds) == 0 {
		return errors.New("migration 242 致命漂移: adDriftAlertFTypeSeeds 为空")
	}

	// 升级库去重：238 历史随机 UUID F 行与 207 seed 固定 id F 行（perms 相同）
	// 可能并存（seed 逐行仅 ON CONFLICT(id) DO NOTHING，不按 perms 拦截）。
	// ensureActionPermissionMenu 的 First 无 ORDER BY，命中哪行不确定 —— 必须先收敛。
	if err := dedupeADDriftAlertFPermissions(db, cfgMenu.ID); err != nil {
		return err
	}

	for _, seed := range adDriftAlertFTypeSeeds {
		btnID, err := ensureActionPermissionMenu(db, seed, cfgMenu.ID)
		if err != nil {
			log.Printf("Migration 242: WARNING 创建/复用按钮菜单失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}
		// ensureActionPermissionMenu 复用既有行时**不改父** —— 这里显式改挂宿主菜单。
		if err := db.Model(&models.Menu{}).
			Where("id = ? AND menu_type = ?", btnID, models.MenuTypeButton).
			Update("parent_id", cfgMenu.ID).Error; err != nil {
			log.Printf("Migration 242: WARNING 改挂按钮父菜单失败(perms=%s): %v, 跳过授权", seed.perms, err)
			continue
		}
		if err := grantActionPermissionToParentHolders(db, btnID, []string{cfgMenu.ID}); err != nil {
			log.Printf("Migration 242: WARNING 授权按钮菜单失败(perms=%s): %v, 跳过授权", seed.perms, err)
			continue
		}
	}

	log.Printf("Migration 242 completed: 4 个 system:adDriftAlert:* 已挂到「%s」菜单下", cfgMenu.MenuName)
	return nil
}

// hideADDriftAlertLegacyMenu 隐藏（visible=0 + status=1）migration_238 播种的
// C 型「同步漂移告警」菜单；行本身保留（历史审计 + 旧直达链接可解析）。
func hideADDriftAlertLegacyMenu(db *gorm.DB) error {
	var menus []models.Menu
	if err := db.
		Where("menu_type = ? AND (menu_name = ? OR component = ?)",
			models.MenuTypeMenu, adDriftAlertLegacyMenuName, adDriftAlertLegacyComponent).
		Find(&menus).Error; err != nil {
		return fmt.Errorf("查询旧「%s」菜单失败: %w", adDriftAlertLegacyMenuName, err)
	}
	for _, menu := range menus {
		if menu.Visible == models.VisibleHidden && menu.Status == models.MenuStatusStop {
			continue // 已是隐藏停用状态 → 幂等短路，避免无意义写
		}
		if err := db.Model(&models.Menu{}).
			Where("id = ?", menu.ID).
			Updates(map[string]interface{}{
				"visible": models.VisibleHidden,
				"status":  models.MenuStatusStop,
			}).Error; err != nil {
			return fmt.Errorf("隐藏旧菜单(%s)失败: %w", menu.ID, err)
		}
		log.Printf("Migration 242: 隐藏旧菜单 %s (visible=0, status=1)", menu.ID)
	}
	return nil
}

// findADConfigPageMenu 定位「AD域控配置」C 型宿主菜单。
// 依次按 component（最稳定）→ path+名称 → 纯名称三层兜底，每层先只看 status=0。
func findADConfigPageMenu(db *gorm.DB) (*models.Menu, error) {
	builders := []func() *gorm.DB{
		func() *gorm.DB {
			return db.Where("menu_type = ? AND component = ?",
				models.MenuTypeMenu, adConfigPageMenuComponent)
		},
		func() *gorm.DB {
			return db.Where("menu_type = ? AND path = ? AND menu_name IN ?",
				models.MenuTypeMenu, adConfigPageMenuPath,
				[]string{"AD域控配置", "域控配置"})
		},
		func() *gorm.DB {
			return db.Where("menu_type = ? AND menu_name IN ?",
				models.MenuTypeMenu, []string{"AD域控配置", "域控配置"})
		},
	}

	for _, build := range builders {
		// 先只认启用中的菜单，再放宽到任意状态（停用的宿主也比找不到好）
		for _, onlyNormal := range []bool{true, false} {
			q := build()
			if onlyNormal {
				q = q.Where("status = ?", models.MenuStatusNormal)
			}
			var menu models.Menu
			err := q.Order("id ASC").First(&menu).Error
			if err == nil {
				return &menu, nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("查询「AD域控配置」菜单失败: %w", err)
			}
		}
	}
	return nil, nil
}

// dedupeADDriftAlertFPermissions 对 4 个 system:adDriftAlert:{list,config,edit,trigger}
// 按 perms + menu_type='F' 去重（幂等）。
//
// 背景（升级库升级路径）：migration_238 先播种了 4 个随机 UUID 的 F 行；随后
// migration_207 的 seed 慢速路径又逐行插入 4 个固定 id 的 F 行（ON CONFLICT 仅按
// id 判断，perms 相同也照样插入）→ 同一 perms 出现两行。ensureActionPermissionMenu
// 按 perms 查找时 First 无 ORDER BY，命中哪行不确定，导致另一行残留、权限校验侧
// 可能命中未挂宿主的旧行。本函数在每个 perms 仅保留一行：
//   - 优先保留已挂在宿主（AD域控配置 C 菜单）下的行（即 seed 固定 id 行或 242 上次已收敛的行）；
//   - 否则保留最早创建（created_at ASC, id ASC 排序确定性兜底）的行。
//
// 删除遵循 migration_207 reconcileLegacyOpsTree 先例：先清 sys_role_menu 授权关联，
// 再 GORM 软删（deleted_at = now，可恢复；不硬删，避免未来外键/审计需求回归）。
// 软删行被 GORM 默认作用域过滤，重复执行查不到 → 幂等无副作用。
func dedupeADDriftAlertFPermissions(db *gorm.DB, hostMenuID string) error {
	for _, seed := range adDriftAlertFTypeSeeds {
		var rows []models.Menu
		if err := db.
			Where("perms = ? AND menu_type = ?", seed.perms, models.MenuTypeButton).
			Order("created_at ASC, id ASC").
			Find(&rows).Error; err != nil {
			return fmt.Errorf("查询既有 F 按钮行失败(perms=%s): %w", seed.perms, err)
		}
		if len(rows) <= 1 {
			continue // 无重复（含已收敛的库）→ 短路
		}

		// 保留行：优先已挂宿主的行；否则最早创建的行（rows 已按上述顺序排列）
		keep := rows[0]
		for _, r := range rows {
			if r.ParentID != nil && *r.ParentID == hostMenuID {
				keep = r
				break
			}
		}

		removeIDs := make([]string, 0, len(rows)-1)
		for _, r := range rows {
			if r.ID != keep.ID {
				removeIDs = append(removeIDs, r.ID)
			}
		}
		if err := db.Exec("DELETE FROM sys_role_menu WHERE menu_id IN ?", removeIDs).Error; err != nil {
			return fmt.Errorf("清理重复 F 行授权失败(perms=%s): %w", seed.perms, err)
		}
		if err := db.Where("id IN ?", removeIDs).Delete(&models.Menu{}).Error; err != nil {
			return fmt.Errorf("软删重复 F 行失败(perms=%s): %w", seed.perms, err)
		}
		log.Printf("Migration 242: 去重 %s: 保留 %s, 软删 %v", seed.perms, keep.ID, removeIDs)
	}
	return nil
}
