package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db/migrations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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

	if err := configureGORM(db); err != nil {
		return nil, fmt.Errorf("配置GORM失败: %w", err)
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
		dbPath = "./data/ruoyi.db"
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

// configureGORM 配置GORM
func configureGORM(_ *gorm.DB) error {
	return nil
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

// dropDependentMaterializedViews 在 AutoMigrate 之前 DROP 引用业务表的物化视图。
//
// 设计原因: 物化视图(MV)持久存在于数据库中。GORM AutoMigrate 每次启动都会尝试
// ALTER 被引用表的列(即使类型已匹配,GORM 仍会下发 ALTER TYPE 语句)。当列被 MV
// 引用时,PG 抛 SQLSTATE 0A000 "cannot alter type of a column used by a view or rule",
// 导致启动 FATA。
//
// 解决: AutoMigrate 之前 DROP 所有引用业务表的 MV;对应的 migration_NNN 会在
// AutoMigrate 完成后重新 CREATE 它们(用最新表结构 + 列类型)。
//
// 当前已知 MV:
//   - reconciliation_normalized (migration_168 创建,引用 sys_ad_user / sys_user / ops_asset)
//     → migration_168/176 末尾有 CREATE MATERIALIZED VIEW(非 IF NOT EXISTS,强制重建)
//   - reconciliation_physical_chain (migration_175 创建,普通视图,引用 sys_device_mac_address /
//     sys_device_port_status / ops_info_points / sys_workstation / sys_user)
//   - reconciliation_user_lookup (migration_175 创建,普通视图,引用 sys_user / sys_dept / ops_asset)
//
// MAC 历史 MV(mv_mac_*)引用 sys_device_mac_history,该表无 GORM ALTER 需求,无需在此 DROP。
func (d *Database) dropDependentMaterializedViews() {
	// 仅 DROP 引用会触发 ALTER 的表的视图(MV + 普通视图);CASCADE 一并清理其上的索引。
	//
	// 同一对象可能是 MATERIALIZED VIEW(reconciliation_normalized)也可能是普通 VIEW
	// (reconciliation_physical_chain / reconciliation_user_lookup)。用 DO $$ 块做兼容:
	// 先查 pg_class.relkind — 'm' 表 MV, 'v' 表 view,逐个分支 DROP。
	// 任何分支报错都不会阻断 AutoMigrate 流程。
	dependentViews := []string{
		"reconciliation_normalized",
		"reconciliation_physical_chain",
		"reconciliation_user_lookup",
	}
	for _, view := range dependentViews {
		dropSQL := fmt.Sprintf(`
DO $$
DECLARE
    kind char;
BEGIN
    SELECT relkind INTO kind FROM pg_class
     WHERE relname = '%s' AND relnamespace = 'public'::regnamespace;
    IF kind = 'm' THEN
        EXECUTE 'DROP MATERIALIZED VIEW IF EXISTS public.%s CASCADE';
    ELSIF kind = 'v' THEN
        EXECUTE 'DROP VIEW IF EXISTS public.%s CASCADE';
    END IF;
END$$;
`, view, view, view)
		if err := d.DB.Exec(dropSQL).Error; err != nil {
			applogger.Debugf("DROP VIEW/MV %s 跳过: %v", view, err)
			continue
		}
		applogger.Infof("[迁移] 预清理视图 %s(AutoMigrate 前置,后续 migration 重建)", view)
	}
}

// AutoMigrate 自动迁移所有模型
func (d *Database) AutoMigrate() error {
	// 先清理可能存在的旧约束
	if d.Type == "postgres" {
		d.cleanupOldConstraints()
		// 关键: 先 DROP 引用业务表的物化视图,否则 ALTER TABLE 被阻塞。
		// reconciliation_normalized 引用 sys_ad_user.username / sys_user.id / ops_asset,
		// 当 GORM AutoMigrate 试图 ALTER 这些表的列类型(如 sys_ad_user.username → varchar(255))
		// 时,PG 抛 SQLSTATE 0A000 "cannot alter type of a column used by a view or rule"。
		// migration_168 会在 AutoMigrate 完成后重新 CREATE 该 MV。
		d.dropDependentMaterializedViews()
	}

	// 禁用外键约束的自动创建，避免类型不匹配的问题
	// 外键约束已通过 SQL 脚本手动创建
	err := d.DB.Migrator().AutoMigrate(
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
	)
	if err != nil {
		return err
	}

	applogger.Infof("所有表迁移成功")

	// 执行凭证模型重新设计迁移
	if err := d.migrateCredentialModel(); err != nil {
		applogger.Errorf("凭证模型迁移失败: %v", err)
		// 迁移失败不中断启动，只记录日志
	}

	// 执行VDI虚拟机status字段移除迁移
	if err := migrations.Migrate143VDIRemoveVMStatus(d.DB); err != nil {
		applogger.Errorf("VDI虚拟机status字段移除迁移失败: %v", err)
	}
	// 执行VDI虚拟机细粒度权限迁移
	if err := migrations.Migrate144VDIGranularPermissions(d.DB); err != nil {
		applogger.Errorf("VDI虚拟机细粒度权限迁移失败: %v", err)
	}
	// 执行VDI虚拟机bound_user_id UUID转换迁移
	if err := migrations.Migration_145_FixBoundUserIDUUID(d.DB); err != nil {
		applogger.Errorf("VDI虚拟机bound_user_id UUID转换迁移失败: %v", err)
	}
	// 执行资产管理菜单和权限迁移
	if err := migrations.Migrate146AddAssetMenuPermissions(d.DB); err != nil {
		applogger.Errorf("资产管理菜单和权限迁移失败: %v", err)
	}
	// 修复资产管理菜单结构（添加缺失的C类型子菜单）
	if err := migrations.Migrate147FixAssetMenuStructure(d.DB); err != nil {
		applogger.Errorf("资产管理菜单结构修复迁移失败: %v", err)
	}

		// 创建资产管理表
		if err := migrations.Migrate148CreateOpsAssetTable(d.DB); err != nil {
			applogger.Errorf("创建资产管理表迁移失败: %v", err)
		}
		// 修复列配置visible字段默认值问题
		if err := migrations.Migrate149FixColumnVisibleDefault(d.DB); err != nil {
			applogger.Errorf("修复列配置visible字段默认值迁移失败: %v", err)
		}
		// 为工位设备关联表添加 IP 地址字段
		if err := migrations.Migrate150AddWorkstationDeviceIPAddress(d.DB); err != nil {
			applogger.Errorf("工位设备 IP 地址字段迁移失败: %v", err)
		}
		// Phase 15: 性能优化 (PERF-01 复合索引 + PERF-02 物化视图 DDL)
		if err := migrations.Migrate151MACPerfIndexes(d.DB); err != nil {
			applogger.Errorf("MAC 历史复合索引迁移失败: %v", err)
		}
		if err := migrations.Migrate152MACMatView(d.DB); err != nil {
			applogger.Errorf("MAC 历史物化视图迁移失败: %v", err)
		}
		// Phase 15 PERF-04: 端口使用热力图菜单与权限
		if err := migrations.Migrate153MacHeatmapMenu(d.DB); err != nil {
			applogger.Errorf("MAC 端口使用热力图菜单迁移失败: %v", err)
		}
		// Phase A: sys_user 服务端排序索引（user-driven ORDER BY 性能支撑）
		if err := migrations.Migrate154UserSortIndexes(d.DB); err != nil {
			applogger.Errorf("sys_user 排序索引迁移失败: %v", err)
		}
		// Phase B: 日志/楼宇/工位服务端排序索引
		if err := migrations.Migrate155OperLogSortIndexes(d.DB); err != nil {
			applogger.Errorf("sys_oper_log 排序索引迁移失败: %v", err)
		}
		if err := migrations.Migrate156LoginLogSortIndexes(d.DB); err != nil {
			applogger.Errorf("sys_logininfor 排序索引迁移失败: %v", err)
		}
		if err := migrations.Migrate157OpsBuildingSortIndexes(d.DB); err != nil {
			applogger.Errorf("ops_buildings 排序索引迁移失败: %v", err)
		}
		if err := migrations.Migrate158WorkstationSortIndexes(d.DB); err != nil {
			applogger.Errorf("ops_workstation 排序索引迁移失败: %v", err)
		}
		// ops 模块菜单 perms 命名对齐: 复数/连字符/remove → 单数/无连字符/delete
		// (与路由 RequirePermissions 及 system/network/asset 模块一致), 修复角色读写 403
		if err := migrations.Migrate159AlignOpsPerms(d.DB); err != nil {
			applogger.Errorf("ops 菜单 perms 对齐迁移失败: %v", err)
		}
		// Phase 28: 操作日志添加 nickname 字段，支持显示"nickname（username）"格式
		if err := migrations.Migration160OperLogAddNickname(d.DB); err != nil {
			applogger.Errorf("sys_oper_log nickname 字段迁移失败: %v", err)
			}
		// Phase 28: 登录日志添加 nickname 字段,支持显示"nickname（username）"格式
		if err := migrations.Migration161LoginLogAddNickname(d.DB); err != nil {
			applogger.Errorf("sys_logininfor nickname 字段迁移失败: %v", err)
		}
		// Phase 36: AD 域控服务账号池 + 自动故障切换
		// 创建 sys_ad_service_accounts 表 + 从 sys_ad_config 迁移首个管理员账号
		if err := migrations.Migrate162ADServiceAccounts(d.DB); err != nil {
			applogger.Errorf("AD 服务账号池迁移失败: %v", err)
		}
		// Phase 36: AD 服务账号池菜单 + 权限 seed
		if err := migrations.Migrate163ADAccountPoolMenu(d.DB); err != nil {
			applogger.Errorf("AD 服务账号池菜单迁移失败: %v", err)
		}
		// Phase 38: 放宽 sys_ad_config 单管理员列约束 + 幂等校验账号已迁入池（D-02 + D-03）
		if err := migrations.Migrate164Phase38VerifyAdminMigrated(d.DB); err != nil {
			applogger.Errorf("Phase 38 账号池补迁校验失败: %v", err)
		}
		// Phase 39: 工位部门物理位置映射表 (sys_dept_location_alias) + 4 条按钮权限 seed
		if err := migrations.Migrate165SysDeptLocationAlias(d.DB); err != nil {
			applogger.Errorf("Phase 39 工位部门映射迁移失败: %v", err)
		}
		// Phase 40 (TECH-03): 修正 sys_menu.path 重复拼接 (/system/apikeys/system/apikeys)
		if err := migrations.Migrate166ApikeyRoutePathFix(d.DB); err != nil {
			applogger.Errorf("Phase 40 API 密钥菜单 path 修正失败: %v", err)
		}
		// n71: 工位状态联动 — 回填历史 user_id 有值但 status=0 的工位为 status=1 (占用)
		if err := migrations.Migrate167WorkstationBackfillStatus(d.DB); err != nil {
			applogger.Errorf("n71 工位状态回填失败: %v", err)
		}
		// Phase 42 R1: 资产对账观测底座 — 主表 + 物化视图 + partial unique index
		if err := migrations.Migrate168ReconciliationTables(d.DB); err != nil {
			applogger.Errorf("Phase 42 R1 reconciliation tables 迁移失败: %v", err)
		}
		// Phase 42 R1: 字典 + config + workorder category + sys_job + 菜单 seed
		if err := migrations.Migrate169ReconciliationDictsConfigs(d.DB); err != nil {
			applogger.Errorf("Phase 42 R1 reconciliation seed 迁移失败: %v", err)
		}
		// Phase 42 R1 后置修复: 资产列表菜单 path 'assets' -> 'list'。
		// migration_147 种子数据 bug: childPath = "assets" 与父菜单 path='assets' 重复,
		// routeGenerator 拼出 'assets/assets' 无路由 → fallback /dashboard。
		// 这里用 SQL UPDATE 修复已存在记录;新部署 147 同步改 childPath = "list"。
		if err := migrations.Migrate170FixAssetListMenuPath(d.DB); err != nil {
			applogger.Errorf("Phase 42 R1 资产列表菜单 path 修复失败: %v", err)
		}
		// Phase 43 R2: 资产对账 workorder assignee_role + SLA minutes sys_config seed
		// (D-A2-02 + D-A2-03)
		if err := migrations.Migrate171ReconciliationWorkorderAssigneeRole(d.DB); err != nil {
			applogger.Errorf("Phase 43 R2 资产对账 workorder assignee_role 失败: %v", err)
		}
		// Phase 43 R2: 6 个对账 category description 补种(D-A2-04)
		if err := migrations.Migrate172ReconciliationWorkorderTemplatesSeed(d.DB); err != nil {
			applogger.Errorf("Phase 43 R2 资产对账 workorder templates seed 失败: %v", err)
		}
		// Phase 43 R2: reconciliation_normalized MV 扩展 last_resolved_at / last_resolved_by /
		// last_conflict_type 三字段,给 DetectLayer3 提供 7d 静默期查询(D-A3-01)
		if err := migrations.Migrate173ReconciliationSilenceMV(d.DB); err != nil {
			applogger.Errorf("Phase 43 R2 资产对账 silence MV 失败: %v", err)
		}
		// Phase 44 R3: sys_reconciliation_exception GiST inet_ops 索引 + CHECK 约束
		// (D-R3-A1-03 命中测试单点查询 + chk_recon_exc_actions / chk_recon_exc_severity_override)
		if err := migrations.Migrate174ReconciliationExceptionGist(d.DB); err != nil {
			applogger.Errorf("Phase 44 R3 reconciliation GiST/CHECK 失败: %v", err)
		}
		// Phase 45 R5: 物理链路底座 — ops_asset_physical + reconciliation_physical_chain 视图
		// + reconciliation_user_lookup 视图(D-R5-A1-01)
		if err := migrations.Migrate175ReconciliationPhysicalLink(d.DB); err != nil {
			applogger.Errorf("Phase 45 R5 reconciliation 物理链路底座失败: %v", err)
		}
		// Phase 45 R5: reconciliation_normalized MV 重写接双源 declared + 真物理链路(D-R5-A2-01)
		if err := migrations.Migrate176ReconciliationPhysicalMV(d.DB); err != nil {
			applogger.Errorf("Phase 45 R5 reconciliation_normalized MV 重写失败: %v", err)
		}
		// 重新应用 sys_device_port_status 唯一约束(原 migration 010 被 archive,从未被部署)
		// 修复 port 采集 UPSERT 实际 INSERT 重复行 + infoPoint.port_id 漂移问题
		if err := migrations.Migrate177ReapplyPortStatusUnique(d.DB); err != nil {
			applogger.Errorf("Phase 45+ 端口状态唯一约束重应用失败: %v", err)
		}
		// 修复 reconciliation_physical_chain 视图的 JOIN 失配(MAC 格式 + interface_name 格式)
		// 同步重建 reconciliation_normalized 物化视图,让对账健康度不再误判"物理无"
		if err := migrations.Migrate178FixPhysicalChainNormalization(d.DB); err != nil {
			applogger.Errorf("Phase 45+ reconciliation_physical_chain 归一化修复失败: %v", err)
		}
		// Phase 45 R5 修复 (2026-06-29): migration_178 view 注释声称 iface 折叠但实际
		// 只做了 LOWER + strip spaces,导致 reconciliation_physical_chain.physical_user_id
		// 命中率为 0。migration_179 用 PL/pgSQL 函数 normalize_iface() 真正折叠
		// GE/Gi/GigabitEthernet/te/TenGigE/xe/twe/tw/hge/fge 等跨厂商命名,重建 view + MV
		if err := migrations.Migrate179ReconciliationPhysicalIfaceFold(d.DB); err != nil {
			applogger.Errorf("Phase 45+ reconciliation_physical_chain iface 真折叠失败: %v", err)
		}
		// Phase 45 R5 性能修复 (2026-06-30): migration_179 的 PL/pgSQL normalize_iface()
		// 配合视图里的相关子查询产生 O(N×M) 调用,SELECT COUNT(*) 直接触发 statement_timeout。
		// migration_180 把 normalize_iface 预算到 CTE(O(N+M)),消除相关子查询 + 去 device_id::text
		// 强转,COUNT 回到秒级。normalize_iface 函数本身不动(结构修复后已非主因)。
		if err := migrations.Migrate180ReconciliationPhysicalChainPerf(d.DB); err != nil {
			applogger.Errorf("Phase 45+ reconciliation_physical_chain O(N²)→O(N+M) 性能重构失败: %v", err)
		}
		// Phase 45 R5 收尾 (2026-06-30): reconciliation_normalized MV 增加 workstation_id 列。
		// migration_176 创建 MV 时漏 select pc.workstation_id,导致 WorkstationIDForException
		// 触发 SQLSTATE 42703,R4 工位健康度缓存失效链断裂。本迁移 DROP + CREATE + REFRESH 修复。
		if err := migrations.Migrate181ReconciliationNormalizedWorkstationID(d.DB); err != nil {
			applogger.Errorf("Phase 45+ reconciliation_normalized MV workstation_id 列修复失败: %v", err)
		}
		// Phase 45 R5 数据治理 (2026-06-30): 加 sys_device_port_status.device_id FK 约束,
		// 防止未来 device_id 漂移到不存在的设备(orphan)。NOT VALID + 异步 VALIDATE
		// 避免锁表。FK 局限:只能防 orphan,无法防"指向错误但存在的设备",
		// 后者靠 Excel 信息点重新录入 + 每日定时对账任务捕获。
		// (注:2026-06-30 migration_182 自动化修复被删除,改用 Excel 重新录入策略)
		if err := migrations.Migrate183AddPortStatusDeviceFK(d.DB); err != nil {
			applogger.Errorf("Phase 45+ 数据治理 加 port_status.device_id FK 约束失败: %v", err)
		}
		// 创建 MAC 地址过滤规则表(对应 117_create_mac_filter_rules.sql, 含 5 条系统默认规则)
		if err := migrations.Migrate117CreateMacFilterRules(d.DB); err != nil {
			applogger.Errorf("MAC 过滤规则表迁移失败: %v", err)
		}
		// 创建 OUI 厂商表(对应 033_create_mac_oui_vendor_table.up.sql)
		if err := migrations.Migrate033CreateMacOuiVendorTable(d.DB); err != nil {
			applogger.Errorf("OUI 厂商表迁移失败: %v", err)
		}

	// 所有迁移完成后,审计约束命名是否一致,提前暴露潜在 AutoMigrate 冲突
	d.auditConstraintNaming()

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

	// 改为 INFO 级别,因为多数是历史关联表/复合 unique,不一定造成实际冲突
	// 真正冲突会在 AutoMigrate 阶段 FATA,届时根据错误信息加入 cleanupOldConstraints
	applogger.Infof("[约束审计] 发现 %d 个 PG 自动命名 unique 约束(若对应 model 无 uniqueIndex tag 可忽略):", len(rows))
	for _, r := range rows {
		applogger.Infof("  · %s.%s", r.TableName, r.ConstraintName)
	}
	applogger.Infof("[约束审计] 如遇 `DROP CONSTRAINT uni_*` FATA,把 {表名,约束名} 加入 cleanupOldConstraints 即可")
}

// migrateCredentialModel 迁移凭证模型（从 credential_type 到 protocol_type）
// 这个迁移将凭证从 SSH/Telnet/SNMP 分类改为统一的凭证模型
func (d *Database) migrateCredentialModel() error {
	// 只对 PostgreSQL 执行 SQL 迁移
	if d.Type != "postgres" {
		applogger.Infof("非 PostgreSQL 数据库，跳过凭证模型 SQL 迁移")
		return nil
	}

	// 检查迁移是否已经执行过（通过检查 protocol_type 列是否存在）
	var columnExists bool
	err := d.DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'sys_auth_credential' AND column_name = 'protocol_type'
		)
	`).Scan(&columnExists).Error
	if err != nil {
		return fmt.Errorf("检查迁移状态失败: %w", err)
	}

	// 检查旧列是否存在，如果存在则删除（即使主迁移已执行）
	var oldColumnExists bool
	err = d.DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'sys_auth_credential' AND column_name = 'credential_type'
		)
	`).Scan(&oldColumnExists).Error
	if err != nil {
		return fmt.Errorf("检查旧列状态失败: %w", err)
	}

	// 如果旧列存在，需要删除
	if oldColumnExists {
		applogger.Infof("发现旧的 credential_type 列，执行清理...")
		if err := d.dropOldCredentialColumns(); err != nil {
			return fmt.Errorf("删除旧列失败: %w", err)
		}
	}

	// 如果 protocol_type 列已存在，说明迁移已执行过
	if columnExists {
		applogger.Infof("凭证模型迁移已执行，跳过")
		return nil
	}

	applogger.Infof("开始执行凭证模型迁移...")

	// 使用事务
	if err := d.DB.Transaction(func(tx *gorm.DB) error {
		// 步骤1：添加新列
		if err := tx.Exec("ALTER TABLE sys_auth_credential ADD COLUMN IF NOT EXISTS protocol_type VARCHAR(10)").Error; err != nil {
			return fmt.Errorf("添加 protocol_type 列失败: %w", err)
		}

		if err := tx.Exec("ALTER TABLE sys_auth_credential ADD COLUMN IF NOT EXISTS snmp_communities TEXT[]").Error; err != nil {
			return fmt.Errorf("添加 snmp_communities 列失败: %w", err)
		}

		// 步骤2：根据旧的 credential_type 设置 protocol_type
		if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'ssh' WHERE credential_type = 'ssh'").Error; err != nil {
			return fmt.Errorf("迁移 SSH 凭证失败: %w", err)
		}

		if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'telnet' WHERE credential_type = 'telnet'").Error; err != nil {
			return fmt.Errorf("迁移 Telnet 凭证失败: %w", err)
		}

		if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'ssh' WHERE credential_type = 'snmp' AND protocol_type IS NULL").Error; err != nil {
			return fmt.Errorf("迁移 SNMP 凭证失败: %w", err)
		}

		// 步骤3：将旧的 snmp_community 转换为数组
		if err := tx.Exec("UPDATE sys_auth_credential SET snmp_communities = ARRAY[snmp_community] WHERE snmp_community IS NOT NULL AND snmp_community != '' AND snmp_community != ' '").Error; err != nil {
			return fmt.Errorf("转换 SNMP community 失败: %w", err)
		}

		// 步骤4：初始化空数组
		if err := tx.Exec("UPDATE sys_auth_credential SET snmp_communities = ARRAY[]::TEXT[] WHERE snmp_communities IS NULL").Error; err != nil {
			return fmt.Errorf("初始化 SNMP communities 数组失败: %w", err)
		}

		// 步骤5：设置默认 protocol_type
		if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'ssh' WHERE protocol_type IS NULL").Error; err != nil {
			return fmt.Errorf("设置默认 protocol_type 失败: %w", err)
		}

		// 步骤6：删除旧列
		if err := tx.Exec("ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS credential_type").Error; err != nil {
			return fmt.Errorf("删除 credential_type 列失败: %w", err)
		}

		if err := tx.Exec("ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS snmp_community").Error; err != nil {
			return fmt.Errorf("删除 snmp_community 列失败: %w", err)
		}

		// 步骤7：添加约束
		if err := tx.Exec("ALTER TABLE sys_auth_credential ADD CONSTRAINT chk_protocol_type CHECK (protocol_type IN ('ssh', 'telnet'))").Error; err != nil {
			// 约束可能已存在，忽略错误
			applogger.Warnf("添加 protocol_type 约束失败（可能已存在）: %v", err)
		}

		if err := tx.Exec("ALTER TABLE sys_auth_credential ADD CONSTRAINT chk_snmp_version CHECK (snmp_version IN ('v1', 'v2c', 'v3'))").Error; err != nil {
			// 约束可能已存在，忽略错误
			applogger.Warnf("添加 snmp_version 约束失败（可能已存在）: %v", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("凭证模型迁移失败: %w", err)
	}

	applogger.Infof("凭证模型迁移完成")

	return nil
}

// dropOldCredentialColumns 删除旧的凭证列（credential_type 和 snmp_community）
func (d *Database) dropOldCredentialColumns() error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		// 删除旧的 credential_type 列
		if err := tx.Exec("ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS credential_type").Error; err != nil {
			return fmt.Errorf("删除 credential_type 列失败: %w", err)
		}

		// 删除旧的 snmp_community 列
		if err := tx.Exec("ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS snmp_community").Error; err != nil {
			return fmt.Errorf("删除 snmp_community 列失败: %w", err)
		}

		return nil
	})
}

// InitData 初始化基础数据
func (d *Database) InitData() error {
	// 两种数据库都使用相同的初始化逻辑
	return initData(d.DB)
}

// initData 初始化基础数据
func initData(db *gorm.DB) error {
	// 创建默认部门
	if err := createDefaultDept(db); err != nil {
		return fmt.Errorf("创建默认部门失败: %w", err)
	}

	// 创建默认用户
	if err := createDefaultUser(db); err != nil {
		return fmt.Errorf("创建默认用户失败: %w", err)
	}

	// 创建默认角色
	if err := createDefaultRole(db); err != nil {
		return fmt.Errorf("创建默认角色失败: %w", err)
	}

	// 创建用户角色关联
	if err := createUserRoleRelations(db); err != nil {
		return fmt.Errorf("创建用户角色关联失败: %w", err)
	}

	// 创建网络设备系统参数
	if err := createNetworkDeviceSystemParams(db); err != nil {
		return fmt.Errorf("创建网络设备系统参数失败: %w", err)
	}

	// 创建网络设备定时任务
	if err := createNetworkDeviceScheduledJobs(db); err != nil {
		return fmt.Errorf("创建网络设备定时任务失败: %w", err)
	}

	// 创建验证码背景图系统参数
	if err := createCaptchaBackgroundSystemParams(db); err != nil {
		return fmt.Errorf("创建验证码背景图系统参数失败: %w", err)
	}

	// 创建运维管理菜单
	if err := createOperationsManagementMenus(db); err != nil {
		return fmt.Errorf("创建运维管理菜单失败: %w", err)
	}

	// 创建请求加密开关配置参数
	if err := createRequestEncryptionToggleConfig(db); err != nil {
		return fmt.Errorf("创建请求加密开关配置失败: %w", err)
	}
	// 创建AD认证配置参数
	if err := createADAuthConfig(db); err != nil {
		return fmt.Errorf("创建AD认证配置参数失败: %w", err)
	}

	applogger.Infof("基础数据初始化完成")
	return nil
}

// createDefaultDept 创建默认部门
func createDefaultDept(db *gorm.DB) error {
	// 检查是否已有部门数据
	var count int64
	db.Model(&models.Department{}).Count(&count)
	if count > 0 {
		applogger.Infof("部门数据已存在，跳过初始化")
		return nil
	}

	// 创建顶级部门
	topDept := models.Department{
		DeptName: "若依科技有限公司",
		OrderNum: 1,
		Leader:   func() *string { s := "若依"; return &s }(),
		Phone:    func() *string { s := "15888888888"; return &s }(),
		Email:    func() *string { s := "ruoyi@qq.com"; return &s }(),
		Status:   models.DeptStatusNormal,
		Remark:   "",
	}

	if err := db.Create(&topDept).Error; err != nil {
		return fmt.Errorf("创建顶级部门失败: %w", err)
	}

	// 创建子部门
	subDepts := []models.Department{
		{
			DeptName:  "深圳总公司",
			ParentID:  &topDept.ID,
			Ancestors: topDept.ID,
			OrderNum:  1,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
		{
			DeptName:  "长沙分公司",
			ParentID:  &topDept.ID,
			Ancestors: topDept.ID,
			OrderNum:  2,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
	}

	var shenzhenDeptID string
	for _, dept := range subDepts {
		if err := db.Create(&dept).Error; err != nil {
			return fmt.Errorf("创建部门 %s 失败: %w", dept.DeptName, err)
		}
		if dept.DeptName == "深圳总公司" {
			shenzhenDeptID = dept.ID
		}
		applogger.Infof("创建部门 %s 成功", dept.DeptName)
	}

	// 创建深圳总公司的子部门
	shenzhenSubDepts := []models.Department{
		{
			DeptName:  "研发部门",
			ParentID:  &shenzhenDeptID,
			Ancestors: topDept.ID + "," + shenzhenDeptID,
			OrderNum:  1,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
		{
			DeptName:  "市场部门",
			ParentID:  &shenzhenDeptID,
			Ancestors: topDept.ID + "," + shenzhenDeptID,
			OrderNum:  2,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
		{
			DeptName:  "测试部门",
			ParentID:  &shenzhenDeptID,
			Ancestors: topDept.ID + "," + shenzhenDeptID,
			OrderNum:  3,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
	}

	for _, dept := range shenzhenSubDepts {
		if err := db.Create(&dept).Error; err != nil {
			return fmt.Errorf("创建部门 %s 失败: %w", dept.DeptName, err)
		}
		applogger.Infof("创建部门 %s 成功", dept.DeptName)
	}

	applogger.Infof("默认部门创建完成")
	return nil
}

// createDefaultUser 创建默认管理员用户
func createDefaultUser(db *gorm.DB) error {
	var count int64

	// 检查是否已存在管理员用户
	db.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return nil // 已存在，跳过
	}

	// 使用新的SM3密码管理器
	pwdManager := security.NewPasswordManager(nil)

	// 生成密码哈希
	passwordHash, err := pwdManager.HashPassword("admin123")
	if err != nil {
		return err
	}

	// 创建默认管理员用户
	user := models.User{
		Username: "admin",
		Password: passwordHash,
		Salt:     "default",
		Nickname: func() *string { s := "超级管理员"; return &s }(),
		Email:    func() *string { s := "admin@ruoyi.com"; return &s }(),
		Gender:   models.GenderMale,
		Status:   models.UserStatusEnabled,
		DeptName: func() *string { s := "总公司"; return &s }(),
		Roles:    []string{"admin"},
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("创建默认用户失败: %w", err)
	}

	applogger.Infof("创建默认管理员用户成功")
	return nil
}

// createDefaultRole 创建默认角色
func createDefaultRole(db *gorm.DB) error {
	roles := []models.Role{
		{
			RoleName:          "超级管理员",
			RoleKey:           "admin",
			RoleSort:          1,
			DataScope:         models.DataScopeAll,
			MenuCheckStrictly: true,
			DeptCheckStrictly: true,
			Status:            models.RoleStatusEnabled,
			Remark:            "超级管理员",
		},
		{
			RoleName:          "普通用户",
			RoleKey:           "user",
			RoleSort:          2,
			DataScope:         models.DataScopeSelf,
			MenuCheckStrictly: true,
			DeptCheckStrictly: true,
			Status:            models.RoleStatusEnabled,
			Remark:            "普通用户",
		},
	}

	for _, role := range roles {
		var count int64
		// 检查角色是否已存在（通过role_key）
		db.Model(&models.Role{}).Where("role_key = ?", role.RoleKey).Count(&count)
		if count > 0 {
			applogger.Infof("角色 %s (role_key: %s) 已存在，跳过创建", role.RoleName, role.RoleKey)
			continue
		}

		if err := db.Create(&role).Error; err != nil {
			return fmt.Errorf("创建角色 %s 失败: %w", role.RoleName, err)
		}
		applogger.Infof("创建角色 %s 成功", role.RoleName)
	}

	applogger.Infof("默认角色检查/创建完成")
	return nil
}

// createUserRoleRelations 创建用户角色关联
func createUserRoleRelations(db *gorm.DB) error {
	// 获取默认用户
	var adminUser models.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		applogger.Warnf("未找到管理员用户: %v", err)
		return nil // 如果用户不存在，跳过
	}

	// 获取管理员角色
	var adminRole models.Role
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		applogger.Warnf("未找到管理员角色: %v", err)
		return nil // 如果角色不存在，跳过
	}

	// 检查关联是否已存在
	var count int64
	db.Table("sys_user_role").Where("user_id = ? AND role_id = ?", adminUser.ID, adminRole.ID).Count(&count)
	if count > 0 {
		applogger.Infof("用户角色关联已存在，跳过创建")
		return nil
	}

	// 创建用户角色关联
	if err := db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)",
		adminUser.ID, adminRole.ID).Error; err != nil {
		return fmt.Errorf("创建用户角色关联失败: %w", err)
	}

	applogger.Infof("创建用户角色关联成功")
	return nil
}

// createDatabaseIfNotExists 如果数据库不存在则创建
func createDatabaseIfNotExists(adminDSN, dbName string) error {
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
		createQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
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

// createNetworkDeviceSystemParams 创建网络设备系统参数
func createNetworkDeviceSystemParams(db *gorm.DB) error {
	params := []models.Config{
		{
			ConfigName:  "配置备份文件大小阈值",
			ConfigKey:   "network.config.backup.threshold",
			ConfigValue: "100",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "配置备份文件大小阈值（单位：KB），小于此值的配置存储在数据库，大于此值存储在文件系统",
		},
		{
			ConfigName:  "设备连接超时时间",
			ConfigKey:   "network.device.connect.timeout",
			ConfigValue: "30",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "设备连接超时时间（单位：秒）",
		},
		{
			ConfigName:  "命令执行超时时间",
			ConfigKey:   "network.command.execute.timeout",
			ConfigValue: "300",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "命令执行超时时间（单位：秒）",
		},
		{
			ConfigName:  "批量命令并发数",
			ConfigKey:   "network.command.batch.concurrency",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "批量命令执行时的最大并发设备数量",
		},
		// 新增：网络设备并发配置参数
		{
			ConfigName:  "网络设备监控并发数",
			ConfigKey:   "network.device.monitor.concurrent",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "设备状态检查和信息更新的最大并发数，默认10",
		},
		{
			ConfigName:  "端口采集并发数",
			ConfigKey:   "network.port.collection.concurrent",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "端口状态采集的最大并发数，默认10",
		},
		{
			ConfigName:  "MAC地址采集并发数",
			ConfigKey:   "network.mac.collection.concurrent",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "MAC地址表采集的最大并发数，默认10",
		},
		{
			ConfigName:  "配置备份并发数",
			ConfigKey:   "network.config.backup.concurrent",
			ConfigValue: "5",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "配置备份的最大并发数，默认5（配置备份较耗时，建议较低并发）",
		},
		{
			ConfigName:  "设备操作超时时间",
			ConfigKey:   "network.device.timeout",
			ConfigValue: "30",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "单个设备连接和操作的超时时间（秒），默认30秒",
		},
	}

	for _, param := range params {
		var count int64
		db.Model(&models.Config{}).Where("config_key = ?", param.ConfigKey).Count(&count)
		if count > 0 {
			applogger.Infof("系统参数 %s 已存在，跳过创建", param.ConfigName)
			continue
		}

		if err := db.Create(&param).Error; err != nil {
			return fmt.Errorf("创建系统参数 %s 失败: %w", param.ConfigName, err)
		}
		applogger.Infof("创建系统参数 %s 成功", param.ConfigName)
	}

	applogger.Infof("网络设备系统参数创建完成")
	return nil
}

// createNetworkDeviceScheduledJobs 创建网络设备定时任务
func createNetworkDeviceScheduledJobs(db *gorm.DB) error {
	remark := func(s string) *string { return &s }

	jobs := []models.Job{
		{
			JobName:        "设备状态检查",
			JobGroup:       "NETWORK",
			InvokeTarget:   "device_status_check",
			CronExpression: "0 */5 * * * ?",       // 每5分钟执行一次
			Status:         models.JobStatusPause, // 默认暂停，由用户手动启动
			Concurrent:     false,                 // 禁止并发
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("通过SNMP定时检查所有网络设备的在线/离线状态"),
		},
		{
			JobName:        "设备信息更新",
			JobGroup:       "NETWORK",
			InvokeTarget:   "device_info_update",
			CronExpression: "0 0 * * * ?", // 每小时执行一次
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("通过SSH采集网络设备的详细信息（型号、版本、序列号等）"),
		},
		{
			JobName:        "端口状态采集",
			JobGroup:       "NETWORK",
			InvokeTarget:   "port_collection",
			CronExpression: "0 0 * * * ?", // 每小时执行一次
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("采集所有在线网络设备的端口状态信息（启用/禁用、802.1X、端口安全等）"),
		},
		{
			JobName:        "MAC地址采集",
			JobGroup:       "NETWORK",
			InvokeTarget:   "mac_collection",
			CronExpression: "0 0 * * * ?", // 每小时执行一次
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("采集网络设备的MAC地址表信息"),
		},
		{
			JobName:        "配置备份",
			JobGroup:       "NETWORK",
			InvokeTarget:   "config_backup",
			CronExpression: "0 0 2 * * ?", // 每天凌晨2点执行
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("自动备份网络设备配置文件"),
		},
	}

	for _, job := range jobs {
		var count int64
		db.Model(&models.Job{}).Where("invoke_target = ?", job.InvokeTarget).Count(&count)
		if count > 0 {
			applogger.Infof("定时任务 %s 已存在，跳过创建", job.JobName)
			continue
		}

		if err := db.Create(&job).Error; err != nil {
			return fmt.Errorf("创建定时任务 %s 失败: %w", job.JobName, err)
		}
		applogger.Infof("创建定时任务 %s 成功", job.JobName)
	}

	applogger.Infof("网络设备定时任务创建完成")
	return nil
}

// createCaptchaBackgroundSystemParams 创建验证码背景图系统参数
func createCaptchaBackgroundSystemParams(db *gorm.DB) error {
	params := []models.Config{
		{
			ConfigName:  "验证码背景图模式",
			ConfigKey:   "sys.account.captchaBackgroundMode",
			ConfigValue: "mixed",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "背景图模式: auto=自动生成 custom=仅自定义图片 mixed=混合模式",
		},
		{
			ConfigName:  "验证码默认拼图形状",
			ConfigKey:   "sys.account.captchaPieceShape",
			ConfigValue: "circle",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "默认拼图形状: circle=圆形 square=方形 star=星形 heart=心形",
		},
		{
			ConfigName:  "验证码默认难度",
			ConfigKey:   "sys.account.captchaDifficulty",
			ConfigValue: "1",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "难度级别: 1=简单 2=中等 3=困难",
		},
		{
			ConfigName:  "验证码缓存池大小",
			ConfigKey:   "sys.account.captchaCachePoolSize",
			ConfigValue: "50",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "每种形状和难度预生成的验证码数量",
		},
		{
			ConfigName:  "验证码图片存储路径",
			ConfigKey:   "sys.account.captchaStoragePath",
			ConfigValue: "./uploads/captcha/backgrounds",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "背景图存储路径",
		},
		{
			ConfigName:  "验证码图片最大大小",
			ConfigKey:   "sys.account.captchaMaxFileSize",
			ConfigValue: "2097152",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "单张图片最大大小(字节)，默认2MB",
		},
		{
			ConfigName:  "验证码允许的图片格式",
			ConfigKey:   "sys.account.captchaAllowedFormats",
			ConfigValue: "jpg,jpeg,png",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "允许的图片格式，逗号分隔",
		},
	}

	for _, param := range params {
		var count int64
		db.Model(&models.Config{}).Where("config_key = ?", param.ConfigKey).Count(&count)
		if count > 0 {
			applogger.Infof("系统参数 %s 已存在，跳过创建", param.ConfigName)
			continue
		}

		if err := db.Create(&param).Error; err != nil {
			return fmt.Errorf("创建系统参数 %s 失败: %w", param.ConfigName, err)
		}
		applogger.Infof("创建系统参数 %s 成功", param.ConfigName)
	}

	applogger.Infof("验证码背景图系统参数创建完成")
	return nil
}

// createCaptchaBackgroundMenus 创建验证码背景图管理菜单
// TODO: 此函数未使用，如需要启用验证码背景图功能，请取消注释并调用此函数
/*
func createCaptchaBackgroundMenus(db *gorm.DB) error {
	// 先查找系统管理菜单的ID
	var systemMenu models.Menu
	if err := db.Where("menu_name = ? AND menu_type = ?", "系统管理", "M").First(&systemMenu).Error; err != nil {
		log.Printf("未找到系统管理菜单，跳过创建验证码背景图子菜单: %v", err)
		return nil
	}

	// 检查验证码背景图菜单是否已存在
	var existingCount int64
	db.Model(&models.Menu{}).Where("menu_name = ?", "验证码背景图").Count(&existingCount)
	if existingCount > 0 {
		log.Println("验证码背景图菜单已存在，跳过创建")
		return nil
	}

	menus := []models.Menu{
		{
			MenuName:  "验证码背景图",
			ParentID:  &systemMenu.ID,
			OrderNum:  10,
			Path:      func() *string { s := "captcha-background"; return &s }(),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeDir,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     NULL_STRING_PTR(""),
			Icon:      func() *string { s := "picture"; return &s }(),
			Remark:    "验证码背景图管理",
		},
		{
			MenuName:  "背景图查询",
			ParentID:  nil, // 稍后更新
			OrderNum:  1,
			Path:      func() *string { s := "background"; return &s }(),
			Component: func() *string { s := "system/captcha-background/index"; return &s }(),
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:list"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图查询菜单",
		},
		{
			MenuName:  "背景图新增",
			ParentID:  nil, // 稍后更新
			OrderNum:  2,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:add"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图新增按钮",
		},
		{
			MenuName:  "背景图修改",
			ParentID:  nil, // 稍后更新
			OrderNum:  3,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:edit"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图修改按钮",
		},
		{
			MenuName:  "背景图删除",
			ParentID:  nil, // 稍后更新
			OrderNum:  4,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:remove"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图删除按钮",
		},
	}

	// 首先创建目录菜单
	if err := db.Create(&menus[0]).Error; err != nil {
		return fmt.Errorf("创建验证码背景图目录菜单失败: %w", err)
	}
	log.Printf("创建菜单 %s 成功", menus[0].MenuName)

	// 更新子菜单的ParentID为目录菜单的ID
	catalogID := menus[0].ID
	for i := 1; i < len(menus); i++ {
		menus[i].ParentID = &catalogID
		if err := db.Create(&menus[i]).Error; err != nil {
			return fmt.Errorf("创建菜单 %s 失败: %w", menus[i].MenuName, err)
		}
		log.Printf("创建菜单 %s 成功", menus[i].MenuName)
	}

	log.Println("验证码背景图管理菜单创建完成")
	return nil
}
*/

// NULL_STRING_PTR 返回字符串指针的辅助函数
func NULL_STRING_PTR(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// createDutyManagementMenus 创建值班管理菜单
//
// 已废弃：请使用 migrations/018_unify_duty_menus.sql 迁移文件
//
// 此函数保留仅为向后兼容，不会在菜单已存在时重复创建
// 所有值班管理菜单（值班池管理、排班管理、节假日管理、值班配置、我的值班）
// 现在通过 SQL 迁移文件统一创建
//
// createOperationsManagementMenus 创建运维管理菜单
func createOperationsManagementMenus(db *gorm.DB) error {
	// 检查运维管理菜单是否已存在
	var opsMenu models.Menu
	err := db.Where("menu_name = ? AND menu_type = ?", "运维管理", "M").First(&opsMenu).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询运维管理菜单失败: %w", err)
	}

	// 如果运维管理菜单不存在，创建它
	if err == gorm.ErrRecordNotFound {
		opsMenu = models.Menu{
			MenuName:  "运维管理",
			OrderNum:  4,
			Path:      func() *string { s := "ops"; return &s }(),
			Component: NULL_STRING_PTR("Layout"),
			MenuType:  models.MenuTypeDir,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Icon:      func() *string { s := "Control"; return &s }(),
			Remark:    "运维管理目录",
		}
		if err := db.Create(&opsMenu).Error; err != nil {
			return fmt.Errorf("创建运维管理菜单失败: %w", err)
		}
		applogger.Infof("创建菜单 %s 成功", opsMenu.MenuName)
	} else {
		applogger.Infof("运维管理菜单已存在，跳过创建")
	}

	// 定义需要创建的页面菜单（检查是否存在，不存在才创建）
	pageMenus := []struct {
		name      string
		path      string
		component string
		perms     string
		remark    string
		orderNum  int
		icon      string
	}{
		{"楼宇管理", "buildings", "operations/buildings/index", "ops:building:list", "楼宇管理菜单", 1, "BuildOutlined"},
		{"楼层管理", "floors", "operations/floors/index", "ops:floor:list", "楼层管理菜单", 2, "ApartmentOutlined"},
		{"工位管理", "workstations", "operations/workstations/index", "ops:workstation:list", "工位管理菜单", 3, "DesktopOutlined"},
		{"信息点管理", "info-points", "operations/info-points/index", "ops:infopoint:list", "信息点管理菜单", 4, "DotChartOutlined"},
		{"机房管理", "server-rooms", "operations/server-rooms/index", "ops:serverroom:list", "机房管理菜单", 5, "CloudServerOutlined"},
		{"专线管理", "dedicated-lines", "operations/dedicated-lines/index", "ops:dedicatedline:list", "专线管理菜单", 6, "LineChartOutlined"},
		{"机房设备管理", "room-devices", "operations/room-devices/index", "ops:roomdevice:list", "机房设备管理菜单", 7, "AppstoreOutlined"},
	}

	// 存储页面菜单的ID，用于后续创建按钮权限
	menuIDs := make(map[string]string)

	for _, pm := range pageMenus {
		// 检查菜单是否已存在
		var existingMenu models.Menu
		err := db.Where("menu_name = ? AND parent_id = ?", pm.name, opsMenu.ID).First(&existingMenu).Error

		if err == nil {
			// 菜单已存在，使用现有菜单ID，跳过创建
			menuIDs[pm.name] = existingMenu.ID
			applogger.Infof("菜单 %s 已存在，跳过创建", pm.name)
			continue
		}

		// 菜单不存在，创建新菜单
		menu := models.Menu{
			MenuName:  pm.name,
			ParentID:  &opsMenu.ID,
			OrderNum:  pm.orderNum,
			Path:      func() *string { s := pm.path; return &s }(),
			Component: func() *string { s := pm.component; return &s }(),
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := pm.perms; return &s }(),
			Icon:      func() *string { s := pm.icon; return &s }(),
			Remark:    pm.remark,
		}
		if err := db.Create(&menu).Error; err != nil {
			return fmt.Errorf("创建菜单 %s 失败: %w", pm.name, err)
		}
		menuIDs[pm.name] = menu.ID
		applogger.Infof("创建菜单 %s 成功", pm.name)
	}

	// 定义需要创建的按钮权限菜单
	buttonMenus := []struct {
		parent   string
		name     string
		perms    string
		remark   string
		orderNum int
	}{
		// 楼宇管理的按钮
		{"楼宇管理", "楼宇查询", "ops:building:query", "楼宇查询", 1},
		{"楼宇管理", "楼宇新增", "ops:building:add", "楼宇新增", 2},
		{"楼宇管理", "楼宇修改", "ops:building:edit", "楼宇修改", 3},
		{"楼宇管理", "楼宇删除", "ops:building:delete", "楼宇删除", 4},
		// 楼层管理的按钮
		{"楼层管理", "楼层查询", "ops:floor:query", "楼层查询", 1},
		{"楼层管理", "楼层新增", "ops:floor:add", "楼层新增", 2},
		{"楼层管理", "楼层修改", "ops:floor:edit", "楼层修改", 3},
		{"楼层管理", "楼层删除", "ops:floor:delete", "楼层删除", 4},
		// 工位管理的按钮
		{"工位管理", "工位查询", "ops:workstation:query", "工位查询", 1},
		{"工位管理", "工位新增", "ops:workstation:add", "工位新增", 2},
		{"工位管理", "工位修改", "ops:workstation:edit", "工位修改", 3},
		{"工位管理", "工位删除", "ops:workstation:delete", "工位删除", 4},
		// 信息点管理的按钮
		{"信息点管理", "信息点查询", "ops:infopoint:query", "信息点查询", 1},
		{"信息点管理", "信息点新增", "ops:infopoint:add", "信息点新增", 2},
		{"信息点管理", "信息点修改", "ops:infopoint:edit", "信息点修改", 3},
		{"信息点管理", "信息点删除", "ops:infopoint:delete", "信息点删除", 4},
		// 机房管理的按钮
		{"机房管理", "机房查询", "ops:serverroom:query", "机房查询", 1},
		{"机房管理", "机房新增", "ops:serverroom:add", "机房新增", 2},
		{"机房管理", "机房修改", "ops:serverroom:edit", "机房修改", 3},
		{"机房管理", "机房删除", "ops:serverroom:delete", "机房删除", 4},
		// 专线管理的按钮
		{"专线管理", "专线查询", "ops:dedicatedline:query", "专线查询", 1},
		{"专线管理", "专线新增", "ops:dedicatedline:add", "专线新增", 2},
		{"专线管理", "专线修改", "ops:dedicatedline:edit", "专线修改", 3},
		{"专线管理", "专线删除", "ops:dedicatedline:delete", "专线删除", 4},
		// 机房设备管理的按钮
		{"机房设备管理", "设备查询", "ops:roomdevice:query", "设备查询", 1},
		{"机房设备管理", "设备新增", "ops:roomdevice:add", "设备新增", 2},
		{"机房设备管理", "设备修改", "ops:roomdevice:edit", "设备修改", 3},
		{"机房设备管理", "设备删除", "ops:roomdevice:delete", "设备删除", 4},
	}

	for _, bm := range buttonMenus {
		// 检查按钮菜单是否已存在
		var existingButton models.Menu
		parentMenuID := menuIDs[bm.parent]
		err := db.Where("menu_name = ? AND parent_id = ?", bm.name, parentMenuID).First(&existingButton).Error

		if err == nil {
			// 按钮已存在，跳过创建
			applogger.Infof("按钮菜单 %s 已存在，跳过创建", bm.name)
			continue
		}

		// 按钮不存在，创建新按钮
		menu := models.Menu{
			MenuName:  bm.name,
			ParentID:  &parentMenuID,
			OrderNum:  bm.orderNum,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := bm.perms; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    bm.remark,
		}
		if err := db.Create(&menu).Error; err != nil {
			return fmt.Errorf("创建按钮菜单 %s 失败: %w", bm.name, err)
		}
		applogger.Infof("创建按钮菜单 %s 成功", bm.name)
	}

	applogger.Infof("运维管理菜单创建完成")
	return nil
}

// createRequestEncryptionToggleConfig 创建请求加密开关配置参数
func createRequestEncryptionToggleConfig(db *gorm.DB) error {
	// 检查配置是否已存在
	var count int64
	err := db.Table("sys_config").
		Where("config_key = ?", "sys.request.encryption.enabled").
		Count(&count).Error

	if err != nil {
		applogger.Warnf("查询请求加密开关配置失败: %v", err)
		return err
	}

	if count > 0 {
		applogger.Infof("请求加密开关配置已存在，跳过初始化")
		return nil
	}

	// 插入默认配置：true (启用)
	config := models.Config{
		ConfigName:  "请求加密开关",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "true",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    models.ConfigIsSystemYes,
		Remark:      "控制请求体加密功能的启停（true=启用，false=停用），修改后立即生效",
	}

	applogger.Infof("尝试插入请求加密开关配置...")
	if err := db.Create(&config).Error; err != nil {
		applogger.Warnf("创建请求加密开关配置失败: %v", err)
		return err
	}

	applogger.Infof("请求加密开关配置已创建（默认启用）")
	return nil
}

// createADAuthConfig 创建AD认证配置参数
func createADAuthConfig(db *gorm.DB) error {
	authConfigs := []models.Config{
		{
			ConfigName:  "AD认证启用",
			ConfigKey:   "sys.auth.ad.enabled",
			ConfigValue: "false",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "是否启用AD域控认证（true/false）",
		},
		{
			ConfigName:  "默认认证模式",
			ConfigKey:   "sys.auth.default.mode",
			ConfigValue: "local",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "默认认证模式：local=本地, ad=AD, hybrid=混合",
		},
		{
			ConfigName:  "AD配置ID",
			ConfigKey:   "sys.auth.ad.config_id",
			ConfigValue: "",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      "AD域控配置ID（为空则使用第一个启用的配置）",
		},
		{
			ConfigName:  "AD用户默认角色",
			ConfigKey:   "sys.auth.ad.default_role_id",
			ConfigValue: "",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      "AD用户首次登录时分配的默认角色ID",
		},
		{
			ConfigName:  "AD用户默认部门",
			ConfigKey:   "sys.auth.ad.default_dept_id",
			ConfigValue: "",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      "AD用户首次登录时分配的默认部门ID",
		},
	}

	for _, cfg := range authConfigs {
		var count int64
		err := db.Table("sys_config").
			Where("config_key = ?", cfg.ConfigKey).
			Count(&count).Error

		if err != nil {
			applogger.Warnf("查询AD认证配置 %s 失败: %v", cfg.ConfigKey, err)
			return err
		}

		if count > 0 {
			applogger.Infof("AD认证配置 %s 已存在，跳过", cfg.ConfigKey)
			continue
		}

		if err := db.Create(&cfg).Error; err != nil {
			applogger.Warnf("创建AD认证配置 %s 失败: %v", cfg.ConfigKey, err)
			return err
		}

		applogger.Infof("AD认证配置 %s 已创建", cfg.ConfigKey)
	}

	return nil
}
