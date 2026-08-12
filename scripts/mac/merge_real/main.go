// Package main mac_merge_real 是独立的运维 CLI 工具,用于按"状态转换点"
// 合并 sys_device_mac_history 中的 flapping 记录。
//
// 背景(2026-07-01):
//   - 用户原话:"仅保留设备或接口有变化的记录,删除其余的所有记录"
//   - 修复 RecordMACChange 的 L2 过度优化后,稳定 MAC 7 天会写入 ~67 条
//     真实事件(appeared/disappeared flapping)
//   - 一次性把历史/未来 flapping 折叠成"状态转换点":同 (device_id, mac_address)
//     内,仅保留 interface_name/vlan_id 与上一保留记录不同的转换点
//
// 与现有工具对比:
//   - mac_cleanup / CleanupAllDevicesFlapping:按 2h 时间窗口合并,本工具按位置签名
//   - mac_purge_meaningless:保留 vlan_changed;本工具严格按用户原话,VLAN-only 变化删除
//
// 用法:
//
//	go run ./scripts/mac/merge_real                      # 真跑(自动备份)
//	go run ./scripts/mac/merge_real -dry-run             # 只统计,不写 DB
//	go run ./scripts/mac/merge_real -dry-run -verbose    # 详细统计
//
// 备份:
//
//	执行前自动创建 sys_device_mac_history_merge_real_backup_YYYYMMDD_HHMMSS,
//	包含所有当前行,可用于回滚。
//
// 回滚:
//
//	INSERT INTO sys_device_mac_history SELECT * FROM <backup_table>;
//	注意:备份表不含分区(普通表),需手工处理 first_seen 落对应分区。
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
	"github.com/xingran-next/xingran-go-backend/internal/services"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "只统计将删除的记录数,不实际写 DB")
	verbose := flag.Bool("verbose", false, "打印合并前后事件类型分布")
	timeout := flag.Duration("timeout", 30*time.Minute, "全量合并超时(默认 30 分钟)")
	flag.Parse()

	// 1. 加载 .env + config
	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[mac_merge_real] .env 未加载: %v", err)
	}
	cfg := config.Load()
	applogger.Info("=================================================================")
	applogger.Info("= mac_merge_real 启动")
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
		applogger.Errorf("[mac_merge_real] 数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := dbConn.GetDB().DB()
		if err != nil {
			applogger.Errorf("[mac_merge_real] 拿到底层 *sql.DB 失败: %v", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			applogger.Errorf("[mac_merge_real] 关闭 DB 连接失败: %v", err)
		}
	}()
	applogger.Infof("[mac_merge_real] 数据库连接成功")

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 3. 备份表名(时间戳避免重复跑冲突)
	backupTS := time.Now().Format("20060102_150405")
	backupTable := fmt.Sprintf("sys_device_mac_history_merge_real_backup_%s", backupTS)

	// 4. 总行数(用于校验)
	var totalCount int64
	if err := dbConn.GetDB().WithContext(ctx).
		Table("sys_device_mac_history").
		Count(&totalCount).Error; err != nil {
		applogger.Errorf("[mac_merge_real] 查总行数失败: %v", err)
		os.Exit(1)
	}
	applogger.Infof("[mac_merge_real] 当前总行数: %d", totalCount)

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
			applogger.Warnf("[mac_merge_real] event_type 分组统计失败(非阻断): %v", err)
		} else {
			applogger.Info("[mac_merge_real] event_type 分布(merge 前):")
			for _, c := range counts {
				applogger.Infof("  %-15s %d", c.EventType, c.Count)
			}
		}
	}

	// 6. dry-run 模式:只跑 MergeByTransitions 但需事务回滚(实际这里服务方法不暴露 dry-run,
	//    我们用 pg_rollback 模拟:启动事务 → 跑合并 → ROLLBACK)
	if *dryRun {
		applogger.Info("[mac_merge_real] DRY-RUN: 启动事务跑 MergeByTransitions,完成后 ROLLBACK")
		tx := dbConn.GetDB().WithContext(ctx).Begin()
		if tx.Error != nil {
			applogger.Errorf("[mac_merge_real] 开始事务失败: %v", tx.Error)
			os.Exit(1)
		}

		macSvc := services.NewMACHistoryService(tx)
		deleted, err := macSvc.MergeByTransitions(ctx)
		if err != nil {
			_ = tx.Rollback().Error
			applogger.Errorf("[mac_merge_real] DRY-RUN MergeByTransitions 失败: %v", err)
			os.Exit(1)
		}
		if err := tx.Rollback().Error; err != nil {
			applogger.Warnf("[mac_merge_real] DRY-RUN rollback 失败(非阻断): %v", err)
		}

		toKeepCount := totalCount - deleted
		applogger.Info("=================================================================")
		applogger.Infof("= [mac_merge_real] DRY-RUN 预览(已 ROLLBACK)")
		applogger.Infof("= 当前总行数:   %d", totalCount)
		applogger.Infof("= 预计保留:     %d 行(状态转换点)", toKeepCount)
		applogger.Infof("= 预计删除:     %d 行(flapping)", deleted)
		applogger.Infof("= 备份表(若执行): %s", backupTable)
		applogger.Info("=================================================================")
		return
	}

	// 7. 真跑模式:备份 → 合并
	if totalCount == 0 {
		applogger.Infof("[mac_merge_real] 表为空,无需合并")
		return
	}

	// 7.1 备份
	applogger.Infof("[mac_merge_real] 创建备份表: %s", backupTable)
	createBackupSQL := fmt.Sprintf(`
		CREATE TABLE %s AS
		SELECT * FROM sys_device_mac_history
	`, backupTable)
	if err := dbConn.GetDB().WithContext(ctx).Exec(createBackupSQL).Error; err != nil {
		applogger.Errorf("[mac_merge_real] 创建备份表失败: %v", err)
		os.Exit(1)
	}

	var backupCount int64
	if err := dbConn.GetDB().WithContext(ctx).
		Table(backupTable).
		Count(&backupCount).Error; err != nil {
		applogger.Warnf("[mac_merge_real] 校验备份表行数失败(非阻断): %v", err)
	} else {
		applogger.Infof("[mac_merge_real] 备份表行数: %d (期望 %d)", backupCount, totalCount)
	}

	// 7.2 合并
	applogger.Infof("[mac_merge_real] 开始按状态转换点合并...")
	start := time.Now()
	macSvc := services.NewMACHistoryService(dbConn.GetDB())
	deleted, err := macSvc.MergeByTransitions(ctx)
	elapsed := time.Since(start)

	if err != nil {
		applogger.Errorf("[mac_merge_real] 合并失败: %v", err)
		applogger.Infof("[mac_merge_real] 回滚指引: INSERT INTO sys_device_mac_history SELECT * FROM %s", backupTable)
		os.Exit(1)
	}

	// 7.3 后置校验
	var finalCount int64
	if err := dbConn.GetDB().WithContext(ctx).
		Table("sys_device_mac_history").
		Count(&finalCount).Error; err != nil {
		applogger.Errorf("[mac_merge_real] 校验最终行数失败: %v", err)
		os.Exit(1)
	}

	applogger.Info("=================================================================")
	applogger.Infof("= [mac_merge_real] 合并完成")
	applogger.Infof("= 合并前行数: %d", totalCount)
	applogger.Infof("= 合并后行数: %d", finalCount)
	applogger.Infof("= 实际删除:   %d 行(预期 %d)", totalCount-finalCount, deleted)
	applogger.Infof("= 备份表:     %s (%d 行)", backupTable, backupCount)
	applogger.Infof("= 耗时:       %s", elapsed)
	applogger.Info("=================================================================")
	applogger.Infof("[mac_merge_real] 回滚指引(如需): INSERT INTO sys_device_mac_history SELECT * FROM %s", backupTable)

	if totalCount-finalCount != deleted {
		applogger.Warnf("[mac_merge_real] 警告: 实际删除数 (%d) 与预期 (%d) 不一致,需人工核对",
			totalCount-finalCount, deleted)
	}

	// 7.4 合并后 event_type 分布
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
			Scan(&counts).Error; err == nil {
			applogger.Info("[mac_merge_real] event_type 分布(merge 后):")
			for _, c := range counts {
				applogger.Infof("  %-15s %d", c.EventType, c.Count)
			}
		}
	}
}
