// Command mac_purge_drop_backup 是独立 CLI,用于 DROP 旧的 purge 备份表。
//
// 用法:
//
//	go run ./scripts/mac/purge_drop_backup -table sys_device_mac_history_purge_backup_20260630_172543
//	go run ./scripts/mac/purge_drop_backup -auto-older-than 7d   # 自动 DROP >7 天的
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

func main() {
	table := flag.String("table", "", "要 DROP 的备份表名(完整)")
	autoOlderThan := flag.Duration("auto-older-than", 0, "自动 DROP 超过指定时长的备份表(例如 168h = 7d)")
	dryRun := flag.Bool("dry-run", false, "只列出要 DROP 的表,不实际删除")
	flag.Parse()

	if *table == "" && *autoOlderThan == 0 {
		fmt.Fprintln(os.Stderr, "用法: 提供 -table <name> 或 -auto-older-than <duration>")
		os.Exit(2)
	}

	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[drop_backup] .env 未加载: %v", err)
	}
	cfg, err := config.Load(context.Background())
	if err != nil {
		applogger.Errorf("[mac_purge_drop_backup] 配置加载失败: %v", err)
		os.Exit(1)
	}
	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[drop_backup] DB 连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := dbConn.GetDB().DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if *table != "" {
		dropOne(ctx, dbConn, *table, *dryRun)
		return
	}

	// auto 模式:DROP >autoOlderThan 的备份表
	cutoff := time.Now().Add(-*autoOlderThan)
	pattern := regexp.MustCompile(`^sys_device_mac_history_purge_backup_(\d{8})_(\d{6})$`)
	var names []string
	if err := dbConn.GetDB().WithContext(ctx).Raw(`
		SELECT table_name FROM information_schema.tables
		 WHERE table_schema = current_schema()
		   AND table_name LIKE 'sys_device_mac_history_purge_backup_%'
	`).Scan(&names).Error; err != nil {
		applogger.Errorf("[drop_backup] 查备份表失败: %v", err)
		os.Exit(1)
	}

	applogger.Infof("[drop_backup] auto 模式: cutoff=%s, 共 %d 个备份表", cutoff.Format("2006-01-02 15:04:05"), len(names))

	dropped := 0
	for _, name := range names {
		m := pattern.FindStringSubmatch(name)
		if m == nil {
			applogger.Debugf("[drop_backup] 跳过非标准表名: %s", name)
			continue
		}
		ts, perr := time.ParseInLocation("20060102_150405", m[1]+"_"+m[2], time.Local)
		if perr != nil {
			applogger.Warnf("[drop_backup] 时间戳解析失败 %s: %v", name, perr)
			continue
		}
		if ts.Before(cutoff) {
			if *dryRun {
				applogger.Infof("[drop_backup] [DRY-RUN] 会 DROP: %s (创建于 %s)", name, ts.Format("2006-01-02 15:04:05"))
			} else {
				dropOne(ctx, dbConn, name, false)
				dropped++
			}
		} else {
			applogger.Debugf("[drop_backup] 保留: %s (创建于 %s)", name, ts.Format("2006-01-02 15:04:05"))
		}
	}
	applogger.Infof("[drop_backup] 完成: %d 个表被 DROP", dropped)
}

// dropOne DROP 单个备份表
func dropOne(ctx context.Context, dbConn *db.Database, table string, dryRun bool) {
	if dryRun {
		applogger.Infof("[drop_backup] [DRY-RUN] 会 DROP: %s", table)
		return
	}
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
	if err := dbConn.GetDB().WithContext(ctx).Exec(sql).Error; err != nil {
		applogger.Errorf("[drop_backup] DROP %s 失败: %v", table, err)
		os.Exit(1)
	}
	applogger.Infof("[drop_backup] 已 DROP: %s", table)
}
