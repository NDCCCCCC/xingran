// Package main mac_cleanup 是独立的运维 CLI 工具,用于全量清理 MAC 历史
// 表(sys_device_mac_history_*)的 flapping 数据。
//
// 背景(2026-06-30):
//   - collector 每小时采集一次,设备 MAC 表老化时间短(~5min)导致同一 MAC
//     在 1h 间隔内出现 disappeared → appeared 抖动序列
//   - MergeFlappingRecords 算法放宽(2h 窗口 + 允许交替事件类型)后,可把
//     抖动序列合并成单条"持续存在"区间,前端轨迹查询更干净
//   - 该工具对所有有历史的 deviceID 一次性跑 cleanup
//
// 用法:
//
//	go run ./scripts/mac/cleanup                     # 使用默认 config.yaml + .env
//	go run ./scripts/mac/cleanup -dry-run            # 只统计,不写 DB
//	CONFIG_PATH=/path/to/config.yaml go run ./scripts/mac/cleanup
//
// 行为:
//   - 加载 .env(同主程序 godotenv 模式)
//   - 加载 config/config.yaml (env 覆盖)
//   - 仅连 DB,**不**跑 AutoMigrate / InitData / cache / scheduler
//   - 构造 MACHistoryService,调 CleanupAllDevicesFlapping(ctx)
//   - 输出进度到日志(每 50 个设备 1 行)
//   - 全量完成后退出码 0;遇错则退出码 1 并打印首个错误
//
// 回滚:
//
//	merge 后的数据无法自动回滚 — 原始 flapping 行已被 DELETE。
//	建议:跑 cleanup 前先在 DB 备份对应分区的快照,或确认已 flapping 序列
//	对业务确实无意义再跑。
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "只统计会处理的设备数,不实际写 DB")
	timeout := flag.Duration("timeout", 30*time.Minute, "全量清理超时(默认 30 分钟)")
	flag.Parse()

	// 1. 加载 .env(主程序同款 godotenv 模式)
	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[mac_cleanup] .env 未加载(可用环境变量注入): %v", err)
	}

	// 2. 加载配置(config/config.yaml + env override)
	cfg, err := config.Load(context.Background())
	if err != nil {
		applogger.Errorf("[mac_cleanup] 配置加载失败: %v", err)
		os.Exit(1)
	}
	applogger.Info("=================================================================")
	applogger.Info("= mac_cleanup 启动")
	if cfg.Database.Host != "" {
		applogger.Infof("= 数据库: PostgreSQL @ %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	} else {
		applogger.Info("= 数据库: SQLite(本地)")
	}
	applogger.Infof("= dry-run: %v", *dryRun)
	applogger.Infof("= timeout: %s", *timeout)
	applogger.Info("=================================================================")

	// 3. 仅连 DB,**不**跑 AutoMigrate / InitData(CLI 不应改 schema)
	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[mac_cleanup] 数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := dbConn.GetDB().DB()
		if err != nil {
			applogger.Errorf("[mac_cleanup] 拿到底层 *sql.DB 失败: %v", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			applogger.Errorf("[mac_cleanup] 关闭 DB 连接失败: %v", err)
		}
	}()

	applogger.Infof("[mac_cleanup] 数据库连接成功")

	// 4. dry-run 模式:只统计设备数,跳过实际 cleanup
	if *dryRun {
		var deviceIDs []string
		if err := dbConn.GetDB().
			Table("sys_device_mac_history").
			Distinct("device_id").
			Pluck("device_id", &deviceIDs).Error; err != nil {
			applogger.Errorf("[mac_cleanup] dry-run 查 device_id 失败: %v", err)
			os.Exit(1)
		}
		applogger.Infof("[mac_cleanup] DRY-RUN: 会处理 %d 个设备(不实际改 DB)", len(deviceIDs))
		return
	}

	// 5. 构造 service,跑全量 cleanup
	macSvc := services.NewMACHistoryService(dbConn.GetDB())
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	start := time.Now()
	count, err := macSvc.CleanupAllDevicesFlapping(ctx)
	elapsed := time.Since(start)

	if err != nil {
		applogger.Error("=================================================================")
		applogger.Errorf("= [mac_cleanup] 全量清理失败: %v", err)
		applogger.Errorf("= 已成功处理设备数: %d", count)
		applogger.Errorf("= 耗时: %s", elapsed)
		applogger.Error("=================================================================")
		os.Exit(1)
	}

	applogger.Info("=================================================================")
	applogger.Infof("= [mac_cleanup] 全量清理完成: %d 个设备, 耗时 %s", count, elapsed)
	applogger.Info("=================================================================")
}
