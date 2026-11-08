package migrations

// Phase 144 / ALRT-01/02/03 · 同步漂移告警菜单 + 权限 + sys_config seed。
//
// 三件事：
//  ① C 型页面菜单「同步漂移告警」：component=system/ad-drift-alert/index，
//     挂 canonicalSystemRootID「系统管理」根，perms=system:adDriftAlert；
//  ② 4 个 F 型操作码（list/config/edit/trigger）：复用 migration_226 的
//     actionPermissionSeed + resolveActionPermissionParents +
//     ensureActionPermissionMenu + grantActionPermissionToParentHolders，
//     精准授权给持有父 C 菜单的角色（非 admin 不恒 403）；
//  ③ 9 行 sys_config 默认 seed（5 既有 + 4 新增，补 134-SUMMARY.md:32
//     记的「sys_config 旋钮无 seed」缺口）。
//
// 双方言：纯 GORM / db.Exec，无 ::text / ON CONFLICT / gen_random_uuid() 等
// PG 专属语法；幂等：count-then-insert + NOT EXISTS 授权，连跑两次零重复。
// 根菜单缺失时优雅降级（log warning，非阻断，admin 走超管旁路）。

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// F 型声明块（check-permission-codes.mjs 解析为 menuFType，参与 d 类校验）。
// 必须与 adDriftAlertFTypeSeeds 严格 1:1（逐条相等且数量一致）。
// ============================================================================
// F_TYPE_SEED_BEGIN
var adDriftAlertFTypeSeedCodes = []string{
	"system:adDriftAlert:list",
	"system:adDriftAlert:config",
	"system:adDriftAlert:edit",
	"system:adDriftAlert:trigger",
}

// F_TYPE_SEED_END

var adDriftAlertFTypeSeeds = []actionPermissionSeed{
	{name: "漂移告警查询", perms: "system:adDriftAlert:list", parentPerms: "system:adDriftAlert", orderNum: 1},
	{name: "漂移告警配置查询", perms: "system:adDriftAlert:config", parentPerms: "system:adDriftAlert", orderNum: 2},
	{name: "漂移告警配置编辑", perms: "system:adDriftAlert:edit", parentPerms: "system:adDriftAlert", orderNum: 3},
	{name: "漂移检测手动触发", perms: "system:adDriftAlert:trigger", parentPerms: "system:adDriftAlert", orderNum: 4},
}

// adDriftAlertConfigSeeds 9 行 sys_config 默认 seed（5 既有 + 4 新增）。
// 值均为字符串，与 addomain.loadConfig / GetConfig 默认值一致。
var adDriftAlertConfigSeeds = []struct {
	key   string
	value string
	name  string
}{
	{"ad_sync.drift_alert.enabled", "true", "AD 同步漂移告警总开关"},
	{"ad_sync.drift_alert.tolerance", "0", "AD 同步漂移告警容差"},
	{"ad_sync.drift_alert.throttle_minutes", "60", "AD 同步漂移告警节流分钟数"},
	{"ad_sync.drift_alert.notice_enabled", "true", "AD 同步漂移告警站内通知开关"},
	{"ad_sync.drift_alert.baseline", "", "AD 同步漂移基线(member_count,distinct_groups)"},
	{"ad_sync.drift_alert.ws_enabled", "true", "AD 同步漂移告警 WS 广播开关"},
	{"ad_sync.drift_alert.webhook_enabled", "false", "AD 同步漂移告警 Webhook 开关"},
	{"ad_sync.drift_alert.webhook_config_id", "", "AD 同步漂移告警 Webhook 配置ID(空=默认)"},
	{"ad_sync.drift_alert.baseline_key", "ad_sync.drift_alert.baseline", "AD 同步漂移基线 config_key 别名(前向兼容)"},
}

// Migrate238ADDriftAlert 播种漂移告警菜单 / 权限 / 配置（双方言 / 幂等）。
func Migrate238ADDriftAlert(db *gorm.DB) error {
	log.Println("Running migration 238: ad drift alert menu + permission + sys_config seed (Phase 144)")

	// 0) F 型声明块与种子表一致性硬校验（防静默漂移，照 migration_237）
	if len(adDriftAlertFTypeSeedCodes) != len(adDriftAlertFTypeSeeds) {
		return fmt.Errorf("migration 238 致命漂移: F_TYPE_SEED 声明数(%d) != 种子表数(%d)",
			len(adDriftAlertFTypeSeedCodes), len(adDriftAlertFTypeSeeds))
	}
	for i, code := range adDriftAlertFTypeSeedCodes {
		if code != adDriftAlertFTypeSeeds[i].perms {
			return fmt.Errorf("migration 238 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, adDriftAlertFTypeSeeds[i].perms)
		}
	}

	// 1) 定位「系统管理」根菜单（先规范 ID，再按名字兜底）
	var parent models.Menu
	err := db.Where("id = ?", canonicalSystemRootID).First(&parent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Where("menu_name = ? AND menu_type = ?", "系统管理", models.MenuTypeDir).First(&parent).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Warning: 未找到「系统管理」根菜单, migration_238 跳过菜单与授权 (非阻断): %v", err)
			// sys_config seed 仍照常执行（不依赖菜单树）
			return seedADDriftAlertConfigs(db)
		}
		return fmt.Errorf("查询系统管理根菜单失败: %w", err)
	}
	parentID := parent.ID

	// 2) 幂等创建 / 复用 C 型页面菜单
	menuID, err := ensureADDriftAlertPageMenu(db, parentID)
	if err != nil {
		return err
	}
	// C 菜单精准授权给已关联「系统管理」根的角色
	if err := grantMenuToParentRoles(db, menuID, parentID); err != nil {
		log.Printf("Migration 238: WARNING 精准授权 C 菜单失败(非阻断): %v", err)
	}

	// 3) 4 个 F 型按钮权限（父 = 新 C 菜单，精准授权给持父菜单的角色）
	for _, seed := range adDriftAlertFTypeSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 238: WARNING 查询父菜单失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 238: WARNING 未找到 live 父菜单(system:adDriftAlert), 跳过(perms=%s)", seed.perms)
			continue
		}
		btnID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 238: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}
		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, btnID, parentIDs); err != nil {
			log.Printf("Migration 238: WARNING 授权按钮菜单失败(perms=%s): %v, 跳过授权", seed.perms, err)
			continue
		}
	}

	// 4) 9 行 sys_config 默认 seed（幂等，缺行才插）
	if err := seedADDriftAlertConfigs(db); err != nil {
		return err
	}

	log.Println("Migration 238 completed: ad drift alert menu + permissions + 9 sys_config seeds ready")
	return nil
}

// ensureADDriftAlertPageMenu count-then-insert C 型页面菜单，返回菜单 ID。
func ensureADDriftAlertPageMenu(db *gorm.DB, parentID string) (string, error) {
	const menuName = "同步漂移告警"
	var existing models.Menu
	err := db.Where("menu_name = ?", menuName).First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("查询既有菜单 %s 失败: %w", menuName, err)
	}
	path := "ad-drift-alert"
	component := "system/ad-drift-alert/index"
	perms := "system:adDriftAlert"
	icon := "#"
	menu := &models.Menu{
		MenuName:  menuName,
		ParentID:  &parentID,
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeMenu, // 'C'
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
		Icon:      &icon,
		OrderNum:  92,
		Remark:    "AD 同步漂移告警通道（状态/触发记录/配置/手动触发）",
	}
	if err := db.Create(menu).Error; err != nil {
		return "", fmt.Errorf("创建菜单 %s 失败: %w", menuName, err)
	}
	log.Printf("Created menu: %s (ID: %s, perms: %s)", menuName, menu.ID, perms)
	return menu.ID, nil
}

// seedADDriftAlertConfigs 幂等插入 9 行 sys_config（按 config_key count-then-insert）。
func seedADDriftAlertConfigs(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Config{}) {
		log.Printf("Migration 238: WARNING sys_config 表不存在, 跳过 seed (非阻断)")
		return nil
	}
	for _, seed := range adDriftAlertConfigSeeds {
		var count int64
		if err := db.Model(&models.Config{}).
			Where("config_key = ?", seed.key).
			Count(&count).Error; err != nil {
			return fmt.Errorf("统计 sys_config(%s) 失败: %w", seed.key, err)
		}
		if count > 0 {
			continue // 已存在（含用户改过的值）→ 不覆盖，幂等
		}
		if err := db.Create(&models.Config{
			ConfigKey:   seed.key,
			ConfigName:  seed.name,
			ConfigValue: seed.value,
		}).Error; err != nil {
			return fmt.Errorf("插入 sys_config(%s) 失败: %w", seed.key, err)
		}
		log.Printf("Migration 238: seeded sys_config %s = %q", seed.key, seed.value)
	}
	return nil
}
