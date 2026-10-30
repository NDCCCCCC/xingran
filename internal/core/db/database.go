package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/lib/pq"
	"github.com/tjfoc/gmsm/sm3"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models/system"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database 数据库管理器
type Database struct {
	DB   *gorm.DB
	Type string
	// SettingsMenuComponentChanged 表示 Migrate209 (系统设置菜单 component 路径
	// 修正) 本次启动实际改写了数据。core.go 在缓存服务就绪后按此标志失效菜单
	// 缓存 —— 迁移执行期 c.Cache 尚未创建, 迁移函数内拿不到 cache 实例
	// (Phase 70 PATTERNS 事实 1 的时序约束)。
	SettingsMenuComponentChanged bool
	// migrationLockConn 持有启动期 PG advisory lock 的专用 *sql.Conn。
	// 仅 AutoMigrate 期间非空;releaseMigrationAdvisoryLock 后重置为 nil。
	// 不导出,避免外部代码误关连接导致 pg_advisory_unlock noop。
	migrationLockConn *sql.Conn
	// keepaliveStop/keepaliveDone 连接保活 goroutine 的停止信号与完成信号。
	// nil 表示保活未启动(如测试手工构造的 Database) — Close 必须 nil-guard。
	keepaliveStop chan struct{}
	keepaliveDone chan struct{}
}

// NewDatabase 创建数据库连接
//
// 2026-08-15:项目曾硬性切到 PostgreSQL(Supabase),删除旧 SQLite 回退路径。
// 原因:`gorm.io/driver/sqlite` 传递性引入 `github.com/mattn/go-sqlite3`(CGO),
// Windows 上无 gcc 时 `go run` 直接失败。
//
// 2026-08-17:以纯 Go 驱动 `github.com/glebarez/sqlite`(底层 modernc.org/sqlite,
// 无需 CGO)恢复 sqlite 分支 — cfg.Type=="sqlite" 时连接本地文件库,用于 dev 环境
// 摆脱 Supabase 跨国链路延迟。区别于已删除的旧 CGO 路径,CGO 驱动禁令由
// TestNewDatabaseRequiresPostgresConfig 源码断言守护。
//
// 分支语义:
//   - cfg.Type == "sqlite" → createSQLiteConnection(本地文件,不启动 pool keepalive)
//   - 其他(含空字符串)→ 现有 PG 路径一字不动,Host==\"\" 或 Port<=0 时 fail-fast,
//     运维可第一时间发现配置缺失。
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	if cfg == nil {
		return nil, fmt.Errorf("数据库配置缺失:database 配置不能为空")
	}

	// SQLite 分支(dev 本地文件库,纯 Go 驱动,不访问远端 PG/Supabase)
	if cfg.Type == "sqlite" {
		db, err := createSQLiteConnection(cfg)
		if err != nil {
			return nil, fmt.Errorf("连接SQLite失败: %w", err)
		}
		return &Database{DB: db, Type: "sqlite"}, nil
	}

	if cfg.Host == "" || cfg.Port <= 0 {
		return nil, fmt.Errorf("数据库配置缺失:database.host 与 database.port 必须显式配置(项目不再提供 SQLite 回退)")
	}

	dbType := "postgres"
	db, err := createPostgresConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	d := &Database{
		DB:   db,
		Type: dbType,
	}

	// H6 缓解 (login-menu-timeout-20260817): Supabase pooler 上新建连接的
	// TLS+auth 握手实测 ~4.7s;空闲连接被服务端回收后,下一个查询支付全价握手,
	// 在低流量 dev 环境表现为间歇性慢查询。后台保活保持少量连接热备。
	warmConns := cfg.MaxIdleConns
	if warmConns > poolKeepaliveMaxConns {
		warmConns = poolKeepaliveMaxConns
	}
	if warmConns > 0 {
		d.startPoolKeepalive(warmConns)
	}

	return d, nil
}

// poolKeepaliveInterval 连接保活间隔。远短于 Supabase 空闲连接回收窗口,
// 每个周期并发 ping N 次,保持 N 个空闲连接热备(消除 ~4.7s 冷握手)。
const poolKeepaliveInterval = 30 * time.Second

// poolKeepaliveMaxConns 保活连接数上限 — 保活是兜底机制,不应占用大量池配额。
const poolKeepaliveMaxConns = 4

// startPoolKeepalive 启动后台连接保活 goroutine(幂等)。
func (d *Database) startPoolKeepalive(warmConns int) {
	if d.keepaliveStop != nil {
		return // 已启动
	}
	d.keepaliveStop = make(chan struct{})
	d.keepaliveDone = make(chan struct{})
	go d.poolKeepaliveLoop(warmConns)
	applogger.Infof("[pool-keepalive] 已启动, 间隔=%v, 保活连接数=%d", poolKeepaliveInterval, warmConns)
}

func (d *Database) poolKeepaliveLoop(warmConns int) {
	defer close(d.keepaliveDone)
	ticker := time.NewTicker(poolKeepaliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-d.keepaliveStop:
			return
		case <-ticker.C:
			d.pingPoolOnce(warmConns)
		}
	}
}

// pingPoolOnce 并发 ping warmConns 次。database/sql 的 Ping 占用并归还一个连接,
// 并发 N 次即保持 N 个不同连接热备;串行 ping 会复用同一连接,达不到目的。
func (d *Database) pingPoolOnce(warmConns int) {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < warmConns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := sqlDB.PingContext(ctx); err != nil {
				applogger.Warnf("[pool-keepalive] ping 失败: %v", err)
			}
		}()
	}
	wg.Wait()
}

// createFilteredLogger 创建使用 FilterLogger 的 GORM 配置
func createFilteredLogger() *FilterLogger {
	return NewFilterLogger(DefaultLogFilterConfig)
}

// defaultSQLitePath 是 cfg.Path 为空时的兜底 SQLite 文件路径
// (与 config 默认值 database.path 对齐)。
const defaultSQLitePath = "data/xingran.db"

// createSQLiteConnection 创建 SQLite 连接(纯 Go 驱动 glebarez/sqlite,底层
// modernc.org/sqlite,无 CGO 依赖 — 严禁替换为 gorm.io/driver/sqlite)。
//
// 2026-08-17 恢复:dev 环境本地文件库,摆脱 Supabase 跨国链路延迟。
// DSN 走 modernc `_pragma` 参数:busy_timeout 10s + WAL 日志模式 + 外键开启。
//
// dev-only 已知限制(不阻塞,见 plan 260817-hfl):
//   - reconciliation 物化视图/普通 VIEW(175/176)、菜单 seed(202)、sys_config
//     连接池 seed(203)等 PG-only 迁移块不执行(AutoMigrate 的 d.Type=="postgres" guard)
//   - 个别 handler 原生 SQL 若含 PG 方言(::uuid cast / ILIKE / pg_catalog)在 SQLite
//     下会运行期报错 — 按需后续修
func createSQLiteConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	path := cfg.Path
	if path == "" {
		path = defaultSQLitePath
	}

	// 确保数据文件目录存在(相对路径 "." 时跳过)
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建SQLite数据目录失败: %w", err)
		}
	}

	dsn := path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"

	gormConfig := &gorm.Config{
		Logger:  createFilteredLogger(),
		NowFunc: func() time.Time { return time.Now().UTC() },
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              false, // 与 PG 路径对齐(simple protocol)
	}

	db, err := gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接SQLite失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// SQLite 单写者:WAL + busy_timeout 下小池即可,25 连接无意义。
	sqlDB.SetMaxOpenConns(4)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	applogger.Infof("SQLite连接成功: %s", path)
	return db, nil
}

// sanitizeSQLiteModelDefaults 净化 schema 缓存中 PG-only 的 DDL 片段,使 AutoMigrate /
// CreateTable 能在 SQLite 下建表。仅 sqlite 分支调用;PG 路径零改动。
//
// 背景 (2026-08-17, quick-260817-hfl): 多个 model tag 携带 PG 专属片段 —
//   - default:gen_random_uuid() (Asset / APIKeyUsageLog / VDISyncLog / OUGroupMapping 等):
//     SQLite DDL 只允许常量默认值,函数调用触发 "near \"(\": syntax error"
//   - type:text[] (AuthCredential.SNMPCommunities / SysReconciliationException):
//     SQLite type-name 语法不接受方括号
//
// 实现: 经 gorm.Statement{DB: d.DB}.Parse 触发的 schema 解析会写入该 DB 实例共享的
// cacheStore;此处就地修改缓存的 *schema.Schema,后续 Migrator().AutoMigrate /
// CreateTable 的 Parse 命中同一缓存,拿到净化后的 schema(单一事实源仍是 model tag,
// 净化是 sqlite 运行期的投影,不改 model 文件,不影响 PG 语义)。
//
// 配套: 被剥离 gen_random_uuid() 默认值的列,应用层由 model BeforeCreate 钩子
// (uuid.New()) 填充 ID;函数式默认剥离后该字段以应用值写入,PG 下行为等价
// (非空 ID 直接 INSERT,DB 默认值仅对零值生效)。
func (d *Database) sanitizeSQLiteModelDefaults() error {
	modelsToSanitize := MigrateModelList()
	// sys_data_reconciliation 仅在 sqlite 分支注册进 AutoMigrate(见 AutoMigrate
	// sqlite 分支注释),不在 MigrateModelList 里;但其 tag 含 PG-only 片段
	// (DetectedAt default:now() / AppliedActions type:text[]),必须一并净化,
	// 否则 sqlite 建表报 "near \"(\": syntax error" / type-name 方括号语法错误。
	modelsToSanitize = append(modelsToSanitize, &models.SysDataReconciliation{})
	// log-errors-fix-20260818: sys_ad_service_accounts 缺表同族(同 rpa-tasks-table-missing
	// / column-config-table-recon-stats 模式)。模型 internal/models/ad_service_account.go
	// 自 Phase 36 起存在,ID 字段 tag 含 default:gen_random_uuid()(PG-only 函数式默认,
	// sqlite 不接受),必须显式加入净化列表,否则 sqlite 路径上 sys_ad_service_accounts
	// 建表报 "near \"(\": syntax error"。
	modelsToSanitize = append(modelsToSanitize, &models.ADServiceAccount{})
	// Phase 59 G-59-2: sys_dept_ou_mapping 的 ID 字段 tag 含 default:gen_random_uuid()
	// (PG-only 函数式默认,sqlite 不接受 → 建表报 "near \"(\": syntax error")。
	// 该模型在 sqlite 分支注册进 AutoMigrate,故必须一并净化;剥离后由
	// BeforeCreate 钩子(uuid.New()) 填充 ID,PG 下行为等价。
	modelsToSanitize = append(modelsToSanitize, &models.DeptOUMapping{})
	for _, model := range modelsToSanitize {
		stmt := &gorm.Statement{DB: d.DB}
		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("解析模型 schema 失败(%T): %w", model, err)
		}
		for _, field := range stmt.Schema.Fields {
			// 函数式默认值(gen_random_uuid()/now() 等)SQLite 不接受
			if strings.Contains(field.DefaultValue, "(") {
				field.DefaultValue = ""
				field.HasDefaultValue = false
			}
			// PG 数组类型(text[] 等)SQLite type-name 语法不接受方括号;
			// 降为 text — pq.StringArray 的 Valuer/Scan 走 "{a,b}" 字面量文本,可 roundtrip
			if strings.Contains(string(field.DataType), "[") {
				field.DataType = "text"
			}
		}
	}
	return nil
}

// createPostgresConnection 创建PostgreSQL连接
func createPostgresConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	adminDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode)

	// CDX-H1:createDatabaseIfNotExists 失败必须上抛而非吞错。
	// 历史:applogger.Errorf 后继续 → gorm.Open 在 DB 不存在时报 "database does not exist"
	// 而非真实根因(认证/网络/缺 postgres 维护库)。修复后启动 fail-fast 暴露真实原因。
	if err := createDatabaseIfNotExists(adminDSN, cfg.DBName); err != nil {
		return nil, fmt.Errorf("创建数据库失败: %w", err)
	}

	dsn := cfg.GetDSN()

	gormConfig := &gorm.Config{
		Logger:                                   createFilteredLogger(),
		// CDX-M-UTC:全项目 UTC 一致性,与 SQL DEFAULT NOW() 语义对齐。
		// 历史本地时区行由 timestamptz 规范化(timestamptz 列按会话时区规范化存储,
		// 混合期查询按 timestamptz 语义正确;naive timestamp 列若有则存在解释漂移)。
		NowFunc:                                  func() time.Time { return time.Now().UTC() },
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
	// 停止连接保活 goroutine(若已启动);测试手工构造的 Database 该字段为 nil。
	if d.keepaliveStop != nil {
		close(d.keepaliveStop)
		select {
		case <-d.keepaliveDone:
		case <-time.After(3 * time.Second): // ping 进行中最多等 3s,避免 shutdown 卡死
		}
		d.keepaliveStop = nil
	}
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
	// Phase 59 (GUARD-02): v_dept_ou_mismatch 普通 VIEW 引用 sys_user / sys_dept /
	// sys_dept_ou_mapping / sys_ad_config。普通 VIEW 同样会阻塞 GORM ALTER COLUMN TYPE
	// (见本函数上方 :445-451 的 2026-07-04 实测)。AutoMigrate 之前定向 DROP,
	// migration_217 在 AutoMigrate 之后重建定义。
	// 该视图无 MV 依赖(不被 reconciliation_normalized 引用),故**不需要 CASCADE**;
	// 用 RESTRICT 语义的裸 DROP,万一将来有依赖会立刻报错而不是静默级联。
	if err := d.DB.Exec("DROP VIEW IF EXISTS v_dept_ou_mismatch").Error; err != nil {
		applogger.Warnf("[dropDependent] DROP v_dept_ou_mismatch 失败(非阻断,继续 AutoMigrate): %v", err)
	}

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

// ==== sys_api_keys SEC-01 存量迁移 (debug: sys-api-keys-keyhash-23502) ====

// apikeyBackfillSaltLen 存量盐值字节长度(16 字节 → hex 编码 32 字符)。
// ⚠️ 必须与 internal/services/system/apikey_service.go 的 apiKeySaltLen 保持一致。
const apikeyBackfillSaltLen = 16

// apikeyBackfillKeyPrefixLen KeyPrefix 取明文 key 前 N 字符。
// ⚠️ 必须与 internal/services/system/apikey_service.go CreateAPIKey 的 key[:12] 保持一致。
const apikeyBackfillKeyPrefixLen = 12

// backfillAPIKeyHash 计算 SM3(key‖salt) 的 hex 摘要(64 字符)。
// ⚠️ 拼接顺序 key 在前、salt 在后,必须与 internal/services/system/apikey_service.go 的
// hashAPIKey 逐字节一致,否则存量密钥迁移后 ValidateAPIKey 恒不匹配(全部存量密钥静默失效)。
func backfillAPIKeyHash(key, salt string) string {
	h := sm3.New()
	h.Write([]byte(key))
	h.Write([]byte(salt))
	return hex.EncodeToString(h.Sum(nil))
}

// backfillGenerateSalt 生成随机盐(hex 编码,32 字符,per-key 独立)。
// ⚠️ 必须与 internal/services/system/apikey_service.go 的 generateSalt 逐字节一致。
func backfillGenerateSalt() (string, error) {
	buf := make([]byte, apikeyBackfillSaltLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// tableColumnExists 探测表列是否存在(双方言:PG information_schema / sqlite pragma_table_info)。
// 不用 Migrator().HasColumn 的原因:模型已删除 Key 字段,HasColumn(&models.APIKey{}, "key")
// 会因 schema LookUpField 失败恒为 false,无法探测模型之外的遗留列。
func (d *Database) tableColumnExists(conn *gorm.DB, table, column string) bool {
	var count int64
	var err error
	if d.Type == "sqlite" {
		err = conn.Raw(fmt.Sprintf("SELECT count(*) FROM pragma_table_info('%s') WHERE name = ?", table), column).Scan(&count).Error
	} else {
		err = conn.Raw(`SELECT count(*) FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`, table, column).Scan(&count).Error
	}
	return err == nil && count > 0
}

// backfillSysAPIKeysKeyHash 在 AutoMigrate 之前为 sys_api_keys 存量行回填
// key_hash/salt/key_prefix 三列并退役旧明文 key 列(Phase 60 / SEC-01)。
//
// 设计原因(debug: sys-api-keys-keyhash-23502): 模型 internal/models/api_key.go 已删除明文
// Key 字段,新增 KeyHash/Salt/KeyPrefix 三列且均 `not null` 无 DEFAULT。存量表(旧 key 列 +
// 已有行)上 GORM AutoMigrate 发出的 `ADD COLUMN key_hash varchar(64) NOT NULL` 会对已有行
// 填 NULL → SQLSTATE 23502 "column contains null values" 启动 FATA。AutoMigrate 先于
// migrations 执行,普通 migration_NNN 拦不住该错误,故必须在此做存量迁移
// (先例: cleanupOldConstraints / dropDependentMaterializedViews 同为 AutoMigrate 前置步骤)。
//
// 语义(幂等):
//   - 表不存在                  → 全新部署,AutoMigrate 按 model 建表,no-op
//   - 旧 key 列不存在且新列存在 → 已迁移完成,no-op
//   - 旧 key 列存在             → 事务内 ADD COLUMN(先 nullable)→ 逐行回填 →
//     SET NOT NULL(PG)→ DROP COLUMN key。旧明文 key 迁移后仍有效:
//     ValidateAPIKey(apikey_service.go) 按 KeyPrefix 缩窄候选,重算 SM3(key+salt)
//     恒定时间比对 —— 回填算法与其 generateSalt/hashAPIKey/CreateAPIKey(key[:12])
//     完全一致(见 backfillAPIKeyHash/backfillGenerateSalt 注释锚点)。
//
// 实现要点:
//   - 不能一步 ADD COLUMN NOT NULL:存量行被填 NULL → 23502(本 debug 的原始故障);
//     也不能用 Migrator().AddColumn:FullDataTypeOf 对 not null 字段同样带 NOT NULL 约束。
//   - 新列类型(varchar 64/32/12)与 model tag size 严格一致,保证 AutoMigrate 视为已对齐
//     而不再 ALTER。
//   - 约束命名: KeyHash 的 uniqueIndex(uni_sys_api_keys_key_hash)与旧 inline UNIQUE
//     (sys_api_keys_key_hash_key)的命名冲突由 cleanupOldConstraints 统一清理,此处不重复。
//   - 回填含软删行(SET NOT NULL 覆盖全表,漏行即 FATA)。
//   - sqlite 分支跳过 SET NOT NULL(SQLite 无 ALTER SET NOT NULL;仅 dev 本地文件库,
//     AutoMigrate 重建表路径会对齐 NOT NULL,回填后无 NULL 行不会失败)。
func (d *Database) backfillSysAPIKeysKeyHash() error {
	if !d.DB.Migrator().HasTable("sys_api_keys") {
		return nil // 全新部署:AutoMigrate 按 model 建表,无存量行
	}

	if !d.tableColumnExists(d.DB, "sys_api_keys", "key") {
		if d.tableColumnExists(d.DB, "sys_api_keys", "key_hash") {
			return nil // 已迁移完成,幂等 no-op
		}
		// 非预期状态(无旧列可回填):交给 AutoMigrate 按模型建列并暴露问题,此处不吞。
		applogger.Warnf("[apikey-backfill] sys_api_keys 无旧 key 列也无 key_hash 列,跳过回填(若启动仍报 23502 请人工检查存量行)")
		return nil
	}

	applogger.Infof("[apikey-backfill] 检测到 sys_api_keys 旧明文 key 列,执行 SEC-01 存量迁移(salt/key_hash/key_prefix 回填 → NOT NULL → 删除明文列)")

	backfilled := 0
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		// 1) 以 nullable 形态补齐新列(已存在则跳过,兼容上次运行中途失败的半迁移态)。
		newColumns := []struct{ name, ddl string }{
			{"key_hash", "varchar(64)"},
			{"salt", "varchar(32)"},
			{"key_prefix", "varchar(12)"},
		}
		for _, col := range newColumns {
			if !d.tableColumnExists(tx, "sys_api_keys", col.name) {
				if err := tx.Exec(fmt.Sprintf("ALTER TABLE sys_api_keys ADD COLUMN %s %s", col.name, col.ddl)).Error; err != nil {
					return fmt.Errorf("ADD COLUMN %s 失败: %w", col.name, err)
				}
			}
		}

		// 2) 逐行回填。⚠️ 算法必须与 apikey_service.go 的 generateSalt/hashAPIKey/
		//    CreateAPIKey(key[:12]) 逐字节一致,见 backfillAPIKeyHash 注释锚点。
		type legacyKeyRow struct {
			ID  string
			Key string
		}
		var rows []legacyKeyRow
		if err := tx.Raw("SELECT id, key FROM sys_api_keys WHERE key IS NOT NULL").Scan(&rows).Error; err != nil {
			return fmt.Errorf("读取存量 key 行失败: %w", err)
		}
		for _, row := range rows {
			if len(row.Key) < apikeyBackfillKeyPrefixLen {
				return fmt.Errorf("存量 key 行 %s 明文长度 %d < %d,无法派生 key_prefix,请人工处理该行后重试",
					row.ID, len(row.Key), apikeyBackfillKeyPrefixLen)
			}
			salt, err := backfillGenerateSalt()
			if err != nil {
				return fmt.Errorf("生成盐失败: %w", err)
			}
			if err := tx.Exec("UPDATE sys_api_keys SET salt = ?, key_hash = ?, key_prefix = ? WHERE id = ?",
				salt, backfillAPIKeyHash(row.Key, salt), row.Key[:apikeyBackfillKeyPrefixLen], row.ID).Error; err != nil {
				return fmt.Errorf("回填行 %s 失败: %w", row.ID, err)
			}
		}
		backfilled = len(rows)

		// 3) 回填完成后再收紧 NOT NULL,与 model not null tag 对齐 —— AutoMigrate 看到
		//    列已存在且语义一致 → no-op(不再发会触发 23502 的 ADD COLUMN NOT NULL)。
		if d.Type == "postgres" {
			for _, col := range []string{"key_hash", "salt", "key_prefix"} {
				if err := tx.Exec(fmt.Sprintf("ALTER TABLE sys_api_keys ALTER COLUMN %s SET NOT NULL", col)).Error; err != nil {
					return fmt.Errorf("SET NOT NULL %s 失败: %w", col, err)
				}
			}
		}

		// 4) 明文使命完成,删除旧列(PG 连带删除其上的 legacy uniqueIndex)。
		if err := tx.Exec("ALTER TABLE sys_api_keys DROP COLUMN key").Error; err != nil {
			return fmt.Errorf("DROP COLUMN key 失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	applogger.Infof("[apikey-backfill] 完成: 回填 %d 行,salt/key_hash/key_prefix 已就位,旧明文 key 列已删除(存量密钥经 ValidateAPIKey 仍有效)", backfilled)
	return nil
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
		&models.ConfigRestoreTask{}, // Phase 93: 配置恢复任务 (D-15 异步编排载体)
		&models.ADPushQueue{},      // Phase 57: AD 推送队列 (outbox-lite 队列表)
		// Phase 59 G-59-2: 部门-OU 映射表。其建表 DDL 原先只存在于归档 .sql
		// (archive/legacy-2026-06-15/126_create_dept_ou_mapping_table.sql) 与
		// scripts/dbprovision(新部署专用工具),而 .sql 迁移不被自动加载 ——
		// 生产库因当年手工执行过该 SQL 而有表,全新库走常规启动路径必然缺表,
		// 导致 migration_217 的 CREATE VIEW 报 SQLSTATE 42P01(视图建不出来,
		// 对账端点 500)。此处补注册进 AutoMigrate(存量库无操作,全新库建表)。
		&models.DeptOUMapping{},
		&models.DeviceMACAddress{},
		&models.DevicePortStatus{},
		&models.LLDPNeighborInfo{}, // LLDP 邻居信息(拓扑发现)
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
		// KnowledgeCategory, KnowledgeTag 已通过 SQL 迁移创建，PG 不需要 AutoMigrate;
		// sqlite 分支在 AutoMigrate() 内显式补注册(归档 SQL 不执行,且 many2many
		// 级联不会建 sys_knowledge_tag,详见下方 sqlite 分支注释)
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
		&models.OUGroupMappingSyncLog{}, // OU 组映射同步日志
		// VDI虚拟化相关模型
		&models.VDIVirtualMachine{},
		&models.VDIServer{},
		&models.VDIResourceGroup{},
		&models.VDIUserBinding{},
		&models.VDISyncLog{},
		// 运维管理相关模型（Workstation 复用系统已有的 sys_workstation）
		&operations.OpsBuilding{},
		&operations.OpsFloor{},
		&operations.OpsServerRoom{},
		&operations.OpsDedicatedLine{},
		&operations.OpsRoomDevice{},
		&operations.OpsRoomNetworkDevice{},
		&operations.OpsRoomPhoto{},
		&operations.OpsInfoPoint{},
		&models.Asset{},
		// 仪表盘系统相关模型
		&models.Dashboard{},
		&models.DashboardVersion{},
		// Phase 46 R5: 半自动修复建议表
		&models.SysReconciliationFixSuggestion{},
		// 对账异常表(原 migration_174 带 GiST 索引, 归档于 archive/applied 不自动跑;
		// 此处补注册进 AutoMigrate 建表, GiST 索引可选 — 不影响功能, 仅 IP 匹配性能)
		&models.SysReconciliationException{},
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
	// 该旁路仅对 postgres 有意义;sqlite 无 pooler 问题且新文件库必须全量 AutoMigrate。
	if os.Getenv("SKIP_AUTOMIGRATE") == "true" && d.Type == "postgres" {
		applogger.Warnf("[SKIP_AUTOMIGRATE=true] 跳过 cleanup/dropDependent/AutoMigrate,改由 BootstrapMissingTables 补建")
		return nil
	}

	// SQLite:净化 PG-only DDL 片段(gen_random_uuid() 默认值 / text[] 数组类型),
	// 否则 AutoMigrate 在建 Asset/APIKeyUsageLog/AuthCredential 等表时报语法错误。
	if d.Type == "sqlite" {
		if err := d.sanitizeSQLiteModelDefaults(); err != nil {
			return err
		}
	}

	// Phase 60 / SEC-01: sys_api_keys 存量明文 key → key_hash/salt/key_prefix 回填(双方言)。
	// 必须在 AutoMigrate 之前执行:模型新增三列均 `not null` 无 DEFAULT,存量行会被
	// ADD COLUMN NOT NULL 填 NULL → SQLSTATE 23502 启动 FATA
	// (debug: sys-api-keys-keyhash-23502;AutoMigrate 先于 migrations 执行,普通
	// migration_NNN 拦不住 —— 同 gorm-automigrate-blocked-by-matview 教训)。
	// 先例: cleanupOldConstraints / dropDependentMaterializedViews 同为 AutoMigrate 前置步骤。
	if err := d.backfillSysAPIKeysKeyHash(); err != nil {
		return fmt.Errorf("sys_api_keys key_hash 存量回填失败: %w", err)
	}

	// 先清理可能存在的旧约束。dropDependentMaterializedViews() 当前 noop,
	// 保留调用便于将来 model tag 漂移时回退 (260704-regression-fix-5)。
	if d.Type == "postgres" {
		d.cleanupOldConstraints()
		d.dropDependentMaterializedViews()
	}

	// 禁用外键约束的自动创建，避免类型不匹配的问题
	// 外键约束已通过 SQL 脚本手动创建
	migrateList := MigrateModelList()
	if d.Type == "sqlite" {
		// log-errors-fix-20260818 AutoMigrate-view-dep: GORM ALTER TABLE 在 sqlite 下走
		// __temp → RENAME 路径,而 reconciliation_normalized 视图持 schema-level 引用
		// sys_data_reconciliation,RENAME 会触发 "SQL logic error: error in view
		// reconciliation_normalized: no such table: main.sys_data_reconciliation"
		// (实测 2026-08-18 10:32 启动期,Phase 48 加 recon_category 列触发)。
		// 必须先 DROP 三视图让 ALTER 路径无依赖完成,本函数下方
		// ensureSQLiteReconciliationViews 会在 AutoMigrate 后重建它们。
		// PG 不受影响(走 advisory-lock + migration_175/176 块流)。
		if err := d.dropSQLiteReconciliationViews(); err != nil {
			return err
		}
		// sys_user_preference 历史由归档 SQL(004/005/044)创建,PG 存量库已存在;
		// 仅 sqlite 分支注册进 AutoMigrate(全新文件库必须建表,否则登录后首屏
		// GET /system/settings/preferences 500)。PG 不注册的原因:GORM 对存量表
		// 会按 model tag 发起漂移 ALTER(DROP NOT NULL / 默认值 240→280 改写等),
		// 生产语义必须零改动;PG 新部署由 scripts/dbprovision 建表。
		migrateList = append(migrateList, &models.UserPreference{})
		// sys_rpa_workers / sys_rpa_executions / sys_mac_oui_vendor 历史由归档 SQL
		// (102_add_rpa_tables.sql / 033_create_mac_oui_vendor_table.up.sql)创建,
		// PG 存量库已存在;仅 sqlite 分支注册进 AutoMigrate(全新文件库必须建表,
		// 否则 RPA 扩缩容统计查询 / OUI 厂商导入报 "no such table")。
		// PG 不注册的原因同上:避免 GORM 对存量表按 model tag 发起漂移 ALTER;
		// PG 新部署由 scripts/dbprovision 建表(已含 MACOUIVendor)。
		// 单一事实源:rpamodels.Worker/Execution(internal/models/rpa/,
		// services/rpa 实际使用;internal/models/rpa.go 的 RPAWorker/RPAExecution
		// 为无引用遗留定义,不注册)、models.MACOUIVendor。
		// 三个模型 tag 均无 PG-only DDL 片段(无函数式默认值/数组类型),
		// 无需经过 sanitizeSQLiteModelDefaults 净化。
		migrateList = append(migrateList,
			&rpamodels.Worker{},
			&rpamodels.Execution{},
			&models.MACOUIVendor{},
		)
		// rpa-tasks-table-missing (2026-08-18): sys_rpa_tasks 与上块同族缺漏
		// (同一归档 SQL 102_add_rpa_tables.sql 建表,PG 存量库已存在;上方注册
		// Worker/Execution 时漏了 Task),sqlite 全新文件库不建表则
		// /rpa/tasks/list 报 "no such table: sys_rpa_tasks"。仅 sqlite 分支注册,
		// PG 不注册的原因同上(零漂移)。单一事实源同 rpamodels.Worker/Execution。
		// rpamodels.Task tag 已核查:无函数式默认值/数组类型等 PG-only DDL 片段
		// (type:jsonb 为合法 type-name,与 SysDataReconciliation 先例同级;
		// default:300/5/0 为常量默认值),无需经过 sanitizeSQLiteModelDefaults 净化。
		// sys_rpa_schedules/variables/templates 未注册:services/api 无任何运行期
		// 查询引用,不在本症链路,避免无谓漂移。
		migrateList = append(migrateList, &rpamodels.Task{})
		// sys_oper_log 历史由归档 SQL 创建,PG 存量库已存在;仅 sqlite 分支注册进
		// AutoMigrate(全新文件库必须建表,否则 operlog.Record 写入报 "no such table")。
		// PG 不注册的原因同上(零漂移);PG 新部署由 scripts/dbprovision 建表。
		// models.OperLog tag 已核查:无函数式默认值/数组类型等 PG-only DDL 片段
		// (BaseTimeLine 的 type:uuid 是合法 type-name,ID 由 BeforeCreate 钩子填充),
		// 无需经过 sanitizeSQLiteModelDefaults 净化。
		migrateList = append(migrateList, &models.OperLog{})
		// sys_logininfor 历史由归档 SQL 创建,PG 存量库已存在;仅 sqlite 分支注册进
		// AutoMigrate(全新文件库必须建表,否则登录成功后 auth.go 记录登录日志写入报
		// "no such table: sys_logininfor")。PG 不注册的原因同上(零漂移);
		// PG 新部署由 scripts/dbprovision 建表(已含 LoginLog)。
		// models.LoginLog tag 已核查:无函数式默认值/数组类型等 PG-only DDL 片段
		// (与 OperLog 同一 BaseTimeLine,type:uuid 合法 type-name,ID 由 BeforeCreate
		// 钩子填充),无需经过 sanitizeSQLiteModelDefaults 净化。
		migrateList = append(migrateList, &models.LoginLog{})
		// sys_data_reconciliation 历史由归档 migration_168(archive/applied,启动期
		// 不执行)创建,PG 存量库已存在;仅 sqlite 分支注册进 AutoMigrate(全新文件库
		// 必须建表,否则 cron「对账-自动转工单critical/high」与异常列表查询报
		// "no such table: sys_data_reconciliation")。PG 不注册的原因同上(零漂移);
		// PG 新部署由 scripts/dbprovision 建表。
		// 注意:该模型 tag 含 PG-only 片段(DetectedAt default:now() /
		// AppliedActions type:text[]),已由 sanitizeSQLiteModelDefaults 一并净化
		// (净化列表显式追加了本模型);migration_168 的 partial unique index
		// uniq_recon_asset_type_open 不在 tag 中,sqlite 下不建 — 与
		// SysReconciliationException 的 GiST 索引同级取舍(功能不受损,仅约束降级)。
		migrateList = append(migrateList, &models.SysDataReconciliation{})
		// ops-sqlite-tables-uuid-cast (2026-08-18): 运维模块列表页依赖的 6 张表
		// 同为"历史由归档 SQL/迁移创建,运行期 sqlite 分支缺注册"缺漏类
		// (dbprovision 已注册其中 5 个,但那只服务 PG 新部署):
		//   - sys_workstation / sys_dept_location_alias / sys_dict_data(+sys_dict_type
		//     同族一并注册): /ops/workstation/list|statistics、/ops/location-alias/list、
		//     /system/dicts/data/list 报 "no such table"
		//   - ops_workstation_device: workstation List/GetByID 的 joinSelect 子查询
		//     (primary_device_serial)引用该表,缺表则列表仍 500
		//   - sys_files: floor List/GetByID LEFT JOIN 取 plan_image_url,缺表则列表仍 500
		// 六个模型 tag 已逐一核查:无函数式默认值/数组类型等 PG-only DDL 片段
		// (SysFile.Metadata 的 type:jsonb 为合法 type-name,非 STRICT sqlite 表可接受,
		// 与 SysDataReconciliation 的 jsonb 列同级);SysDeptLocationAlias 的 partial
		// unique index(migration_165)不在 tag 中,sqlite 下不建 — 同 GiST 取舍。
		// models.WorkstationDevice 虽带 3 个 GORM 关联指针,但本配置
		// DisableForeignKeyConstraintWhenMigrating=true,不产生 FK/级联 DDL。
		migrateList = append(migrateList,
			&models.Workstation{},
			&models.SysDeptLocationAlias{},
			&models.DictType{},
			&models.DictData{},
			&models.WorkstationDevice{},
			&sysmodels.SysFile{},
		)
		// kb-tag-table-stats-400 (2026-08-18): 知识库 sys_knowledge_tag 缺表同类缺漏。
		// sys_knowledge_category / sys_knowledge_tag 历史由归档 SQL(legacy 012)创建,
		// PG 存量库已存在;sqlite 分支此前依赖 GORM AutoMigrate 的关系级联"碰巧"建出
		// sys_knowledge_category(Article.Category belongs-to 依赖在 ReorderModels
		// valuesMap 存值前 append,级联生效),但 many2many 远端(Article.Tags→
		// KnowledgeTag)的 append 位于 defer 闭包、map 存值后执行,slice 扩容后旧副本
		// 丢失 → sys_knowledge_tag 永不建表,/knowledge/tags/all 报 "no such table"。
		// 显式注册两模型,不再依赖 cascade 的未定义行为;PG 不注册的原因同上(零漂移)。
		// 两模型 tag 已核查:无函数式默认值/数组类型等 PG-only DDL 片段(KnowledgeTag
		// 的 ID 由 BeforeCreate 钩子填充),无需经过 sanitizeSQLiteModelDefaults 净化。
		migrateList = append(migrateList,
			&models.KnowledgeCategory{},
			&models.KnowledgeTag{},
		)
		// column-config-table-recon-stats (2026-08-18): sys_user_column_config 缺表
		// 同族第四次缺漏(同 kb-tag-table-stats-400 / rpa-tasks-table-missing 模式 ——
		// 模型已存在 internal/models/user_column_config.go,历史从未注册进 AutoMigrate
		// 任何分支,sys_user_column_config 永远不被建,/system/column-config/asset.list 500
		// "no such table: sys_user_column_config")。仅 sqlite 分支注册:PG 存量库已存在
		// (假设 production 已 seed),GORM 对存量表按 model tag 发起漂移 ALTER 会破坏
		// 生产语义(零漂移是项目惯例);PG 新部署由 scripts/dbprovision 建表。
		// 模型 tag 已核查:无函数式默认值(`BeforeCreate` 钩子填充 ID)/数组类型等
		// PG-only DDL 片段(type:uuid/bool/int/size:100 均为合法 type-name),无需经过
		// sanitizeSQLiteModelDefaults 净化。
		migrateList = append(migrateList, &models.UserColumnConfig{})
		// log-errors-fix-20260818: sys_ad_service_accounts 缺表同族(同
		// rpa-tasks-table-missing / kb-tag-table-stats-400 /
		// column-config-table-recon-stats 模式)。Phase 36 引入的多账号池表
		// internal/models/ad_service_account.go 历史由归档 SQL migration_162
		// 创建,PG 存量库已存在;sqlite 全新文件库从未被任何 AutoMigrate 分支注册,
		// 导致 ADAccountPool.RecoverExpiredBreakers 每 5 分钟 cron 触发
		// "no such table: sys_ad_service_accounts"(实测 585+ 次持续失败)。
		// 仅 sqlite 分支注册:PG 不注册的原因同 UserPreference(零漂移,GORM 对存量表
		// 按 model tag 发起漂移 ALTER 会破坏生产语义);PG 新部署由 scripts/dbprovision 建表。
		// 模型 tag 已核查:无数组类型 text[] 等 PG-only DDL 片段(type:uuid/size:255/500/64
		// 均为合法 type-name),但 ID 字段 default:gen_random_uuid() 是 PG-only 函数式默认,
		// 已在上方 sanitizeSQLiteModelDefaults 显式追加本模型净化,SQLite 下默认剥离由应用层
		// uuid.New() 填充(BeforeCreate 钩子),PG 下行为等价。
		migrateList = append(migrateList, &models.ADServiceAccount{})
		// Phase 59 G-59-2: sys_dept_ou_mapping 缺表同族(与 ADServiceAccount 同款)。
		// 建表 DDL 只在归档 .sql(126_create_dept_ou_mapping_table.sql)与
		// scripts/dbprovision 里,两者都不在常规启动路径上 → sqlite 全新文件库
		// 缺表 → migration_217 建视图报 42P01 → 对账端点 500。
		// ID 字段 default:gen_random_uuid() 是 PG-only 函数式默认,已在
		// sanitizeSQLiteModelDefaults 显式追加本模型净化(sqlite 下由
		// BeforeCreate 钩子 uuid.New() 填充 ID,PG 下行为等价)。
		migrateList = append(migrateList, &models.DeptOUMapping{})
	}
	err := d.DB.Migrator().AutoMigrate(migrateList...)
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
	//
	// C3 修复:多副本/滚动重启下 REFRESH CONCURRENTLY 与 CREATE OR REPLACE VIEW 竞态失败
	// ("tuple concurrently updated" / "CONCURRENTLY refresh in progress")。
	// 用会话级 advisory lock 包裹整块迁移,未获锁实例 WARN 跳过(fail-safe 而非 fail-deadly):
	// 单实例部署锁永远可得,只有 HA/滚动重启场景才落到跳过。
	// SQLite 不需要此锁(d.Type guard 已排除);锁键 'xingran-migrations' 由 hashtext 哈希为 int4
	// 走单参 pg_advisory_lock 变体。
	if d.Type == "postgres" {
		if !d.acquireMigrationAdvisoryLock() {
			applogger.Warnf("[advisory-lock] 另一实例正在执行启动迁移,本实例跳过 175/176/202-205 迁移块")
			return nil
		}
		defer d.releaseMigrationAdvisoryLock()

		// Phase 84 PMODEL-04: 迁移记录表 —— 必须先于所有 runAndRecord 执行（建表 + seed）。
		// 非阻断：表建不出来时后续 upsert 只 WARN，迁移照常执行（记录失败不阻断启动）。
		if err := d.runAndRecord(225, "schema_migrations_table", migrations.Migrate225SchemaMigrationsTable); err != nil {
			applogger.Errorf("迁移记录表建表失败 (非阻断,启动继续,对账能力缺失): %v", err)
		}

		if err := d.runAndRecord(175, "reconciliation_physical_link", migrations.Migrate175ReconciliationPhysicalLink); err != nil {
			applogger.Errorf("reconciliation 前置视图重建失败 (非阻断,留待下次启动): %v", err)
		}
		if err := d.runAndRecord(176, "reconciliation_physical_mv", migrations.Migrate176ReconciliationPhysicalMV); err != nil {
			applogger.Errorf("reconciliation_normalized MV 重建失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 52: 端口写审计表(由 AutoMigrate 建表) + 菜单 seed + 角色授权
		if err := d.runAndRecord(202, "port_write_audit", migrations.Migrate202PortWriteAudit); err != nil {
			applogger.Errorf("端口写审计迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// 连接池配置 sys_config seed (max_connections / max_idle_seconds, web 可配, 默认 50/300s)
		if err := d.runAndRecord(203, "connection_pool_sys_config", migrations.Migrate203ConnectionPoolSysConfig); err != nil {
			applogger.Errorf("连接池配置 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// 锐捷 dot1x default-user-limit 缓存列 (enable 必须显式恢复 N)
		if err := d.runAndRecord(204, "dot1x_user_limit", migrations.Migrate204AddDot1xUserLimit); err != nil {
			applogger.Errorf("dot1x_user_limit 列迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// RPA Worker id 列补 DEFAULT gen_random_uuid()
		// (Register 原生 SQL 路径绕过 BeforeCreate, 列需自带 DEFAULT 与全库 UUID 惯例对齐, 防 23502)
		if err := d.runAndRecord(205, "rpa_worker_id_default", migrations.Migrate205RpaWorkerIdDefault); err != nil {
			applogger.Errorf("sys_rpa_workers.id DEFAULT 迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// 登录菜单加载慢: 为关联表增加复合索引, 缓解远程 DB 全表扫描导致的超时
		if err := d.runAndRecord(206, "user_role_role_menu_indexes", migrations.Migrate206AddUserRoleRoleMenuIndexes); err != nil {
			applogger.Errorf("sys_user_role / sys_role_menu / sys_menu 索引迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// 规范菜单目录种子 (239 条, 幂等; debug admin-role-incomplete-menus 修复)
		// 必须在 InitData 的 createOperationsManagementMenus 之前执行:
		// 规范「运维管理」根就位后, Go 种子按 (name, parent) 查重自然全部跳过
		if err := d.runAndRecord(207, "seed_canonical_menu_catalog", migrations.Migrate207SeedCanonicalMenuCatalog); err != nil {
			applogger.Errorf("规范菜单目录种子失败 (非阻断,留待下次启动): %v", err)
		}
		// 字典 seed (11 组, 幂等; Phase 69 DICT-02: dev 库字典表为空的修复)
		if err := d.runAndRecord(208, "dict_seed", migrations.Migrate208DictSeed); err != nil {
			applogger.Errorf("字典 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// 系统设置菜单 component 路径修正 (Phase 70 D-11: 前端 settings-page 目录
		// 并入 system/settings; 幂等 UPDATE, changed 标志供 core.go 失效菜单缓存)
		changed209 := false
		if err := d.runAndRecord(209, "update_settings_menu_component", func(db *gorm.DB) error {
			var err error
			changed209, err = migrations.Migrate209UpdateSettingsMenuComponent(db)
			return err
		}); err != nil {
			applogger.Errorf("系统设置菜单 component 修正失败 (非阻断,留待下次启动): %v", err)
		} else if changed209 {
			d.SettingsMenuComponentChanged = true
		}
		// Q-11: 归一化 sys_menu.parent_id='0' 为 NULL(幂等,修复 Update 路径落库字面 "0" 造成的孤儿节点)
		if err := d.runAndRecord(210, "normalize_menu_parent_id", func(db *gorm.DB) error {
			_, err := migrations.Migrate210NormalizeMenuParentID(db)
			return err
		}); err != nil {
			applogger.Errorf("sys_menu parent_id='0' 归一失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 93: 配置恢复任务表 (PG 显式建表,幂等;sqlite 由 AutoMigrate 建表,分支无需注册)
		if err := d.runAndRecord(211, "create_config_restore_task", migrations.Migrate211CreateConfigRestoreTask); err != nil {
			applogger.Errorf("配置恢复任务表创建失败 (非阻断,留待下次启动): %v", err)
		}
		// v129-recheck C-1: 同设备活跃恢复任务部分唯一索引 (双方言;PG 分支在 advisory-lock 块内执行)
		if err := d.runAndRecord(212, "restore_task_active_unique_index", migrations.Migrate212CreateRestoreTaskActiveUniqueIndex); err != nil {
			applogger.Errorf("恢复任务活跃唯一索引创建失败 (非阻断,留待下次启动): %v", err)
		}
		// 261013-lqb: 楼栋 total_floors 存量漂移一次性归正 (双方言纯 SQL,幂等,无常驻对象)
		if err := d.runAndRecord(213, "recompute_building_total_floors", migrations.Migrate213RecomputeBuildingTotalFloors); err != nil {
			applogger.Errorf("楼栋 total_floors 存量重算失败 (非阻断,留待下次启动): %v", err)
		}
		// quick 261016-ad-login-history: AD 设备登录历史月分区表 (PG-only,幂等;
		// sqlite 不建表,writer/query 侧由 addomain 包 isPostgres 守卫跳过。
		// 对应模型禁止加入 AutoMigrate 列表 — GORM 对分区父表
		// 做 ALTER 会触发 42P16 类问题)
		if err := d.runAndRecord(214, "create_ad_login_history", migrations.Migrate214CreateADLoginHistory); err != nil {
			applogger.Errorf("AD登录历史分区表创建失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 57 F-04: AD 推送队列表 + 部分唯一索引 (双方言 DDL,幂等)。
		// 与 AutoMigrate 内的建表分开:模型刻意不声明 uniqueIndex tag(见 ensureADPushQueue
		// 注释),索引只能由显式 DDL 建。**失败阻断启动** —— uq_ad_push_queue_active 是每次
		// 入队 ON CONFLICT 的硬前提,缺失会让运行期第一次用户编辑报 42P10 并因入队失败
		// 回滚整个业务事务(用户管理全线不可用),而真实原因只留在几小时前的启动日志里。
		if err := d.ensureADPushQueue(); err != nil {
			return err
		}
		// Phase 58 D-05: ad_push_queue 排水间隔单旋钮 seed（幂等）。
		// **非阻断**：缺旋钮只会让 LoadDrainIntervalSeconds 回落默认 30s，
		// 与队列表建表的"失败阻断启动"档（uq_ad_push_queue_active 是入队硬前提）不同档。
		if err := d.runAndRecord(216, "ad_push_drain_sys_config", migrations.Migrate216ADPushDrainSysConfig); err != nil {
			applogger.Errorf("ad_push_queue 排水间隔 sys_config seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 59 GUARD-02: 部门↔OU 对账普通视图 v_dept_ou_mismatch（双方言 DDL，幂等）。
		// **非阻断**：视图缺失只让对账端点 500，不影响登录 / 入队 / 排水主链路。
		// 视图在 AutoMigrate 之前被 dropDependentMaterializedViews() DROP 过，此处重建。
		if err := d.runAndRecord(217, "dept_ou_mismatch_view", migrations.Migrate217DeptOUMismatchView); err != nil {
			applogger.Errorf("v_dept_ou_mismatch 视图创建失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 59 GUARD-02: 对账查询按钮权限 seed（幂等，精准授权）。
		// **非阻断**：缺权限行只让普通角色对账端点 403 / 前端 Tab 不渲染，
		// 不影响登录 / 入队 / 排水主链路；父菜单缺失时迁移内部降级返回 nil。
		if err := d.runAndRecord(218, "ad_reconciliation_permission", migrations.Migrate218ADReconciliationPermission); err != nil {
			applogger.Errorf("对账查询按钮权限 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// AD 域控用户同步修复: sys_user 补 mobile/title/ad_config_id 列 + sys_ad_user 补 manager 列
		// (双方言幂等,走 GORM Migrator 的 HasColumn/AddColumn)。
		// **非阻断**：缺列只让 AD 同步丢弃手机号/职位、多域隔离退化为按用户名匹配。
		if err := d.runAndRecord(219, "user_ad_sync_fields", migrations.Migrate219UserADSyncFields); err != nil {
			applogger.Errorf("用户/AD影子表 AD 同步列补建失败 (非阻断,留待下次启动): %v", err)
		}
		// 通用经理同步: sys_manager_org / sys_manager_rule 两张表 (AutoMigrate 建表,幂等)。
		// **非阻断**：缺表只让经理同步端点 500,不影响登录 / 入队 / 排水主链路。
		if err := d.runAndRecord(220, "manager_rule_tables", migrations.Migrate220ManagerRuleTables); err != nil {
			applogger.Errorf("经理同步规则表创建失败 (非阻断,留待下次启动): %v", err)
		}
		// 通用经理同步: 经理同步规则/机构两个菜单 + 按钮权限 seed (幂等,精准授权)。
		// **非阻断**：缺菜单只让普通角色无入口(admin 超管旁路可见),不影响主链路。
		if err := d.runAndRecord(221, "manager_sync_menus", migrations.Migrate221ManagerSyncMenus); err != nil {
			applogger.Errorf("经理同步菜单 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// 清理 4 张幻影关联表 (user_posts/user_roles/role_menus/role_depts):
		// many2many 标签早期写错表名导致 GORM 额外建表,真实关联表是
		// sys_user_post/sys_user_role/sys_role_menu/sys_role_dept。
		// 仅删 0 行表;有数据则只告警。**非阻断**：失败只留残余空表,不影响主链路。
		if err := d.runAndRecord(222, "drop_phantom_join_tables", migrations.Migrate222DropPhantomJoinTables); err != nil {
			applogger.Errorf("幻影关联表清理失败 (非阻断,留待下次启动): %v", err)
		}
		// 通用经理同步: sys_manager_org.dept_id 唯一索引改为部分唯一 (WHERE deleted_at IS NULL),
		// 并物理清理已软删机构行——修复「删除机构后无法重建同部门机构」(23505)。
		// **非阻断**：失败只保留旧索引语义,不影响主链路。
		if err := d.runAndRecord(223, "manager_org_partial_unique", migrations.Migrate223ManagerOrgPartialUnique); err != nil {
			applogger.Errorf("机构表部分唯一索引迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// 域控审查: seed dept_to_ad_sync 定时任务(P1-1) + sys_ou_group_mapping
		// 唯一键扩展加入 ad_config_id(P1-7 多域支持)。**非阻断**：失败降级 WARN。
		if err := d.runAndRecord(224, "dept_sync_seed_and_ou_group_unique", migrations.Migrate224DeptSyncSeedAndOUGroupUnique); err != nil {
			applogger.Errorf("部门同步seed与OU组唯一键迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 87 动作级 F 型权限种子与精准授权 (RBAC-05)。**非阻断**：
		// 缺权限行只让普通角色写操作 403 / 前端按钮不显示，不影响登录 / 入队 / 排水主链路；父菜单缺失时迁移内部降级返回 nil。
		if err := d.runAndRecord(226, "action_permission_f_type_seed", migrations.Migrate226ActionPermissionFTypeSeed); err != nil {
			applogger.Errorf("动作级权限种子迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// Phase 87 批次二 7 条动作级权限补播 (OQ-6 用户确认值，RBAC-05)。**非阻断**。
		if err := d.runAndRecord(227, "action_permission_f_type_seed_hold", migrations.Migrate227ActionPermissionFTypeSeedHold); err != nil {
			applogger.Errorf("批次二动作级权限补播迁移失败 (非阻断,留待下次启动): %v", err)
		}
	} else {
		// sqlite 分支: Phase 84 PMODEL-04 迁移记录表 —— 必须先于所有 runAndRecord 执行。
		if err := d.runAndRecord(225, "schema_migrations_table", migrations.Migrate225SchemaMigrationsTable); err != nil {
			applogger.Errorf("迁移记录表建表失败 (非阻断,启动继续,对账能力缺失): %v", err)
		}
		// sqlite 分支: 规范菜单目录种子 (双方言迁移; PG 分支在上方 advisory-lock 块内执行)
		if err := d.runAndRecord(207, "seed_canonical_menu_catalog", migrations.Migrate207SeedCanonicalMenuCatalog); err != nil {
			applogger.Errorf("规范菜单目录种子失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 字典 seed (双方言迁移; PG 分支在上方 advisory-lock 块内执行)
		if err := d.runAndRecord(208, "dict_seed", migrations.Migrate208DictSeed); err != nil {
			applogger.Errorf("字典 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 系统设置菜单 component 路径修正 (双方言迁移; PG 分支在上方
		// advisory-lock 块内执行)
		changed209 := false
		if err := d.runAndRecord(209, "update_settings_menu_component", func(db *gorm.DB) error {
			var err error
			changed209, err = migrations.Migrate209UpdateSettingsMenuComponent(db)
			return err
		}); err != nil {
			applogger.Errorf("系统设置菜单 component 修正失败 (非阻断,留待下次启动): %v", err)
		} else if changed209 {
			d.SettingsMenuComponentChanged = true
		}
		// sqlite 分支: Q-11 归一化 sys_menu.parent_id='0' 为 NULL
		if err := d.runAndRecord(210, "normalize_menu_parent_id", func(db *gorm.DB) error {
			_, err := migrations.Migrate210NormalizeMenuParentID(db)
			return err
		}); err != nil {
			applogger.Errorf("sys_menu parent_id='0' 归一失败 (非阻断,留待下次启动): %v", err)
		}
		// v129-recheck C-1: 同设备活跃恢复任务部分唯一索引 (双方言;sqlite 3.8+ 支持 partial index)
		if err := d.runAndRecord(212, "restore_task_active_unique_index", migrations.Migrate212CreateRestoreTaskActiveUniqueIndex); err != nil {
			applogger.Errorf("恢复任务活跃唯一索引创建失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 261013-lqb 楼栋 total_floors 存量漂移一次性归正 (双方言纯 SQL,幂等)
		if err := d.runAndRecord(213, "recompute_building_total_floors", migrations.Migrate213RecomputeBuildingTotalFloors); err != nil {
			applogger.Errorf("楼栋 total_floors 存量重算失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: Phase 57 AD 推送队列表 + 部分唯一索引 (与 PG 分支同款调用 + 同样阻断,
		// sqlite 既是 dev 运行时也是测试运行时,必须建到表)
		if err := d.ensureADPushQueue(); err != nil {
			return err
		}
		// sqlite 分支: Phase 58 D-05 排水间隔单旋钮 seed（与 PG 分支同款调用 + 同款非阻断档；
		// 只注册 PG 分支会让 sqlite dev/测试环境无旋钮行、行为与生产不一致）。
		if err := d.runAndRecord(216, "ad_push_drain_sys_config", migrations.Migrate216ADPushDrainSysConfig); err != nil {
			applogger.Errorf("ad_push_queue 排水间隔 sys_config seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: Phase 59 GUARD-02 部门↔OU 对账视图（与 PG 分支同款调用 + 同款非阻断档）。
		// sqlite 既是 dev 运行时也是测试运行时；只注册 PG 分支会让 dev 环境缺视图。
		// 视图在 AutoMigrate 之前被 dropSQLiteReconciliationViews() DROP 过，此处重建。
		if err := d.runAndRecord(217, "dept_ou_mismatch_view", migrations.Migrate217DeptOUMismatchView); err != nil {
			applogger.Errorf("v_dept_ou_mismatch 视图创建失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: Phase 59 GUARD-02 对账查询按钮权限 seed（与 PG 分支同款调用 + 同款非阻断档）。
		// sqlite 既是 dev 运行时也是测试运行时；只注册 PG 分支会让 dev 环境缺权限行。
		if err := d.runAndRecord(218, "ad_reconciliation_permission", migrations.Migrate218ADReconciliationPermission); err != nil {
			applogger.Errorf("对账查询按钮权限 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: AD 域控用户同步修复补列 (与 PG 分支同款调用 + 同款非阻断档)。
		// sqlite 既是 dev 运行时也是测试运行时;只注册 PG 分支会让 dev 环境缺列。
		if err := d.runAndRecord(219, "user_ad_sync_fields", migrations.Migrate219UserADSyncFields); err != nil {
			applogger.Errorf("用户/AD影子表 AD 同步列补建失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 通用经理同步两张表 (与 PG 分支同款调用 + 同款非阻断档)。
		if err := d.runAndRecord(220, "manager_rule_tables", migrations.Migrate220ManagerRuleTables); err != nil {
			applogger.Errorf("经理同步规则表创建失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 经理同步菜单 seed (与 PG 分支同款调用 + 同款非阻断档)。
		if err := d.runAndRecord(221, "manager_sync_menus", migrations.Migrate221ManagerSyncMenus); err != nil {
			applogger.Errorf("经理同步菜单 seed 失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 幻影关联表清理 (与 PG 分支同款调用 + 同款非阻断档)。
		if err := d.runAndRecord(222, "drop_phantom_join_tables", migrations.Migrate222DropPhantomJoinTables); err != nil {
			applogger.Errorf("幻影关联表清理失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 机构表部分唯一索引 (与 PG 分支同款调用 + 同款非阻断档)。
		if err := d.runAndRecord(223, "manager_org_partial_unique", migrations.Migrate223ManagerOrgPartialUnique); err != nil {
			applogger.Errorf("机构表部分唯一索引迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: 部门同步 seed + OU 组唯一键扩展 (与 PG 分支同款 + 非阻断档)。
		if err := d.runAndRecord(224, "dept_sync_seed_and_ou_group_unique", migrations.Migrate224DeptSyncSeedAndOUGroupUnique); err != nil {
			applogger.Errorf("部门同步seed与OU组唯一键迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: Phase 87 动作级 F 型权限种子与精准授权 (RBAC-05，双方言迁移)。
		// sqlite 既是 dev 运行时也是测试运行时；只注册 PG 分支会让 dev 环境缺权限行。
		if err := d.runAndRecord(226, "action_permission_f_type_seed", migrations.Migrate226ActionPermissionFTypeSeed); err != nil {
			applogger.Errorf("动作级权限种子迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite 分支: Phase 87 批次二 7 条动作级权限补播 (OQ-6 用户确认值，RBAC-05，双方言迁移)。
		if err := d.runAndRecord(227, "action_permission_f_type_seed_hold", migrations.Migrate227ActionPermissionFTypeSeedHold); err != nil {
			applogger.Errorf("批次二动作级权限补播迁移失败 (非阻断,留待下次启动): %v", err)
		}
		// sqlite-recon-normalized-view (2026-08-18): 补建 reconciliation 三视图
		// (PG 侧由上方 175/176 迁移块创建 MV+前置 VIEW;sqlite 分支不执行 PG-only
		// 迁移,此前 probeMaterializedView 诚实探测后走 fallback 降级 + 每请求 WARN)。
		// 普通 VIEW 实时计算,dev 语义等价或更优;失败非阻断(仍有 fallback 兜底)。
		if err := d.ensureSQLiteReconciliationViews(); err != nil {
			applogger.Errorf("sqlite reconciliation 视图补建失败 (非阻断,exception/list 走 fallback 降级): %v", err)
		}
	}

	return nil
}

// acquireMigrationAdvisoryLock 获取 PG 会话级 advisory lock 用于启动期迁移块排他。
// 返回 true 表示已获锁可执行迁移块;返回 false 表示其他实例正在迁移,本实例应跳过。
// 必须配对调用 releaseMigrationAdvisoryLock 释放(在 defer 中)。
//
// 实现要点:
//   - 专用 sql.Conn(pinning 单连接):advisory lock 是会话级,跨连接不生效
//   - pg_try_advisory_lock(hashtext('xingran-migrations')) 返回 bool:
//     true=获锁,false=其他会话已持锁(本实例跳过)
//   - 取锁本身失败(conn 错误/查询错误)→ Errorf 后按"未获锁"处理(fail-safe)
func (d *Database) acquireMigrationAdvisoryLock() bool {
	sqlDB, err := d.DB.DB()
	if err != nil {
		applogger.Errorf("[advisory-lock] 获取 *sql.DB 失败: %v", err)
		return false
	}

	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		applogger.Errorf("[advisory-lock] 获取专用连接失败: %v", err)
		return false
	}

	var acquired bool
	if err := conn.QueryRowContext(context.Background(),
		"SELECT pg_try_advisory_lock(hashtext('xingran-migrations'))").Scan(&acquired); err != nil {
		applogger.Errorf("[advisory-lock] 尝试获取锁失败: %v", err)
		_ = conn.Close()
		return false
	}

	if !acquired {
		_ = conn.Close()
		return false
	}

	// 保存 conn 到 d 上供 releaseMigrationAdvisoryLock 使用(嵌入结构体避免污染 Database 公共字段)
	d.migrationLockConn = conn
	return true
}

// releaseMigrationAdvisoryLock 释放 acquireMigrationAdvisoryLock 获取的会话级 lock。
// 仅在 acquireMigrationAdvisoryLock 返回 true 时调用。
func (d *Database) releaseMigrationAdvisoryLock() {
	if d.migrationLockConn == nil {
		return
	}
	// pg_advisory_unlock 与 pg_try_advisory_lock 用同样的 hashtext 锁键,
	// 必须使用同一连接(pg_advisory_unlock 在已关闭连接上调用会静默 noop)。
	if _, err := d.migrationLockConn.ExecContext(context.Background(),
		"SELECT pg_advisory_unlock(hashtext('xingran-migrations'))"); err != nil {
		applogger.Errorf("[advisory-lock] 释放锁失败: %v", err)
	}
	_ = d.migrationLockConn.Close()
	d.migrationLockConn = nil
}

// runAndRecord 执行单个迁移函数并在成功后把版本 upsert 进 sys_schema_migrations
// （Phase 84 PMODEL-04：迁移记录表）。
//
// 方向锁定（TestRunAndRecord_NonBlocking）：
//   - fn（迁移本身）的错误**原样返回** —— 记录层绝不能吞掉迁移错误；
//   - 记录（upsert）失败只 WARN，**绝不返回 error、绝不阻断启动** —— 记录表是纯观测面，
//     表缺失 / 写失败时迁移照常执行、启动照常继续；
//   - **不做任何 skip / 版本比较 / 执行控制**：迁移每次启动照常无条件执行
//     （幂等重跑语义零改变），记录只用于事后对账（migrations.MissingRecords）。
func (d *Database) runAndRecord(version int, name string, fn func(*gorm.DB) error) error {
	if err := fn(d.DB); err != nil {
		return err
	}
	upsertSQL := `INSERT INTO sys_schema_migrations (version, name, applied_at)
		VALUES (?, ?, now())
		ON CONFLICT (version) DO UPDATE SET name = EXCLUDED.name, applied_at = now()`
	if d.Type != "postgres" {
		upsertSQL = `INSERT INTO sys_schema_migrations (version, name, applied_at)
			VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (version) DO UPDATE SET name = EXCLUDED.name, applied_at = CURRENT_TIMESTAMP`
	}
	if err := d.DB.Exec(upsertSQL, version, name).Error; err != nil {
		applogger.Warnf("[migration-record] 记录迁移版本 %d (%s) 失败（非阻断，迁移本身已执行）: %v", version, name, err)
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

// BootstrapMissingTables 在跳过 AutoMigrate 时,经 gorm.Migrator().CreateTable 从 model
// 派生补建缺失表 + 显式索引兜底。
//
// 设计原因: Supabase pooler (Session mode 5432) 上 GORM AutoMigrate(80+ DDL)
// 会卡死在 dropDependent 之后,所有表都在但 sys_api_keys + sys_api_key_usage_logs
// 永远建不出来。历史上本函数用硬编码 CREATE TABLE raw SQL 补建 —— 但那成了
// APIKey schema 的第三份拷贝(model tag + MigrateModelList 之外),model 加列即漂移(C7)。
//
// C7 修复: 表结构改由 models.APIKey / models.APIKeyUsageLog 经
// gorm.Migrator().CreateTable 派生 —— 与 AutoMigrate(MigrateModelList) 同一事实源,
// 天然防漂移。CreateTable 在 PrepareStmt:false 连接上走 simple protocol,
// 单表单条 DDL,无 AutoMigrate 80+ DDL 批量路径的 pooler 死锁(原硬编码 DDL 的存在理由),
// 且不带 public. 硬编码 schema 前缀(跟随 search_path)。
//
// 索引: model tag 之外的索引面(usage log 的查询索引等)由六条显式
// CREATE INDEX IF NOT EXISTS 兜底,幂等可重入。
//
// 适用: dev/调试;生产模式(mode=release)由 core.go initDBAndData 直接 fatal,
// 不会走到本函数。
func (d *Database) BootstrapMissingTables() error {
	// 0) SQLite:净化 PG-only DDL 片段(同 AutoMigrate 路径;CreateTable 直接走
	//    schema 缓存,不净化则 sys_api_key_usage_logs 的 gen_random_uuid() 报语法错误)
	if d.Type == "sqlite" {
		if err := d.sanitizeSQLiteModelDefaults(); err != nil {
			return err
		}
	}

	// 1) 表结构:model 派生,先判定再补建(幂等)
	if !d.DB.Migrator().HasTable(&models.APIKey{}) {
		applogger.Infof("[BootstrapMissingTables] sys_api_keys 缺失,经 CreateTable 从 model 派生补建")
		if err := d.DB.Migrator().CreateTable(&models.APIKey{}); err != nil {
			return fmt.Errorf("创建 sys_api_keys 失败: %w", err)
		}
	}
	if !d.DB.Migrator().HasTable(&models.APIKeyUsageLog{}) {
		applogger.Infof("[BootstrapMissingTables] sys_api_key_usage_logs 缺失,经 CreateTable 从 model 派生补建")
		if err := d.DB.Migrator().CreateTable(&models.APIKeyUsageLog{}); err != nil {
			return fmt.Errorf("创建 sys_api_key_usage_logs 失败: %w", err)
		}
	}

	// 1b) Phase 57 F-03: ad_push_queue 是 AD 推送入队的事务内载体,缺表 = 用户编辑/删除
	// 整体 500 并回滚(入队失败按 D-02 第二层回滚业务事务)。该表在 AutoMigrate 路径由
	// Migrate215CreateADPushQueue 建,本旁路路径原本完全缺失。
	//
	// 为什么走 migration_215 而不是 CreateTable(model):模型刻意不声明 uniqueIndex tag
	// (AutoMigrate 的 DROP 无 IF EXISTS,与手写迁移建的 uq_ad_push_queue_active 冲突会
	// 启动 FATA —— W-1),故部分唯一索引无法由 model 派生。裸 CreateTable 会留下一张没有
	// uq_ad_push_queue_active 的表,运行期第一次 ON CONFLICT 即 42P10。DDL 与 AutoMigrate
	// 路径复用同一迁移函数 = 同一事实源,无双份 DDL 可漂移。
	if err := d.ensureADPushQueue(); err != nil {
		return err
	}

	// Phase 84 PMODEL-04：SKIP_AUTOMIGRATE 旁路同样建迁移记录表并 seed（否则旁路库
	// 永无对账能力，且后续 runAndRecord upsert 只能 WARN）。非阻断同档。
	if err := d.runAndRecord(225, "schema_migrations_table", migrations.Migrate225SchemaMigrationsTable); err != nil {
		applogger.Errorf("迁移记录表建表失败 (非阻断,启动继续,对账能力缺失): %v", err)
	}

	// Phase 58：SKIP_AUTOMIGRATE 旁路同样要 seed 排水间隔旋钮（F-03 同款盲区）。
	// 非阻断：缺旋钮只会让 LoadDrainIntervalSeconds 回落默认 30s。
	// 旁路同样经 runAndRecord 留痕 —— 只记录其真实执行的子集（215-220）。
	if err := d.runAndRecord(216, "ad_push_drain_sys_config", migrations.Migrate216ADPushDrainSysConfig); err != nil {
		applogger.Errorf("ad_push_queue 排水间隔 sys_config seed 失败 (非阻断,留待下次启动): %v", err)
	}

	// Phase 59 GUARD-02：SKIP_AUTOMIGRATE 旁路同样要建部门↔OU 对账视图（F-03 同款盲区）。
	// 非阻断：视图缺失只让对账端点 500；该旁路不跑 AutoMigrate，故也不存在视图阻塞 ALTER 的问题。
	if err := d.runAndRecord(217, "dept_ou_mismatch_view", migrations.Migrate217DeptOUMismatchView); err != nil {
		applogger.Errorf("v_dept_ou_mismatch 视图创建失败 (非阻断,留待下次启动): %v", err)
	}

	// Phase 59 GUARD-02：SKIP_AUTOMIGRATE 旁路同样要 seed 对账查询按钮权限（F-03 同款盲区）。
	// 非阻断：缺权限行只让普通角色对账端点 403 / 前端 Tab 不渲染。
	if err := d.runAndRecord(218, "ad_reconciliation_permission", migrations.Migrate218ADReconciliationPermission); err != nil {
		applogger.Errorf("对账查询按钮权限 seed 失败 (非阻断,留待下次启动): %v", err)
	}

	// AD 域控用户同步修复：SKIP_AUTOMIGRATE 旁路同样要补列（F-03 同款盲区）。
	// 该旁路不跑 AutoMigrate，models.User 新增的 mobile/title/ad_config_id 列不会被
	// 自动补上 —— 缺列会让所有 sys_user 查询（GORM 自动 SELECT 新列）报 42703/无此列，
	// 用户管理整体不可用，故此处**失败阻断**（与 ensureADPushQueue 同档）。
	if err := d.runAndRecord(219, "user_ad_sync_fields", migrations.Migrate219UserADSyncFields); err != nil {
		return fmt.Errorf("用户/AD影子表 AD 同步列补建失败: %w", err)
	}

	// 通用经理同步：SKIP_AUTOMIGRATE 旁路同样要建 sys_manager_org / sys_manager_rule。
	// 非阻断：缺表只让经理同步端点 500，不影响登录 / 入队 / 排水主链路。
	if err := d.runAndRecord(220, "manager_rule_tables", migrations.Migrate220ManagerRuleTables); err != nil {
		applogger.Errorf("经理同步规则表创建失败 (非阻断,留待下次启动): %v", err)
	}

	// 2) 索引兜底:model tag 未覆盖的索引面,显式 IF NOT EXISTS 幂等补建
	ddl := []string{
		// Phase 58 SC#1-SC#4 + Phase 60 AUTH-03 + SEC-01 都依赖此表
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON sys_api_keys(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON sys_api_keys(key_prefix)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON sys_api_keys(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_api_key_id ON sys_api_key_usage_logs(api_key_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_created_at ON sys_api_key_usage_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_user_id ON sys_api_key_usage_logs(user_id)`,
	}

	for i, stmt := range ddl {
		applogger.Infof("[BootstrapMissingTables] executing index %d/%d", i+1, len(ddl))
		if err := d.DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("DDL[%d] failed: %w", i+1, err)
		}
	}
	applogger.Infof("[BootstrapMissingTables] tables model-derived, %d index statements OK", len(ddl))
	return nil
}

// ensureADPushQueue 保证 ad_push_queue 表与三条索引就位（Phase 57 F-03）。
//
// 表 + 部分唯一索引 + 两个查询索引全部由 migration_215 建立（幂等 IF NOT EXISTS，可重跑）。
// AutoMigrate 路径与 SKIP_AUTOMIGRATE 旁路（BootstrapMissingTables）都经本函数，
// 保证两条启动路径的队列表结构完全一致 —— 旁路路径曾完全漏掉本表（F-03），
// 导致该部署形态下用户管理的新增/编辑/删除全部 500。
func (d *Database) ensureADPushQueue() error {
	if !d.DB.Migrator().HasTable(&models.ADPushQueue{}) {
		applogger.Infof("[ad_push_queue] 表缺失,经 migration_215 显式 DDL 补建(含 uq_ad_push_queue_active)")
	}
	// 阻断的是迁移本身的失败；版本记录（runAndRecord 内部）失败只 WARN 不阻断。
	if err := d.runAndRecord(215, "create_ad_push_queue", migrations.Migrate215CreateADPushQueue); err != nil {
		return fmt.Errorf("ad_push_queue 表/索引创建失败: %w", err)
	}
	return nil
}

// dbIdentRe 校验 PG 数据库标识符,防止 CREATE DATABASE 拼接时注入非法字符
var dbIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// migrationAdvisoryLockKey 是迁移块 advisory lock 的字符串锁键,
// 跨进程共享同一哈希值(hash key 在 pg_advisory_lock(int8, int4) 双参版本下用 int4)。
const migrationAdvisoryLockKey = "xingran-migrations"

// isDuplicateDatabaseError 判定 err 是否为 PG 42P04 (duplicate_database)。
// errors.As 解包 *pq.Error 以覆盖 fmt.Errorf("...: %w", pqErr) 包装路径。
//
// CDX-H1 修复:并发 bootstrap 撞出的 42P04 容忍为 WARN(不致命)—— 语义就是"已存在",
// 后续 gorm.Open 会真实验证连通性。理论上一个真正失败的 CREATE DATABASE 若恰好报 42P04
// 会被 WARN 放过,但风险可接受(42P04 语义唯一即"已存在")。
func isDuplicateDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "42P04"
	}
	return false
}

// createDatabaseIfNotExists 如果数据库不存在则创建。
//
// CDX-H1 修复:
//   - 错误真实上抛,启动 fail-fast 暴露根因(认证/网络/缺 postgres 维护库)
//   - 容忍 42P04 (duplicate_database) 作为 WARN —— 并发 bootstrap 时另一实例已 CREATE 完毕
//   - 存在性检查加 10s 超时(opencode #10),防 admin DB 不可达导致启动挂死
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

	// 存在性检查加超时,防止 admin PG 不可达时启动挂死(opencode suggestion #10)
	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping 管理员数据库失败: %w", err)
	}

	// 检查数据库是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)"
	err = db.QueryRowContext(pingCtx, query, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("检查数据库是否存在失败: %w", err)
	}

	// 如果数据库不存在，则创建
	if !exists {
		createQuery := fmt.Sprintf("CREATE DATABASE %s", pq.QuoteIdentifier(dbName))
		_, err = db.ExecContext(pingCtx, createQuery)
		if err != nil {
			// CDX-H1:42P04 duplicate_database 容忍为 WARN,其他错误上抛。
			if isDuplicateDatabaseError(err) {
				applogger.Warnf("数据库 %s 已被并发实例创建,继续", dbName)
				return nil
			}
			return fmt.Errorf("创建数据库失败: %w", err)
		}
		applogger.Infof("数据库 %s 创建成功", dbName)
	} else {
		applogger.Infof("数据库 %s 已存在", dbName)
	}

	return nil
}
