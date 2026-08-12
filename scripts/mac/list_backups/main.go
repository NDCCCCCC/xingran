// Package main mac_list_backups 一次性列出所有 sys_device_mac_history 备份表(行数+大小+创建时间),
// 帮助 DROP 前确认目标。只读不写。
package main

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load(context.Background())
	if err != nil {
		applogger.Errorf("[mac_list_backups] 配置加载失败: %v", err)
		os.Exit(1)
	}
	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[list] DB 连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, _ := dbConn.GetDB().DB()
		_ = sqlDB.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type Row struct {
		TableName string
		Size      string
	}
	var rows []Row
	dbConn.GetDB().WithContext(ctx).Raw(`
		SELECT
		  c.relname AS table_name,
		  pg_size_pretty(pg_total_relation_size(c.oid)) AS size
		  FROM pg_class c
		  JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = 'public'
		   AND c.relname LIKE 'sys_device_mac_history%backup%'
		 ORDER BY c.relname DESC
	`).Scan(&rows)

	if len(rows) == 0 {
		applogger.Info("[list] 没有任何 mac_history 备份表")
		return
	}
	applogger.Infof("[list] 共 %d 个备份表:", len(rows))
	for _, r := range rows {
		var cnt int64
		dbConn.GetDB().WithContext(ctx).Table(r.TableName).Count(&cnt)
		applogger.Infof("[list]  %-60s  %8s  %d 行", r.TableName, r.Size, cnt)
	}
}
