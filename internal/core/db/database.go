package db

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/lib/pq"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// Database 数据库管理器
type Database struct {
	DB   *gorm.DB
	Type string
}

// NewDatabase 创建数据库连接
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	var db *gorm.DB
	var dbType string

	if cfg.Host != "" && cfg.Port > 0 {
		dbType = "postgres"
		var err error
		db, err = createPostgresConnection(cfg)
		if err != nil {
			return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
		}
	} else {
		dbType = "sqlite"
		var err error
		db, err = createSQLiteConnection(cfg)
		if err != nil {
			return nil, fmt.Errorf("连接SQLite失败: %w", err)
		}
	}

	return &Database{
		DB:   db,
		Type: dbType,
	}, nil
}

// createSQLiteConnection 创建SQLite连接
func createSQLiteConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dbPath := cfg.Host
	if dbPath == "" {
		dbPath = "./data/xingran.db"
	}

	gormConfig := &gorm.Config{
		Logger: createFilteredLogger(),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接SQLite失败: %w", err)
	}

	applogger.Infof("SQLite连接成功: %s", dbPath)
	return db, nil
}

// createFilteredLogger 创建使用 FilterLogger 的 GORM 配置
func createFilteredLogger() *FilterLogger {
	return NewFilterLogger(DefaultLogFilterConfig)
}

// createPostgresConnection 创建PostgreSQL连接
func createPostgresConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	adminDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode)

	if err := createDatabaseIfNotExists(adminDSN, cfg.DBName); err != nil {
		applogger.Errorf("创建数据库失败: %v", err)
	}

	dsn := cfg.GetDSN()

	gormConfig := &gorm.Config{
		Logger:                                   createFilteredLogger(),
		NowFunc:                                  func() time.Time { return time.Now().Local() },
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		// Supabase Pooler(Supavisor)兼容:pooler(Transaction/Session)不支持
		// 跨连接的 prepared statement 缓存,GORM 默认 PrepareStmt=true 会在 pooler 上
		// 反复 prepare/失败重试,导致 AutoMigrate(80+ DDL)卡死。设为 false 走 simple protocol。
		PrepareStmt: false,
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	configureConnectionPool(sqlDB, cfg)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	applogger.Infof("PostgreSQL连接成功")
	return db, nil
}

// configureConnectionPool 配置数据库连接池
func configureConnectionPool(sqlDB *sql.DB, cfg *config.DatabaseConfig) {
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.DB != nil {
		sqlDB, err := d.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// cleanupOldConstraints 清理可能存在的旧约束
//
// 设计原因: GORM AutoMigrate 与手写 SQL migration 混用导致 unique 约束命名冲突 ——
//   - SQL inline `UNIQUE`     → PG 自动命名 `<table>_<column>_key`
//   - GORM `uniqueIndex` tag  → GORM 拼接 `uni_<table>_<column>` 或显式名 `uni_*_*`
// GORM Migrator.DropConstraint 无 IF EXISTS,试图 DROP 自己命名的旧约束失败 → FATA。
// 本函数在 AutoMigrate 之前主动清理已知所有可能命名,让 AutoMigrate 走 ADD path 而非 DROP+ADD。
//
// 实现: 用 `DROP CONSTRAINT IF EXISTS`(PG 9.0+)单步替代旧的"SELECT count + DROP"两步,
// 更原子、更少 round-trip、对不存在的约束安全跳过。
func (d *Database) cleanupOldConstraints() {
	constraints := []struct {
		table      string
		constraint string
	}{
		// === 知识库 (SQL CONSTRAINT 命名 vs GORM uniqueIndex:idx_kb_*) ===
		{"sys_knowledge_category", "uk_knowledge_category_name"},
		{"sys_knowledge_tag", "uk_knowledge_tag_name"},
		{"sys_knowledge_tag", "uni_sys_knowledge_tag_tag_name"},
		// === 工单管理 ===
		{"sys_workorder_category", "uk_workorder_category_name"},
		{"sys_workorder_category", "uni_sys_workorder_category_name"},
		{"sys_workorder", "uk_workorder_no"},
		{"sys_workorder", "uni_sys_workorder_work_order_no"},
		// === 资产管理 (ops_asset.devicesn) ===
		// migration_148 用 inline UNIQUE → PG 自动名 ops_asset_devicesn_key
		// models.Asset.DeviceSN 标 uniqueIndex → GORM 期望 uni_ops_asset_devicesn
		{"ops_asset", "uni_ops_asset_devicesn"},
		{"ops_asset", "ops_asset_devicesn_key"},
		// === API Key (Phase 58/60) ===
		// BootstrapMissingTables 用 inline `key_hash ... UNIQUE` → PG 自动名 sys_api_keys_key_hash_key;
		// models.APIKey.KeyHash 标 uniqueIndex → GORM 期望 uni_sys_api_keys_key_hash。
		// 不带 bypass 启动时 AutoMigrate 会 DROP CONSTRAINT uni_...(无 IF EXISTS)→ 42704 FATA。
		{"sys_api_keys", "uni_sys_api_keys_key_hash"},
		{"sys_api_keys", "sys_api_keys_key_hash_key"},
		// === 系统配置 sys_config (2026-08-13 远端 DB 实测, 同类 42704) ===
		// sys_config 早期由 inline `config_key ... UNIQUE` 建表 → PG 自动名 sys_config_config_key_key
		// (pg_constraint contype=u); models.Config.ConfigKey 标 uniqueIndex → GORM 期望 uni_sys_config_config_key。
		// AutoMigrate 走 DropConstraint(无 IF EXISTS)→ 42704 FATA(实测 2026-08-13 21:53 部署期)。
		// 远端 pg_constraint contype=u 实测仅此一例 constraint 级冲突; user/role/post/dept/dict 等是
		// index 级 idx_* 与 GORM uni_* 共存(GORM CREATE 不报错, 非致命), 仅 sys_config 是 constraint 级致命。
		{"sys_config", "uni_sys_config_config_key"},
		{"sys_config", "sys_config_config_key_key"},
		// === VDI (128_create_vdi_tables.sql 多个 inline UNIQUE) ===
		{"sys_vdi_virtual_machines", "sys_vdi_virtual_machines_vm_id_key"},
		{"sys_vdi_virtual_machines", "uni_sys_vdi_virtual_machines_vm_id"},
		{"sys_vdi_resource_groups", "sys_vdi_resource_groups_resource_group_id_key"},
		{"sys_vdi_resource_groups", "uni_sys_vdi_resource_groups_resource_group_id"},
		// === RPA (102_add_rpa_tables.sql:97 name UNIQUE) ===
		// 注: 表名待定(需根据实际 SQL 确认), 留作占位由 AutoMigrate 路径走
		// === AD 域 (016_create_ad_domain_tables.sql named `uk_*` vs model `uni_sys_ad_*`) ===
		{"sys_ad_groups", "uk_ad_group_dn"},
		{"sys_ad_users", "uk_ad_user_dn"},
		{"sys_ad_ous", "uk_ad_ou_dn"},
		{"sys_ad_configs", "uk_ad_config_name"},
		{"sys_ad_group_members", "uk_ad_group_member"},
		// === 端口状态 (migration_177 重新应用 uniq_device_interface) ===
		// model.DevicePortStatus 现在有 uniqueIndex:uniq_device_interface,
		// AutoMigrate 期望此名,旧 SQL 约束(若有)在此清理
		{"sys_device_port_status", "uniq_device_interface"},
	}

	for _, c := range constraints {
		// 先检查表是否存在,不存在则跳过(避免对未创建的表 ALTER 触发 ERRO 日志)
		// 多数 sys_* / ops_* 表由后续 AutoMigrate 创建,此处 cleanup 阶段尚未存在
		if !d.DB.Migrator().HasTable(c.table) {
			continue
		}

		// PG 9.0+ 支持 `DROP CONSTRAINT IF EXISTS`,单步原子,不存在则安全跳过
		sql := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", c.table, c.constraint)
		if err := d.DB.Exec(sql).Error; err != nil {
			applogger.Debugf("清理约束 %s.%s 跳过: %v", c.table, c.constraint, err)
			continue
		}

		// 同步清理可能存在的同名索引(GORM 有时用 CREATE UNIQUE INDEX 而非 ADD CONSTRAINT)
		d.DB.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", c.constraint))
	}
}

// dropDependentMaterializedViews 在 AutoMigrate 之前 DROP 引用业务表的视图(MV + 普通 VIEW)。
//
// 设计原因: 视图(MV 和普通 VIEW)持久存在于数据库中。GORM AutoMigrate 每次启动都会
// ALTER 被引用表的列(类型/长度变化)。当列被视图引用时,PG 抛 SQLSTATE 0A000
// "cannot alter type of a column used by a view or rule",导致启动 FATA。
//
// 重要: 普通 VIEW 也会触发 0A000!
//   - 2026-07-04 17:35 验证:只 DROP MV reconciliation_normalized,启动时报
//     "ALTER TABLE ops_asset ALTER COLUMN devicesn TYPE varchar(200)
//      ERROR: cannot alter type of a column used by a view or rule"
//     因为 reconciliation_physical_chain 普通 VIEW 引用了 ops_asset.devicesn。
//   - 回归守护 memory gorm-automigrate-blocked-by-matview.md 描述的是 MV 场景,
//     未涵盖普通 VIEW。已补"普通 VIEW 也会阻塞 GORM ALTER TYPE"的子条目。
//
// 解决: AutoMigrate 之前 DROP 所有引用业务表的视图(MV + 普通 VIEW);
//       对应的 migration_NNN 会在 AutoMigrate 完成后重建。
//
// 当前已知视图(全部 DROP):
//   - reconciliation_normalized (MV, migration_168/176 创建,引用 sys_ad_user / sys_user /
//     reconciliation_user_lookup / reconciliation_physical_chain / ops_asset)
//   - reconciliation_physical_chain (普通 VIEW, migration_175 创建,引用 sys_device_mac_address /
//     sys_device_port_status / ops_info_points / sys_workstation / sys_user / ops_asset.devicesn)
//   - reconciliation_user_lookup (普通 VIEW, migration_175 创建,引用 sys_user / sys_dept / ops_asset)
//
// MAC 历史 MV(mv_mac_*)引用 sys_device_mac_history,该表无 GORM ALTER 需求,无需在此 DROP。
//
// 优化历史:
//   - 260704-ne5: 误删此函数 → SQLSTATE 0A000 FATA,regression-fix 恢复
//   - 260704-round3 (17:30): 误以为"普通 VIEW 不阻塞"只 DROP MV → 0A000 在 devicesn ALTER 复现
//   - 260704-round3 (17:38): 回滚,恢复 3 视图全 DROP
//
// 回归守护: docs/.planning/memory/gorm-automigrate-blocked-by-matview.md
func (d *Database) dropDependentMaterializedViews() {
	// 2026-07-04 21:05 round-3 教训:DROP VIEW reconciliation_physical_chain
	// **永远会失败**(SQLSTATE 2BP01 "other objects depend on it")。
	//
	// 原因:reconciliation_normalized MV LEFT JOIN reconciliation_physical_chain /
	// reconciliation_user_lookup → MV 依赖普通 VIEW → RESTRICT 也无法 DROP。
	// 而 CASCADE 会级联 DROP MV(2026-07-04 19:56 教训)。
	//
	// 新设计 (260704-regression-fix-5):
	//   - DROP 普通 VIEW 是冗余的(永远不会成功,且破坏依赖)
	//   - 175 用 CREATE OR REPLACE VIEW 已经 idempotent 重建视图定义
	//   - DROP reconciliation_normalized MV 不再需要(GORM 不发 ALTER,见上面注释)
	//
	// 此函数现在 noop + 仅 log 文档。如果将来 model tag 改回 size:N,GORM 重新发 ALTER,
	// 0A000 会再次出现 → 届时恢复 DROP 普通 VIEW + DROP MV (CASCADE)。
	applogger.Infof("[dropDependent] 当前架构下 noop(reconciliation_normalized MV 保留,普通 VIEW 由 175 CREATE OR REPLACE 重建)。" +
		"如需恢复 DROP,见 internal/core/db/database.go 历史 commit。")
}

// MigrateModelList 返回 AutoMigrate 注册的完整模型列表。
//
// 抽取为独立函数的原因 (2026-08-13, debug backend-hang-on-automigrate):
// scripts/dbprovision 一次性补建工具复用同一列表(仅对缺失表 CREATE),
// 保持单一事实源,避免两处列表漂移再次出现 "表永不会被建" 的问题。
func MigrateModelList() []interface{} {
	return []interface{}{
		&models.User{},
		&models.UserRole{},
		&models.Department{},
		&models.Role{},
		&models.Menu{},
		&models.Post{},
		&models.RoleMenu{},
		&models.RoleDept{},
		&models.UserPost{},
		&models.Job{},
		&models.JobLog{},
		// 网络设备相关模型
		&models.AuthCredential{},
		&models.NetworkDevice{},
		&models.ConfigTemplate{},
		&models.ConfigExecution{},
		&models.ConfigExecutionDetail{},
		&models.ConfigBackup{},
		&models.DeviceMACAddress{},
		&models.DevicePortStatus{},
		&models.PortWriteAudit{}, // Phase 52: 端口写审计 append-only 表
		&models.DeviceDiscovery{},
		&models.DeviceEnrichmentTask{},
		// 通知公告相关模型（增强版）
		&models.Notice{},
		&models.NoticeTarget{},
		&models.NoticeRead{},
		&models.NoticeAttachment{},
		// NotificationChannel 已通过 SQL 迁移创建，不需要 AutoMigrate
		// 通知配置相关模型
		&models.EmailConfig{},
		&models.APINotificationConfig{},
		// API Key 管理 (Phase 58/60 修复: AutoMigrate 缺漏导致 sys_api_keys 永不会被建)
		&models.APIKey{},
		&models.APIKeyUsageLog{},
		// 系统参数配置 sys_config (init_data.go 写入,缺则 seed 中途失败)
		&models.Config{},
		// 验证码背景图相关模型
		&models.CaptchaBackground{},
		// 值班管理相关模型
		&models.DutyPool{},
		&models.DutyPoolMember{},
		&models.DutyScheduleConfig{},
		&models.DutySchedule{},
		&models.DutyExchange{},
		&models.Holiday{},
		&models.DutyConfig{},
		// 工单管理相关模型（排除有约束问题的表）
		// WorkOrder, WorkOrderCategory 已通过 SQL 迁移创建，不需要 AutoMigrate
		&models.WorkOrderComment{},
		&models.WorkOrderHistory{},
		&models.WorkOrderRating{},
		&models.WorkOrderConfig{},
		&models.WorkOrderTemplate{},
		&models.PeriodicWorkOrderTemplate{},
		&models.PeriodicWorkOrderLog{},
		// 知识库相关模型（排除有约束问题的表）
		// KnowledgeCategory, KnowledgeTag 已通过 SQL 迁移创建，不需要 AutoMigrate
		&models.KnowledgeArticle{},
		&models.KnowledgeArticleTag{},
		// AD域管理相关模型
		&models.ADConfig{},
		&models.ADOU{},
		&models.ADGroup{},
		&models.ADUser{},
		&models.ADGroupMember{},
		&models.ADComputer{},
		&models.ADSyncLog{},
		// OU组映射相关模型（替代旧的部门组映射）
		&models.OUGroupMapping{},
		// VDI虚拟化相关模型
		&models.VDIVirtualMachine{},
		&models.VDIServer{},
		&models.VDIResourceGroup{},
		&models.VDIUserBinding{},
		// 运维管理相关模型（Workstation 复用系统已有的 sys_workstation）
		&operations.OpsBuilding{},
		&operations.OpsFloor{},
		&operations.OpsServerRoom{},
		&operations.OpsDedicatedLine{},
		&operations.OpsRoomDevice{},
		&operations.OpsInfoPoint{},
		&models.Asset{},
		// 仪表盘系统相关模型
		&models.Dashboard{},
		&models.DashboardVersion{},
		// Phase 46 R5: 半自动修复建议表
		&models.SysReconciliationFixSuggestion{},
	}
}

// AutoMigrate 自动迁移所有模型
//
// 设计变更 (260704-ne5): 启动期不再调用 200+ migration 函数 ——
//
//   1. 所有迁移(Migrate033/117/143~201)都是 schema 演进期一次性脚本,生产 DB 已全部应用过,
//      每次启动都是 idempotent noop,但累计产生 ~30s 启动开销 + 250 行启动日志噪音。
//
//   2. 新部署流程改为:
//        a) 用 scripts/db/snapshot.sh 从生产 DB 导出 schema-{date}.sql + seed-{date}.sql
//        b) 新机 psql -f schema-snapshot.sql && psql -f seed-snapshot.sql 一次性导入
//        c) ./xingran-backend 启动
//
//   3. GORM AutoMigrate(model) 仍保留 — 它是 model 加新字段时的安全网(ALTER TABLE ADD COLUMN)。
//      启动期只跑 ~2s,日志 1 行 "所有表迁移成功"。
//
//   4. cleanupOldConstraints() 仍保留 — 清理 GORM uniqueIndex 与 SQL inline UNIQUE 命名冲突,
//      ~5ms,无日志噪音。
//
//   5. dropDependentMaterializedViews() 已恢复(260704-ne5-regression-fix)——
//      GORM AutoMigrate 仍会 ALTER sys_ad_user.username 等被 MV 引用的列,导致 PG SQLSTATE 0A000 FATA。
//      前置 DROP 引用业务表的 MV,AutoMigrate 完成后由 migrations.Migrate176ReconciliationPhysicalMV
//      重建 reconciliation_normalized(MV 重建 ~10s 是 R5 双源+物理链路版本的固有开销)。
//
//   6. migrateCredentialModel() 已删除 — 凭证模型已在生产迁移完成(protocol_type 列已存在)。
//
// 所有 migrations.MigrateNNN 函数定义仍保留在 internal/core/db/migrations/,仅不再启动期调用。
// 如需重放某次迁移,手动跑对应函数(d.DB)即可。
func (d *Database) AutoMigrate() error {
	// SKIP_AUTOMIGRATE=true: dev/调试旁路,跳过 cleanup + dropDependent + AutoMigrate,
	// 仅执行 BootstrapMissingTables(在 core.go initDBAndData 中分支调用)。
	// Supabase pooler 上 cleanupOldConstraints 也会卡死,故一并跳过。
	if os.Getenv("SKIP_AUTOMIGRATE") == "true" {
		applogger.Warnf("[SKIP_AUTOMIGRATE=true] 跳过 cleanup/dropDependent/AutoMigrate,改由 BootstrapMissingTables 补建")
		return nil
	}

	// 先清理可能存在的旧约束。dropDependentMaterializedViews() 当前 noop,
	// 保留调用便于将来 model tag 漂移时回退 (260704-regression-fix-5)。
	if d.Type == "postgres" {
		d.cleanupOldConstraints()
		d.dropDependentMaterializedViews()
	}

	// 禁用外键约束的自动创建，避免类型不匹配的问题
	// 外键约束已通过 SQL 脚本手动创建
	err := d.DB.Migrator().AutoMigrate(MigrateModelList()...)
	if err != nil {
		return err
	}

	applogger.Infof("所有表迁移成功")

	// 所有迁移完成后,审计约束命名是否一致,提前暴露潜在 AutoMigrate 冲突
	d.auditConstraintNaming()

	// 重建被 dropDependentMaterializedViews() DROP 的 reconciliation MV
	// (R5 双源 declared + 真物理链路版本,migration_176 内置 DROP+CREATE 重建)。
	// 必须调用而非跳过的原因:cron `对账-物化视图刷新` 用 REFRESH CONCURRENTLY,
	// 需要视图存在才能成功;若不重建,下次 cron 任务会 FATA。
	// 重建顺序固定:175 先建前置 VIEW → 176 再建 MV(MV 引用 175 的 VIEW)。
	// migration_175 用 CREATE OR REPLACE VIEW,完全 idempotent;~5s。
	// migration_176 用 DROP+CREATE MATERIALIZED VIEW,~10s(R5 双源 declared + 物理链路版本固有成本)。
	if d.Type == "postgres" {
		if err := migrations.Migrate175ReconciliationPhysicalLink(d.DB); err != nil {
			applogger.Errorf("reconciliation 前置视图重建失败 (非阻断,留待下次启动): %v", err)
		}
		if err := migrations.Migrate176ReconciliationPhysicalMV(d.DB); err != nil {
			applogger.Errorf("reconciliation_normalized MV 重建失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 52: 端口写审计表(由 AutoMigrate 建表) + 菜单 seed + 角色授权
		if err := migrations.Migrate202PortWriteAudit(d.DB); err != nil {
			applogger.Errorf("端口写审计迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// 连接池配置 sys_config seed (max_connections / max_idle_seconds, web 可配, 默认 50/300s)
		if err := migrations.Migrate203ConnectionPoolSysConfig(d.DB); err != nil {
			applogger.Errorf("连接池配置 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// 锐捷 dot1x default-user-limit 缓存列 (enable 必须显式恢复 N)
		if err := migrations.Migrate204AddDot1xUserLimit(d.DB); err != nil {
			applogger.Errorf("dot1x_user_limit 列迁移失败 (非阻断,留待下次启动): %v", err)
		}
	}

	return nil
}

// auditConstraintNaming 启动期审计 unique 约束命名一致性
//
// 背景: GORM `uniqueIndex` tag 期望约束/索引名为 `uni_<table>_<column>`;
// 而 SQL inline `UNIQUE` 会被 PostgreSQL 自动命名为 `<table>_<column>_key`。
// 两者并存且 model 有 uniqueIndex tag 时,AutoMigrate 可能 DROP `uni_*`(无 IF EXISTS)→ FATA。
//
// 本审计仅扫描 PG 端 `_key` 后缀的 unique 约束并以 INFO 提示 —— 这些"潜在风险"约束
// 不一定造成实际问题(取决于对应 model 是否有冲突的 uniqueIndex tag):
//   - 若 model 该字段无 uniqueIndex tag(如纯关联表的 PK,或仅做 NOT NULL UNIQUE):安全,可忽略
//   - 若 model 该字段有 uniqueIndex tag:需要把 {table, <table>_<col>_key} 加入 cleanupOldConstraints,
//     并最好把 SQL 改为显式命名 `CONSTRAINT uni_<table>_<col> UNIQUE` 与 GORM 对齐
func (d *Database) auditConstraintNaming() {
	if d.Type != "postgres" {
		return
	}

	type auditRow struct {
		TableName      string `gorm:"column:table_name"`
		ConstraintName string `gorm:"column:constraint_name"`
	}

	var rows []auditRow
	err := d.DB.Raw(`
		SELECT
			conrelid::regclass::text AS table_name,
			conname AS constraint_name
		FROM pg_constraint
		WHERE contype = 'u'
		  AND conname LIKE '%\_key' ESCAPE '\'
		  AND (conrelid::regclass::text LIKE 'sys\_%' ESCAPE '\'
		    OR conrelid::regclass::text LIKE 'ops\_%' ESCAPE '\')
		ORDER BY table_name, constraint_name
	`).Scan(&rows).Error

	if err != nil {
		applogger.Debugf("约束命名审计跳过: %v", err)
		return
	}

	if len(rows) == 0 {
		applogger.Infof("[约束审计] 所有 unique 约束命名规范一致")
		return
	}

	// 改为 DEBUG 级别:这 8 个 PG 自动命名 unique 约束是历史关联表/复合 unique 的稳定结构,
	// 真正冲突会在 AutoMigrate 阶段 FATA,届时根据错误信息加入 cleanupOldConstraints。
	// DEBUG 保留诊断能力,但不污染每次启动的 INFO 日志(降噪 ~10 行)。
	applogger.Debugf("[约束审计] 发现 %d 个 PG 自动命名 unique 约束(若对应 model 无 uniqueIndex tag 可忽略):", len(rows))
	for _, r := range rows {
		applogger.Debugf("  · %s.%s", r.TableName, r.ConstraintName)
	}
	applogger.Debugf("[约束审计] 如遇 `DROP CONSTRAINT uni_*` FATA,把 {表名,约束名} 加入 cleanupOldConstraints 即可")
}

// InitData 初始化基础数据
func (d *Database) InitData() error {
	// 两种数据库都使用相同的初始化逻辑
	return initData(d.DB)
}

// BootstrapMissingTables 在跳过 AutoMigrate 时,直接用 raw SQL 补建缺失表。
//
// 设计原因: Supabase pooler (Session mode 5432) 上 GORM AutoMigrate(80+ DDL)
// 会卡死在 dropDependent 之后,所有表都在但 sys_api_keys + sys_api_key_usage_logs
// 永远建不出来 (database.go AutoMigrate 注册列表里也没加 &models.APIKey{})。
// 本函数用 d.DB.Exec() 走 simple protocol 单条 DDL,避开 GORM AutoMigrate 的批量
// statement 优化路径 —— 在 pooler 上更稳定。
//
// 安全: CREATE TABLE / INDEX IF NOT EXISTS,幂等。
// 适用: dev/调试;生产不应绕过 AutoMigrate。
func (d *Database) BootstrapMissingTables() error {
	ddl := []string{
		// Phase 58 SC#1-SC#4 + Phase 60 AUTH-03 + SEC-01 都依赖此表
		`CREATE TABLE IF NOT EXISTS public.sys_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version BIGINT,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    salt VARCHAR(32) NOT NULL,
    key_prefix VARCHAR(12) NOT NULL,
    user_id UUID,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    ip_whitelist JSONB NOT NULL DEFAULT '[]'::jsonb,
    description VARCHAR(500),
    inherit_perms BOOLEAN NOT NULL DEFAULT FALSE
)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON public.sys_api_keys(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON public.sys_api_keys(key_prefix)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON public.sys_api_keys(deleted_at)`,
		`CREATE TABLE IF NOT EXISTS public.sys_api_key_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL,
    user_id UUID NOT NULL,
    method VARCHAR(10),
    path VARCHAR(500),
    status_code INTEGER,
    client_ip VARCHAR(50),
    user_agent TEXT,
    duration INTEGER,
    success BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_api_key_id ON public.sys_api_key_usage_logs(api_key_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_created_at ON public.sys_api_key_usage_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_user_id ON public.sys_api_key_usage_logs(user_id)`,
	}

	for i, stmt := range ddl {
		applogger.Infof("[BootstrapMissingTables] executing %d/%d", i+1, len(ddl))
		if err := d.DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("DDL[%d] failed: %w", i+1, err)
		}
	}
	applogger.Infof("[BootstrapMissingTables] %d statements OK", len(ddl))
	return nil
}

// dbIdentRe 校验 PG 数据库标识符,防止 CREATE DATABASE 拼接时注入非法字符
var dbIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// createDatabaseIfNotExists 如果数据库不存在则创建
func createDatabaseIfNotExists(adminDSN, dbName string) error {
	// CREATE DATABASE 不支持 $1 占位符,必须拼接标识符,故先校验 dbName 合法性
	if !dbIdentRe.MatchString(dbName) {
		return fmt.Errorf("非法数据库名 %q（仅允许 [a-zA-Z_][a-zA-Z0-9_]*）", dbName)
	}

	db, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("连接管理员数据库失败: %w", err)
	}
	defer db.Close()

	// 检查数据库是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)"
	err = db.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("检查数据库是否存在失败: %w", err)
	}

	// 如果数据库不存在，则创建
	if !exists {
		createQuery := fmt.Sprintf("CREATE DATABASE %s", pq.QuoteIdentifier(dbName))
		_, err = db.Exec(createQuery)
		if err != nil {
			return fmt.Errorf("创建数据库失败: %w", err)
		}
		applogger.Infof("数据库 %s 创建成功", dbName)
	} else {
		applogger.Infof("数据库 %s 已存在", dbName)
	}

	return nil
}
