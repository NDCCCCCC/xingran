package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// ============================================================================
// Migration 257 — Phase 158 / CLEANUP-01~02：RPA 批量报告表 + 定时调度权限码
//
// 背景：
//   - CLEANUP-01：批量执行报告此前返回硬编码常量（100/95/5），真值无处可落。
//     本迁移建 sys_rpa_batch_item_reports（逐项粒度），报告读侧据此真统计。
//   - CLEANUP-02：sys_rpa_schedules 此前零服务零 API（D18），本 phase 补齐
//     CRUD + 调度注册，故需要 4 个新权限码的 F 型按钮播种载体。
//   - sys_rpa_schedules 表在部分部署里从未建过（models/rpa/schedule.go 有模型、
//     但历史归档 SQL 是否建表不确定）→ 本迁移在表缺失时补建（HasTable 守卫）。
//
// 幂等：HasTable 守卫 + runAndRecord 版本记录，重复执行无副作用。
// 双方言：PG / sqlite 通用（无方言专属 DDL；模型 tag 无 PG-only 片段）。
// 非阻断降级：权限播种单条失败仅记录日志并继续（同 migration_245 约定）。
// ============================================================================

// rpaSchedulePermissionFTypeSeedCodes F 型声明块（供前端 check-permission-codes.mjs
// 的 F_TYPE 解析器识别；必须与 rpaSchedulePermissionSeeds 逐条 1:1 一致）。
// F_TYPE_SEED_BEGIN
var rpaSchedulePermissionFTypeSeedCodes = []string{
	// 1. ops:rpa:schedule:list（调度查询，父=RPA 管理）
	"ops:rpa:schedule:list",

	// 2. ops:rpa:schedule:add（调度新增）
	"ops:rpa:schedule:add",

	// 3. ops:rpa:schedule:edit（调度编辑/启停/立即触发）
	"ops:rpa:schedule:edit",

	// 4. ops:rpa:schedule:remove（调度删除）
	"ops:rpa:schedule:remove",
}
// F_TYPE_SEED_END

// rpaSchedulePermissionSeeds 种子表（与上方声明块逐条一致）。
var rpaSchedulePermissionSeeds = []actionPermissionSeed{
	{name: "调度查询", perms: "ops:rpa:schedule:list", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 15},
	{name: "调度新增", perms: "ops:rpa:schedule:add", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 16},
	{name: "调度编辑", perms: "ops:rpa:schedule:edit", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 17},
	{name: "调度删除", perms: "ops:rpa:schedule:remove", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 18},
}

// Migrate257RPABatchReportAndSchedulePerm Phase 158 / CLEANUP-01~02。
func Migrate257RPABatchReportAndSchedulePerm(db *gorm.DB) error {
	log.Println("Running migration 257: RPA batch report table + schedule permissions")

	// 1) sys_rpa_batch_item_reports：批量执行项报告真值表
	if err := ensureRPABatchItemReportTable(db); err != nil {
		return fmt.Errorf("migration 257: 建 sys_rpa_batch_item_reports 失败: %w", err)
	}

	// 2) sys_rpa_schedules：表缺失时补建（存在则零改动，避免 PG 存量表漂移 ALTER）
	if err := ensureRPAScheduleTable(db); err != nil {
		return fmt.Errorf("migration 257: 建 sys_rpa_schedules 失败: %w", err)
	}

	// 3) 权限码 F 型按钮播种（复用 migration_226 全套 helper）
	if err := seedRPASchedulePermissions(db); err != nil {
		return fmt.Errorf("migration 257: 播种调度权限码失败: %w", err)
	}

	log.Println("Migration 257 completed: RPA batch report table + schedule permissions seeded")
	return nil
}

// ensureRPABatchItemReportTable 幂等建表（双方言）。
func ensureRPABatchItemReportTable(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db 为 nil")
	}
	if db.Migrator().HasTable(&rpamodels.BatchItemReport{}) {
		log.Println("Migration 257: sys_rpa_batch_item_reports 已存在，跳过建表")
		return nil
	}
	if err := db.AutoMigrate(&rpamodels.BatchItemReport{}); err != nil {
		return err
	}
	if !db.Migrator().HasTable(&rpamodels.BatchItemReport{}) {
		return fmt.Errorf("AutoMigrate 后仍无 sys_rpa_batch_item_reports 表")
	}
	log.Println("Migration 257: sys_rpa_batch_item_reports 已建表")
	return nil
}

// ensureRPAScheduleTable 幂等建表（双方言）。
func ensureRPAScheduleTable(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db 为 nil")
	}
	if db.Migrator().HasTable(&rpamodels.Schedule{}) {
		log.Println("Migration 257: sys_rpa_schedules 已存在，跳过建表")
		return nil
	}
	if err := db.AutoMigrate(&rpamodels.Schedule{}); err != nil {
		return err
	}
	if !db.Migrator().HasTable(&rpamodels.Schedule{}) {
		return fmt.Errorf("AutoMigrate 后仍无 sys_rpa_schedules 表")
	}
	log.Println("Migration 257: sys_rpa_schedules 已建表")
	return nil
}

// seedRPASchedulePermissions 播种调度权限码（非阻断降级）。
func seedRPASchedulePermissions(db *gorm.DB) error {
	// 声明块与种子表必须条数相同且逐条相等（漂移即致命，避免播种出错误的码）
	if len(rpaSchedulePermissionFTypeSeedCodes) != len(rpaSchedulePermissionSeeds) {
		return fmt.Errorf("migration 257 致命漂移: F_TYPE 块声明数(%d) != 种子表数(%d)",
			len(rpaSchedulePermissionFTypeSeedCodes), len(rpaSchedulePermissionSeeds))
	}
	for i, code := range rpaSchedulePermissionFTypeSeedCodes {
		if code != rpaSchedulePermissionSeeds[i].perms {
			return fmt.Errorf("migration 257 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, rpaSchedulePermissionSeeds[i].perms)
		}
	}

	for _, seed := range rpaSchedulePermissionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 257: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 257: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 257: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 257: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}
	return nil
}

// 编译期引用校验：确保 models 包引用不因重构失效。
var _ = models.MenuTypeButton
