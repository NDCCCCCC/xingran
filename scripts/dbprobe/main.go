// dbprobe: 针对 debug session backend-hang-on-automigrate 的远端 dev DB 补建/诊断工具。
//
// 背景: SKIP_AUTOMIGRATE=true 旁路后, 两类对象在该 dev Supabase 库中缺失 (启动 WARN):
//   1. sys_device_mac_history 父表 (PARTITION BY RANGE first_seen) 未建 → 月度分区创建失败 (42P01)。
//      DeviceMACHistory 既不在 MigrateModelList 也不在 dbprovision extraModels; 原始建表
//      迁移已归档丢失。此处按 model 定义 + 分区键重建 (CREATE IF NOT EXISTS, 幂等, 表空无 corruption 风险)。
//   2. reconciliation_physical_chain (VIEW, 175) / reconciliation_normalized (MV, 176) 未建 →
//      Phase 42 启动 RefreshView + cron "对账-物化视图刷新" 报 42P01。直接调用 migration_175/176。
//   3. APIKey unique 约束归一化: 验证 cleanupOldConstraints 修复 (sys_api_keys 命名冲突 42704)。
//
// 连接: 复用 internal/config.Load + DatabaseConfig.GetDSN() —— 自动带上本次修复的
// connect_timeout + keepalive。每条操作带 60s ctx 超时 + 最多 5 次重试 (吸收链路随机 stall)。
//
// 用法 (项目根目录):
//
//	go run ./scripts/dbprobe
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// macHistoryParentDDL 重建 sys_device_mac_history 父表。
// 列定义与 internal/models/device_mac_history.go 1:1; first_seen 为分区键
// (timestamp without time zone, 见 model 注释 PG 禁止 ALTER 分区键列类型)。
// 分区表的 PRIMARY KEY 必须包含分区键 → PK (id, first_seen)。
const macHistoryParentDDL = `
CREATE TABLE IF NOT EXISTS sys_device_mac_history (
    id                   uuid                     NOT NULL,
    device_id            uuid                     NOT NULL,
    device_name_snapshot varchar(100),
    mac_address          varchar(30)              NOT NULL,
    interface_name       varchar(100)             NOT NULL,
    vlan_id              integer,
    event_type           varchar(20)              NOT NULL,
    first_seen           timestamp without time zone NOT NULL,
    last_seen            timestamp without time zone NOT NULL,
    collected_at         timestamp without time zone NOT NULL,
    created_at           timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, first_seen)
) PARTITION BY RANGE (first_seen);
`

// 175/176 依赖的基础表 (任一缺失则 reconciliation 视图/MV 无法创建)。
var reconDeps = []string{
	"ops_asset", "sys_device_mac_address", "sys_device_port_status",
	"ops_info_points", "sys_workstation", "sys_user", "sys_dept",
}

type conRow struct {
	Name string `gorm:"column:conname"`
}

func main() {
	_ = godotenv.Load() // 加载 .env (与 cmd/main.go 一致)
	cfg, err := config.Load(context.Background())
	if err != nil {
		die("加载配置失败: %v", err)
	}

	gdb, err := gorm.Open(postgres.Open(cfg.Database.GetDSN()), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              false,
	})
	if err != nil {
		die("连接 PG 失败: %v", err)
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(1)
	fmt.Printf("connected (host=%s)\n", cfg.Database.Host)

	// === 1. 盘点 ===
	fmt.Println("\n=== INVENTORY ===")
	has := func(name string) bool {
		var ok bool
		_ = withRetry(gdb, func(tx *gorm.DB) error {
			return tx.Raw("SELECT to_regclass(?) IS NOT NULL", name).Scan(&ok).Error
		})
		return ok
	}
	fmt.Printf("sys_device_mac_history (parent): %v\n", has("sys_device_mac_history"))
	missingDeps := []string{}
	for _, t := range reconDeps {
		ex := has(t)
		fmt.Printf("recon dep %-26s: %v\n", t, ex)
		if !ex {
			missingDeps = append(missingDeps, t)
		}
	}
	fmt.Printf("reconciliation_physical_chain (view): %v\n", has("reconciliation_physical_chain"))
	fmt.Printf("reconciliation_normalized (mv):      %v\n", has("reconciliation_normalized"))

	// === 2. MAC 历史父表 + 分区 ===
	fmt.Println("\n=== MAC HISTORY PARENT ===")
	if err := withRetryExec(gdb, macHistoryParentDDL); err != nil {
		fmt.Printf("!! 创建 sys_device_mac_history 父表失败: %v\n", err)
	} else {
		fmt.Println("ok: sys_device_mac_history 父表就位")
		now := time.Now()
		for i := 0; i <= 2; i++ {
			td := now.AddDate(0, i, 0)
			y, m := td.Year(), int(td.Month())
			ey, em := y, m+1
			if em > 12 {
				ey, em = ey+1, 1
			}
			pname := fmt.Sprintf("sys_device_mac_history_%d_%02d", y, m)
			pddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s PARTITION OF sys_device_mac_history FOR VALUES FROM ('%d-%02d-01') TO ('%d-%02d-01')",
				pname, y, m, ey, em)
			if err := withRetryExec(gdb, pddl); err != nil {
				fmt.Printf("!! 分区 %s 失败: %v\n", pname, err)
			} else {
				fmt.Printf("ok: 分区 %s\n", pname)
			}
		}
	}

	// === 3. reconciliation 175/176 ===
	fmt.Println("\n=== RECONCILIATION 175/176 ===")
	if len(missingDeps) > 0 {
		fmt.Printf("!! 跳过 175/176: 缺失基础表 %v (先跑 scripts/dbprovision 补建基础表)\n", missingDeps)
	} else {
		if err := withRetryErr(gdb, migrations.Migrate175ReconciliationPhysicalLink); err != nil {
			fmt.Printf("!! Migrate175 失败: %v\n", err)
		} else {
			fmt.Println("ok: Migrate175 (reconciliation_physical_chain + user_lookup + ops_asset_physical)")
		}
		if err := withRetryErr(gdb, migrations.Migrate176ReconciliationPhysicalMV); err != nil {
			fmt.Printf("!! Migrate176 失败: %v\n", err)
		} else {
			fmt.Println("ok: Migrate176 (reconciliation_normalized MV)")
		}
	}

	// === 3.5 APIKey unique 约束归一化 ===
	// 验证 cleanupOldConstraints 修复 (database.go): sys_api_keys.key_hash 命名冲突。
	// 现状: BootstrapMissingTables 用 inline UNIQUE → PG 自动名 sys_api_keys_key_hash_key;
	//      models.APIKey.KeyHash 标 uniqueIndex → GORM 期望 uni_sys_api_keys_key_hash。
	// 不带 bypass 启动时 AutoMigrate DROP CONSTRAINT uni_...(无 IF EXISTS) → 42704 FATA。
	// 本段先列出现状, 模拟 cleanupOldConstraints 清理两命名, 隔离跑 AutoMigrate(APIKey) 验证不再 42704。
	fmt.Println("\n=== APIKEY CONSTRAINT NORMALIZE ===")
	if !has("sys_api_keys") {
		fmt.Println("(sys_api_keys 表不存在, 跳过)")
	} else {
		listCons := func() []conRow {
			var rows []conRow
			_ = withRetry(gdb, func(tx *gorm.DB) error {
				return tx.Raw("SELECT conname FROM pg_constraint WHERE conrelid='sys_api_keys'::regclass AND contype='u'").Scan(&rows).Error
			})
			return rows
		}
		fmt.Printf("current unique constraints: %v\n", listCons())
		// 模拟 cleanupOldConstraints (与启动期一致): 两命名都 DROP IF EXISTS
		for _, cname := range []string{"uni_sys_api_keys_key_hash", "sys_api_keys_key_hash_key"} {
			_ = withRetryExec(gdb, fmt.Sprintf("ALTER TABLE sys_api_keys DROP CONSTRAINT IF EXISTS %s", cname))
		}
		// 隔离跑 APIKey AutoMigrate (PrepareStmt=false, 与启动期同款)
		if err := withRetry(gdb, func(tx *gorm.DB) error { return tx.Migrator().AutoMigrate(&models.APIKey{}) }); err != nil {
			fmt.Printf("!! AutoMigrate(APIKey) 失败: %v\n", err)
		} else {
			fmt.Println("ok: AutoMigrate(APIKey) 成功 (无 42704)")
		}
		fmt.Printf("post unique constraints: %v\n", listCons())
	}

	// === 4. 复核 ===
	fmt.Println("\n=== POST-CHECK ===")
	fmt.Printf("sys_device_mac_history (parent): %v\n", has("sys_device_mac_history"))
	fmt.Printf("reconciliation_physical_chain (view): %v\n", has("reconciliation_physical_chain"))
	fmt.Printf("reconciliation_normalized (mv):      %v\n", has("reconciliation_normalized"))
	fmt.Println("\ndone.")
}

// === retry helpers (60s ctx, 最多 5 次, 吸收 keepalive 判死后的连接回收) ===

func withRetry(gdb *gorm.DB, fn func(tx *gorm.DB) error) error {
	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		err := fn(gdb.WithContext(ctx))
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		fmt.Printf("  retry %d/5: %v\n", attempt, err)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
	return lastErr
}

func withRetryExec(gdb *gorm.DB, sql string) error {
	return withRetry(gdb, func(tx *gorm.DB) error { return tx.Exec(sql).Error })
}

func withRetryErr(gdb *gorm.DB, fn func(*gorm.DB) error) error {
	return withRetry(gdb, fn)
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
