// Package main mac_purge_meaningless 是独立的运维 CLI 工具,用于清理
// sys_device_mac_history 表中的"无意义记录",仅保留真实的终端移动事件。
//
// 背景(2026-06-30):
//   - 早期 collector 漏采 1 个周期 + L1/L2 去重缺失,产生大量冗余
//     appeared/disappeared 配对(同一 MAC 7 天 ≈ 366 条)
//   - 真实业务只关心"终端物理移动"(MAC 出现在不同交换机端口 = moved 事件)
//     或"网络位置变化"(vlan_changed 事件)
//   - 纯 appeared / disappeared 大多数是 collector 噪声,无审计价值
//
// 保留策略(用户决策 2026-06-30):
//   - KEEP: 所有 moved 事件
//   - KEEP: 所有 vlan_changed 事件
//   - KEEP: 每个 (device_id, mac_address) 第一条 appeared(first_seen ASC)
//   - DELETE: 其他 appeared / 所有 disappeared
//
// 用法:
//
//	go run ./scripts/mac/purge_meaningless -dry-run           # 只统计,不写 DB
//	go run ./scripts/mac/purge_meaningless                   # 真跑(自动备份)
//	go run ./scripts/mac/purge_meaningless -dry-run -verbose # 详细统计
//
// 备份:
//
//	执行前自动创建 sys_device_mac_history_purge_backup_YYYYMMDD_HHMMSS,
//	包含所有当前行,可用于回滚。
//
// 回滚:
//
//	pg_restore / 手工 INSERT INTO sys_device_mac_history SELECT * FROM backup
//	注意备份表不含分区(普通表),需手工处理 first_seen 落对应分区。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "只统计将删除的记录数,不实际写 DB")
	verbose := flag.Bool("verbose", false, "打印每个 event_type 的保留/删除统计")
	timeout := flag.Duration("timeout", 30*time.Minute, "全量清理超时(默认 30 分钟)")
	flag.Parse()

	// 1. 加载 .env + config
	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[mac_purge] .env 未加载: %v", err)
	}
	cfg, err := config.Load(context.Background())
	if err != nil {
		applogger.Errorf("[mac_purge_meaningless] 配置加载失败: %v", err)
		os.Exit(1)
	}
	applogger.Info("=================================================================")
	applogger.Info("= mac_purge_meaningless 启动")
	if cfg.Database.Host != "" {
		applogger.Infof("= 数据库: PostgreSQL @ %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	} else {
		applogger.Info("= 数据库: SQLite(本地)")
	}
	applogger.Infof("= dry-run: %v", *dryRun)
	applogger.Infof("= verbose: %v", *verbose)
	applogger.Infof("= timeout: %s", *timeout)
	applogger.Info("=================================================================")

	// 2. 仅连 DB(CLI 不应改 schema / 启动 cache / scheduler)
	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[mac_purge] 数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := dbConn.GetDB().DB()
		if err != nil {
			applogger.Errorf("[mac_purge] 拿到底层 *sql.DB 失败: %v", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			applogger.Errorf("[mac_purge] 关闭 DB 连接失败: %v", err)
		}
	}()
	applogger.Infof("[mac_purge] 数据库连接成功")

	// 3. 备份表名(时间戳避免重复跑冲突)
	backupTS := time.Now().Format("20060102_150405")
	backupTable := fmt.Sprintf("sys_device_mac_history_purge_backup_%s", backupTS)

	// 4. 总行数(用于校验)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	var totalCount int64
	if err := dbConn.GetDB().WithContext(ctx).
		Table("sys_device_mac_history").
		Count(&totalCount).Error; err != nil {
		applogger.Errorf("[mac_purge] 查总行数失败: %v", err)
		os.Exit(1)
	}
	applogger.Infof("[mac_purge] 当前总行数: %d", totalCount)

	// 5. 各 event_type 行数(详细模式)
	if *verbose {
		type EventCount struct {
			EventType string
			Count     int64
		}
		var counts []EventCount
		if err := dbConn.GetDB().WithContext(ctx).
			Table("sys_device_mac_history").
			Select("event_type, COUNT(*) AS count").
			Group("event_type").
			Scan(&counts).Error; err != nil {
			applogger.Warnf("[mac_purge] event_type 分组统计失败(非阻断): %v", err)
		} else {
			applogger.Info("[mac_purge] event_type 分布:")
			for _, c := range counts {
				applogger.Infof("  %-15s %d", c.EventType, c.Count)
			}
		}
	}

	// 6. 计算将被删除的行数(无论 dry-run 还是真跑都先统计,便于核对)
	// 删除条件 = 所有 disappeared + 除每个 (device_id, mac_address) 第一条 appeared 外的其他 appeared
	// 保留 = moved + vlan_changed + 每个 (device_id, mac_address) 第一条 appeared
	var toDeleteCount int64
	deleteSQL := `
		WITH first_appeared AS (
		    SELECT DISTINCT ON (device_id, mac_address)
		           id
		      FROM sys_device_mac_history
		     WHERE event_type = 'appeared'
		     ORDER BY device_id, mac_address, first_seen ASC
		)
		SELECT COUNT(*)
		  FROM sys_device_mac_history
		 WHERE (event_type = 'disappeared')
		    OR (event_type = 'appeared' AND id NOT IN (SELECT id FROM first_appeared))
	`
	if err := dbConn.GetDB().WithContext(ctx).Raw(deleteSQL).Scan(&toDeleteCount).Error; err != nil {
		applogger.Errorf("[mac_purge] 计算删除行数失败: %v", err)
		os.Exit(1)
	}
	toKeepCount := totalCount - toDeleteCount

	applogger.Info("=================================================================")
	applogger.Infof("= [mac_purge] 清理预览")
	applogger.Infof("= 当前总行数: %d", totalCount)
	applogger.Infof("= 预计保留:   %d 行(moved + vlan_changed + 每 MAC 首条 appeared)", toKeepCount)
	applogger.Infof("= 预计删除:   %d 行(disappeared 全删 + appeared 冗余)", toDeleteCount)
	applogger.Info("=================================================================")

	if *dryRun {
		applogger.Infof("[mac_purge] DRY-RUN: 不会修改 DB,备份表名(若执行)将为 %s", backupTable)
		return
	}

	if toDeleteCount == 0 {
		applogger.Infof("[mac_purge] 无需删除,退出")
		return
	}

	// 7. 创建备份表(普通表,非分区)
	applogger.Infof("[mac_purge] 创建备份表: %s", backupTable)
	createBackupSQL := fmt.Sprintf(`
		CREATE TABLE %s AS
		SELECT * FROM sys_device_mac_history
	`, backupTable)
	if err := dbConn.GetDB().WithContext(ctx).Exec(createBackupSQL).Error; err != nil {
		applogger.Errorf("[mac_purge] 创建备份表失败: %v", err)
		os.Exit(1)
	}

	var backupCount int64
	if err := dbConn.GetDB().WithContext(ctx).
		Table(backupTable).
		Count(&backupCount).Error; err != nil {
		applogger.Warnf("[mac_purge] 校验备份表行数失败(非阻断): %v", err)
	} else {
		applogger.Infof("[mac_purge] 备份表行数: %d (期望 %d)", backupCount, totalCount)
	}

	// 8. 事务内执行删除
	applogger.Infof("[mac_purge] 开始删除 %d 行... ", toDeleteCount)
	start := time.Now()
	err = dbConn.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		execSQL := `
			WITH first_appeared AS (
			    SELECT DISTINCT ON (device_id, mac_address)
			           id
			      FROM sys_device_mac_history
			     WHERE event_type = 'appeared'
			     ORDER BY device_id, mac_address, first_seen ASC
			)
			DELETE FROM sys_device_mac_history
			 WHERE (event_type = 'disappeared')
			    OR (event_type = 'appeared' AND id NOT IN (SELECT id FROM first_appeared))
		`
		result := tx.Exec(execSQL)
		if result.Error != nil {
			return fmt.Errorf("DELETE 失败: %w", result.Error)
		}
		applogger.Infof("[mac_purge] DELETE 完成: %d 行受影响", result.RowsAffected)
		return nil
	})
	if err != nil {
		applogger.Errorf("[mac_purge] 删除失败: %v", err)
		applogger.Infof("[mac_purge] 回滚指引: INSERT INTO sys_device_mac_history SELECT * FROM %s", backupTable)
		os.Exit(1)
	}

	// 9. 后置校验
	var finalCount int64
	if err := dbConn.GetDB().WithContext(ctx).
		Table("sys_device_mac_history").
		Count(&finalCount).Error; err != nil {
		applogger.Errorf("[mac_purge] 校验最终行数失败: %v", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	applogger.Info("=================================================================")
	applogger.Infof("= [mac_purge] 清理完成")
	applogger.Infof("= 删除前行数: %d", totalCount)
	applogger.Infof("= 删除后行数: %d", finalCount)
	applogger.Infof("= 实际删除:   %d 行(预期 %d)", totalCount-finalCount, toDeleteCount)
	applogger.Infof("= 备份表:     %s (%d 行)", backupTable, backupCount)
	applogger.Infof("= 耗时:       %s", elapsed)
	applogger.Info("=================================================================")
	applogger.Infof("[mac_purge] 回滚指引(如需): INSERT INTO sys_device_mac_history SELECT * FROM %s", backupTable)

	if totalCount-finalCount != toDeleteCount {
		applogger.Warnf("[mac_purge] 警告: 实际删除数 (%d) 与预期 (%d) 不一致,需人工核对",
			totalCount-finalCount, toDeleteCount)
	}
}
