//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate201Phase48ComponentColumns Phase 48 Wave 1: 网络设备组件序列号采集 — schema 扩展
//
// 业务背景:
//   - D-01 / D-03 / D-05: ops_asset 扩展 4 列承载组件序列号对账所需的父交换机/采集来源/类型/槽位。
//     主设备(交换机/路由器整机)这 4 列保持 NULL;组件行(板卡/电源/风扇/光模块)采集器 UPDATE-only 写入。
//   - D-06: sys_data_reconciliation 新增 sibling 列 recon_category varchar(32),与 conflict_type(A-F)
//     正交。组件序列号对账异常 = conflict_type='F'(缺数据) + recon_category='component_serial'。
//     DROP 旧 partial unique uniq_recon_asset_type_open(asset_id, conflict_type),
//     新建 uniq_recon_asset_type_cat_open(asset_id, conflict_type, recon_category) WHERE open。
//   - D-11 防告警风暴语义不变:重复采集对同一 (asset, conflict_type, recon_category) 仍只产生 1 条 open。
//
// 步骤清单:
//   1. SQLite 分支:仅 AutoMigrate(由 GORM tag 自动建 4+1 列),跳过 partial unique index。
//   2. PostgreSQL 分支:ALTER ops_asset ADD 4 列(IF NOT EXISTS) + ALTER sys_data_reconciliation ADD 1 列。
//   3. 5 个普通 index(4 个 idx_ops_asset_* + 1 个 idx_recon_category)IF NOT EXISTS。
//   4. 索引切换:DROP uniq_recon_asset_type_open → CREATE uniq_recon_asset_type_cat_open (DO $$ 块幂等)。
//   5. 字典 seed:dict_type asset_reconciliation_recon_category + 2 条 dict_data(component_serial/future_expansion)。
//
// 幂等保证:
//   - 所有 ALTER 用 ADD COLUMN IF NOT EXISTS(PG 9.6+ 支持)
//   - 所有 CREATE INDEX 用 IF NOT EXISTS
//   - 索引切换用 DROP IF EXISTS + DO $$ 块检查
//   - 字典 seed 用 count-then-insert(migration_169 风格)
//
// 安全性:
//   - 不用 AutoMigrate 改 ops_asset/sys_data_reconciliation 现有列,避免触碰 deleted_at 等触发不必要的
//     ALTER TYPE(xingran-gorm-sql-constraint-naming-conflict 教训);新列由显式 ALTER 创建。
//   - DROP uniq_recon_asset_type_open 不影响现有 124 条 D 异常(conflict_type 不变,recon_category=NULL
//     仍满足新约束的 NULL 语义 — 同一 asset 的不同 recon_category 视为不同异常行)。
func Migrate201Phase48ComponentColumns(db *gorm.DB) error {
	log.Println("Running migration 201: Phase 48 Wave 1 component serial schema (ops_asset 4 cols + sys_data_reconciliation.recon_category)")

	// === SQLite 分支:仅 AutoMigrate,GORM tag 自动建 4+1 列;SQLite 不支持 partial unique index ===
	if !isPostgreSQL(db) {
		if err := db.AutoMigrate(&models.Asset{}, &models.SysDataReconciliation{}); err != nil {
			return fmt.Errorf("SQLite AutoMigrate Phase 48 列失败: %w", err)
		}
		log.Println("Migration 201 completed (non-PostgreSQL dialect: AutoMigrate only, partial unique index 跳过)")
		return nil
	}

	// === PostgreSQL 分支 ===

	// 1. ALTER ops_asset ADD 4 列(显式 IF NOT EXISTS,避免 AutoMigrate 触发现有列的 ALTER TYPE)
	opsAssetCols := []struct {
		col  string
		defn string
	}{
		{"parent_asset_id", "VARCHAR(64)"},
		{"source_device_id", "VARCHAR(64)"},
		{"component_type", "VARCHAR(32)"},
		{"component_slot", "VARCHAR(64)"},
	}
	for _, c := range opsAssetCols {
		sql := fmt.Sprintf("ALTER TABLE ops_asset ADD COLUMN IF NOT EXISTS %s %s", c.col, c.defn)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("ALTER ops_asset ADD %s 失败: %w", c.col, err)
		}
	}
	applogger.Infof("[迁移] Migration 201: ops_asset 4 新列已就位 (parent_asset_id / source_device_id / component_type / component_slot)")

	// 2. ALTER sys_data_reconciliation ADD recon_category VARCHAR(32)
	if err := db.Exec("ALTER TABLE sys_data_reconciliation ADD COLUMN IF NOT EXISTS recon_category VARCHAR(32)").Error; err != nil {
		return fmt.Errorf("ALTER sys_data_reconciliation ADD recon_category 失败: %w", err)
	}
	applogger.Infof("[迁移] Migration 201: sys_data_reconciliation.recon_category 列已就位")

	// 3. 5 个普通 index(4 个 idx_ops_asset_* + 1 个 idx_recon_category)
	opsAssetIndexes := []struct {
		idx string
		col string
	}{
		{"idx_ops_asset_parent_asset_id", "parent_asset_id"},
		{"idx_ops_asset_source_device_id", "source_device_id"},
		{"idx_ops_asset_component_type", "component_type"},
	}
	for _, ix := range opsAssetIndexes {
		// component_type 大量 NULL,加 WHERE deleted_at IS NULL 过滤减小索引体积;同时与 List() 默认
		// WHERE component_type IS NULL 的查询路径相容(PG 部分索引匹配 IS NULL 谓词)
		sql := fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON ops_asset (%s) WHERE deleted_at IS NULL",
			ix.idx, ix.col,
		)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("CREATE INDEX %s 失败: %w", ix.idx, err)
		}
	}
	// component_slot 选择度低且多用于精确查找,普通 index 即可(不加 WHERE,简化)
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_ops_asset_component_slot ON ops_asset (component_slot)").Error; err != nil {
		return fmt.Errorf("CREATE INDEX idx_ops_asset_component_slot 失败: %w", err)
	}
	// recon_category 单独 index,WHERE deleted_at IS NULL 与 uniq_recon 索引语义一致
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_recon_category ON sys_data_reconciliation (recon_category) WHERE deleted_at IS NULL").Error; err != nil {
		return fmt.Errorf("CREATE INDEX idx_recon_category 失败: %w", err)
	}
	applogger.Infof("[迁移] Migration 201: 5 个普通 index 已就位")

	// 4. 索引切换(D-06 核心):DROP uniq_recon_asset_type_open → CREATE uniq_recon_asset_type_cat_open
	if err := db.Exec("DROP INDEX IF EXISTS uniq_recon_asset_type_open").Error; err != nil {
		return fmt.Errorf("DROP uniq_recon_asset_type_open 失败: %w", err)
	}
	// DO $$ 块 + pg_indexes 检查保证幂等(参考 migration_168 模式)
	newPartialUniqueSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_recon_asset_type_cat_open'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE UNIQUE INDEX uniq_recon_asset_type_cat_open
                 ON sys_data_reconciliation (asset_id, conflict_type, recon_category)
                 WHERE resolved_at IS NULL AND deleted_at IS NULL';
    END IF;
END$$;
`
	if err := db.Exec(newPartialUniqueSQL).Error; err != nil {
		return fmt.Errorf("CREATE uniq_recon_asset_type_cat_open 失败: %w", err)
	}
	applogger.Infof("[迁移] Migration 201: partial unique index 已切换 uniq_recon_asset_type_open → uniq_recon_asset_type_cat_open")

	// 5. 字典 seed:dict_type asset_reconciliation_recon_category + 2 条 dict_data
	if err := seedReconCategoryDict(db); err != nil {
		return fmt.Errorf("字典 seed asset_reconciliation_recon_category 失败: %w", err)
	}
	applogger.Infof("[迁移] Migration 201: 字典 asset_reconciliation_recon_category seed 完成")

	log.Println("Migration 201 completed: ops_asset 4 cols + sys_data_reconciliation.recon_category + uniq_recon_asset_type_cat_open + dict seed")
	return nil
}

// seedReconCategoryDict seed 1 个 dict_type + 2 条 dict_data
//
// count-then-insert 模式(migration_169 风格),重复运行不会产生重复行。
// dict_data 2 条:
//   - component_serial(默认)— Phase 48 唯一在用业务分类
//   - future_expansion(非默认)— 预留扩展位
func seedReconCategoryDict(db *gorm.DB) error {
	const dictTypeKey = "asset_reconciliation_recon_category"

	// 1. dict_type count-then-insert
	var typeCount int64
	if err := db.Model(&models.DictType{}).Where("dict_type = ?", dictTypeKey).Count(&typeCount).Error; err != nil {
		return err
	}
	if typeCount == 0 {
		dt := &models.DictType{
			DictName: "资产对账业务分类",
			DictType: dictTypeKey,
			Status:   0, // Status Convention 0=启用
			Remark:   "Phase 48: 与 conflict_type(A-F)正交的业务分类,当前唯一值 component_serial",
		}
		if err := db.Create(dt).Error; err != nil {
			log.Printf("Migration 201: create dict_type %s failed: %v", dictTypeKey, err)
		} else {
			log.Printf("Migration 201: created dict_type %s", dictTypeKey)
		}
	}

	// 2. dict_data 按 (dict_type, dict_value) 幂等
	type dataSpec struct {
		label     string
		value     string
		listClass string
		isDefault bool
	}
	dataValues := []dataSpec{
		{"组件序列号", "component_serial", "warning", true},
		{"预留扩展", "future_expansion", "default", false},
	}

	for _, dv := range dataValues {
		var dataCount int64
		if err := db.Model(&models.DictData{}).
			Where("dict_type = ? AND dict_value = ?", dictTypeKey, dv.value).
			Count(&dataCount).Error; err != nil {
			return err
		}
		if dataCount > 0 {
			continue
		}
		listClass := dv.listClass
		dd := &models.DictData{
			DictSort:  0,
			DictLabel: dv.label,
			DictValue: dv.value,
			DictType:  dictTypeKey,
			ListClass: &listClass,
			IsDefault: dv.isDefault,
			Status:    0, // Status Convention 0=启用
		}
		if err := db.Create(dd).Error; err != nil {
			log.Printf("Migration 201: create dict_data %s/%s failed: %v", dictTypeKey, dv.value, err)
		}
	}
	return nil
}
