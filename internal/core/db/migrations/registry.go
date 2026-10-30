package migrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// ActiveMigrations 当前活跃（仍留在启动调用块中每次启动重跑）的全部迁移版本清单。
//
// 来源：internal/core/db/database.go 三个调用点集合的并集 ——
//   - PG 分支调用块（AutoMigrate 内 advisory-lock 块）：175,176,202-224（215 经 ensureADPushQueue）
//   - sqlite 分支调用块：207-213,215-224（215 经 ensureADPushQueue）
//   - SKIP_AUTOMIGRATE 旁路（BootstrapMissingTables）：215-220
//
// 维护约定：database.go 调用块每新增/移除一个 MigrateNNN 调用点，本清单必须同步增删，
// 否则 MissingRecords 对账会误报。记录表只读不控制执行 —— 本清单**不参与**任何迁移
// 是否运行的判断。
var ActiveMigrations = []struct {
	Version int
	Name    string
}{
	{175, "reconciliation_physical_link"},
	{176, "reconciliation_physical_mv"},
	{202, "port_write_audit"},
	{203, "connection_pool_sys_config"},
	{204, "dot1x_user_limit"},
	{205, "rpa_worker_id_default"},
	{206, "user_role_role_menu_indexes"},
	{207, "seed_canonical_menu_catalog"},
	{208, "dict_seed"},
	{209, "update_settings_menu_component"},
	{210, "normalize_menu_parent_id"},
	{211, "create_config_restore_task"},
	{212, "restore_task_active_unique_index"},
	{213, "recompute_building_total_floors"},
	{214, "create_ad_login_history"},
	{215, "create_ad_push_queue"},
	{216, "ad_push_drain_sys_config"},
	{217, "dept_ou_mismatch_view"},
	{218, "ad_reconciliation_permission"},
	{219, "user_ad_sync_fields"},
	{220, "manager_rule_tables"},
	{221, "manager_sync_menus"},
	{222, "drop_phantom_join_tables"},
	{223, "manager_org_partial_unique"},
	{224, "dept_sync_seed_and_ou_group_unique"},
	{225, "schema_migrations_table"},
	{226, "action_permission_f_type_seed"},
	{227, "action_permission_f_type_seed_hold"},
}

// ReconcileResult 对账结果：registry 与库内 sys_schema_migrations 记录的差异。
type ReconcileResult struct {
	// Missing：registry 有、表无（版本「应该」有记录但没有 —— 典型：SKIP_AUTOMIGRATE
	// 旁路只跑 215-220 子集，其余版本在新旁路库上即 Missing；或 225 未执行过）。
	Missing []int
	// Extra：表有、registry 无（版本号已从调用块移除但记录残留，或人工插入的未知版本）。
	Extra []int
}

// MissingRecords 只读对账：比较 ActiveMigrations 与库内 sys_schema_migrations，
// 返回 missing/extra 两个方向差集。**只读** —— 不写库、不触发任何迁移。
//
// 方言提示：ActiveMigrations 含 PG-only 版本（175/176/202-206/211/214），在 sqlite 库上
// 这些版本天然 Missing，属预期方言差异（见 migration_225 注释），消费方自行甄别。
// 表不存在时返回错误（调用方应先确保 migration_225 已执行）。
func MissingRecords(ctx context.Context, db *gorm.DB) (*ReconcileResult, error) {
	if db == nil {
		return nil, fmt.Errorf("db 为 nil")
	}
	if !db.Migrator().HasTable("sys_schema_migrations") {
		return nil, fmt.Errorf("sys_schema_migrations 表不存在（migration_225 尚未执行）")
	}

	type row struct {
		Version int
	}
	var rows []row
	if err := db.WithContext(ctx).
		Table("sys_schema_migrations").
		Order("version").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询 sys_schema_migrations 失败: %w", err)
	}

	inDB := make(map[int]bool, len(rows))
	for _, r := range rows {
		inDB[r.Version] = true
	}
	inRegistry := make(map[int]bool, len(ActiveMigrations))
	for _, m := range ActiveMigrations {
		inRegistry[m.Version] = true
	}

	result := &ReconcileResult{
		Missing: make([]int, 0),
		Extra:   make([]int, 0),
	}
	for _, m := range ActiveMigrations {
		if !inDB[m.Version] {
			result.Missing = append(result.Missing, m.Version)
		}
	}
	for v := range inDB {
		if !inRegistry[v] {
			result.Extra = append(result.Extra, v)
		}
	}
	return result, nil
}
