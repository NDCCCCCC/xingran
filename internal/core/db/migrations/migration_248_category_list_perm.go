package migrations

// Migration 248 — 修复「工单分类」读接口非超管恒 403：补 status=0 的 F 型授权载体。
//
// 背景：工单分类页已并入「工单管理」页 Tab（CategoryManagement.tsx），但
// internal/api/router.go:781-783 的 /workorder/categories 读组 RequirePermissions
// 只挂 ops:workorder:category:list 一个码。生产库里该码唯一载体是 C 行
// da379d50（visible=0/status=1）；pkg/middleware/permission.go:144-150 的精确匹配
// SQL 带 AND m.status = normal，Step2 hasInheritedButtonPerm(:203-208) 同样要求
// status=0 的载体行，Step3 模块降级 ops:workorder:category 无载体 ⇒ 非超管恒 403。
//
// 处置（照 migration_247 的 F 型种子声明 + migration_242 的「按 perms 复用既有行」
// 模式）：补/复用一条 menu_type='F' 的按钮行，perms 原值不变，status=0，
// parent_id 挂宿主「工单管理」C（component=workorder/orders/index，
// seed id 633bac92-d1e4-416c-bdce-16231a1a4c32，menu_catalog_seed.sql:38）。
//
// 幂等：先按 perms + menu_type='F' 查既有行（NOT EXISTS 语义：查到即复用、不重复插入）；
// 再以 DB 侧条件 UPDATE 把同 perms 的所有未软删 F 行统一 status=0 + parent_id=宿主；
// 授权沿用 migration_226 helper 的 NOT EXISTS 幂等（只 INSERT，不回收任何 sys_role_menu 行）。
// 不软删、不物理删、不改 perms、不改 menu_catalog_seed.sql。
//
// 双方言：纯 GORM 条件更新 + 通用字符串常量，无 PG 专属语法；sqlite 最小夹具下
// sys_role_menu 不存在时授权步骤由 HasTable 守卫跳过（非阻断）。
// 迁移每次启动重跑（runAndRecord 无版本跳过），连跑两次零重复。

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

const (
	categoryListPermCode      = "ops:workorder:category:list"
	categoryListHostSeedID    = "633bac92-d1e4-416c-bdce-16231a1a4c32"
	categoryListHostComp      = "workorder/orders/index"
	categoryListHostCompSlash = "/workorder/orders/index"
)

// F 型种子双声明（照 migration_247 的 xxxFTypeSeedCodes + xxxPermissionSeeds 一致性校验）。
// 刻意不使用 // F_TYPE_SEED_BEGIN ... END 标记：check-permission-codes.mjs 会把块内码
// 计入 menuFType 并触发 d 类「F 型按钮无 TS 声明」判定；该码在后端常量 / TS 声明侧的
// 缺失由 permission-drift-exemptions.json:340 的 grandfathered 豁免覆盖，重复声明只会
// 多引入一个依赖豁免的判定点。与 migration_242「不重复声明 F_TYPE_SEED 块」的先例一致。
var categoryListPermFTypeSeedCodes = []string{
	categoryListPermCode,
}

var categoryListPermSeeds = []actionPermissionSeed{
	{name: "工单分类查询", perms: categoryListPermCode, orderNum: 1},
}

// Migrate248CategoryListPerm 补 status=0 的 F 型授权载体（幂等 / 双方言 / 非阻断）。
func Migrate248CategoryListPerm(db *gorm.DB) error {
	log.Println("Running migration 248: seed status=0 F perm for ops:workorder:category:list")

	if !db.Migrator().HasTable(&models.Menu{}) {
		log.Println("Migration 248: WARNING sys_menu 表不存在, 跳过 (非阻断)")
		return nil
	}
	// 瘦库/残缺 sys_menu（缺 perms、menu_type 等列）不能静默降级：否则 runAndRecord
	// 会把「什么都没做成」记成成功版本，下次启动不再重试，且错误被 continue 吞掉。
	// 这里显式返回 error，交给调用方非阻断 WARN + 下次启动重试（与 241/242/243/249 同口径）。
	for _, col := range []string{"perms", "menu_type", "component", "parent_id", "status"} {
		if !db.Migrator().HasColumn(&models.Menu{}, col) {
			return fmt.Errorf("migration 248 sys_menu 缺列 %s, 留待下次启动重试", col)
		}
	}

	if len(categoryListPermFTypeSeedCodes) != len(categoryListPermSeeds) {
		return fmt.Errorf("migration 248 致命漂移: F_TYPE 声明数(%d) != 种子表数(%d)",
			len(categoryListPermFTypeSeedCodes), len(categoryListPermSeeds))
	}
	for i, code := range categoryListPermFTypeSeedCodes {
		if code != categoryListPermSeeds[i].perms {
			return fmt.Errorf("migration 248 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, categoryListPermSeeds[i].perms)
		}
	}

	hostID := findCategoryListHostMenuID(db)

	for _, seed := range categoryListPermSeeds {
		// 1) 按 perms + menu_type='F' 复用既有行（查到即复用，不重复插入）；
		//    不存在才新建，新建即 status=0 / parent=宿主。
		btnID, err := ensureCategoryListPermButton(db, seed, hostID)
		if err != nil {
			log.Printf("Migration 248: WARNING 创建/复用 F 行失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}

		// 2) 全量统一：同 perms 的所有未软删 F 行 → status=0 + parent=宿主。
		//    GORM 默认作用域自动过滤软删行；不删行、不改 perms、不动 sys_role_menu。
		if err := normalizeCategoryListPermButtons(db, seed.perms, hostID); err != nil {
			log.Printf("Migration 248: WARNING 统一 F 行状态失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}

		// 3) 照 migration_247 模式 NOT EXISTS 幂等精准授权（只 INSERT，不回收）。
		//    sqlite 最小夹具无 sys_role_menu 表 → HasTable 守卫跳过（非阻断）。
		if db.Migrator().HasTable("sys_role_menu") {
			if err := grantActionPermissionToParentHolders(db, btnID, []string{hostID}); err != nil {
				log.Printf("Migration 248: WARNING 授权 F 行失败(perms=%s): %v, 跳过授权", seed.perms, err)
			}
		}
	}

	log.Println("Migration 248 completed: status=0 F carrier for ops:workorder:category:list ready")
	return nil
}

// findCategoryListHostMenuID 定位「工单管理」宿主 C 菜单：
// 优先按 component 变体查启用中的 C 行，回退固定 seed id，最后以 seed id 常量兜底。
// 返回值恒非空（seed 基线 id 本身就是该宿主的主键），保证极端瘦库也能补上 F 行。
func findCategoryListHostMenuID(db *gorm.DB) string {
	var host models.Menu
	err := db.Where("component IN ?", []string{categoryListHostComp, categoryListHostCompSlash}).
		Where("menu_type = ?", models.MenuTypeMenu).
		Where("status = ?", models.MenuStatusNormal).
		Order("id ASC").First(&host).Error
	if err == nil {
		return host.ID
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Migration 248: WARNING 查询宿主菜单失败: %v, 回退 seed 基线 id", err)
	}

	err = db.Where("id = ? AND menu_type = ?", categoryListHostSeedID, models.MenuTypeMenu).
		First(&host).Error
	if err == nil {
		return host.ID
	}
	log.Printf("Migration 248: WARNING 未找到「工单管理」宿主 C 菜单(%v), 使用 seed 基线 id 兜底", err)
	return categoryListHostSeedID
}

// ensureCategoryListPermButton 按 perms + menu_type='F' 复用既有行；不存在才新建。
// 与 migration_226 的 ensureActionPermissionMenu 同款（含 visible=0 显式修正，
// 规避 GORM default:1 标签吞掉零值）。
func ensureCategoryListPermButton(db *gorm.DB, seed actionPermissionSeed, hostID string) (string, error) {
	var existing models.Menu
	err := db.Where("perms = ? AND menu_type = ?", seed.perms, models.MenuTypeButton).
		Order("id ASC").First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("查询既有 F 行失败: %w", err)
	}

	emptyPath := ""
	icon := "#"
	perms := seed.perms
	button := &models.Menu{
		MenuName: seed.name,
		ParentID: &hostID,
		Path:     &emptyPath,
		MenuType: models.MenuTypeButton,
		Visible:  models.VisibleHidden,
		Status:   models.MenuStatusNormal, // status=0：权限中间件精确匹配要求
		Perms:    &perms,
		Icon:     &icon,
		OrderNum: seed.orderNum,
	}
	if err := db.Create(button).Error; err != nil {
		return "", fmt.Errorf("创建 F 行失败: %w", err)
	}
	if err := db.Model(&models.Menu{}).Where("id = ?", button.ID).
		Update("visible", models.VisibleHidden).Error; err != nil {
		return "", fmt.Errorf("显式更新 F 行 visible=0 失败: %w", err)
	}
	log.Printf("Migration 248: 新建 F 行 %s (id=%s, perms=%s, parent=%s)",
		seed.name, button.ID, seed.perms, hostID)
	return button.ID, nil
}

// normalizeCategoryListPermButtons 把同 perms 的所有未软删 F 行统一为
// status=0 + parent_id=宿主（幂等；值不变亦无害，不新增/删除任何行）。
// GORM 默认作用域自动追加 deleted_at IS NULL，软删行不受影响。
func normalizeCategoryListPermButtons(db *gorm.DB, perms, hostID string) error {
	res := db.Model(&models.Menu{}).
		Where("perms = ? AND menu_type = ?", perms, models.MenuTypeButton).
		Updates(map[string]interface{}{
			"status":    models.MenuStatusNormal,
			"parent_id": hostID,
		})
	if res.Error != nil {
		return fmt.Errorf("统一 F 行状态失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		log.Printf("Migration 248: 归位 %d 条 F 行 (perms=%s) → status=0/parent=%s",
			res.RowsAffected, perms, hostID)
	}
	return nil
}
