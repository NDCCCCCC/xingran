// Package main cleanup_port_status 清理 sys_device_port_status 中 "不合理的接口名称" 行。
//
// 删除判定:interface_name 全部由数字 + 斜杠组成(典型异常:纯数字 "5"、X/Y "3/11"、以斜杠开头 "/47")。
//
// 用法:
//
//	go run ./scripts/port/cleanup_status                # 预演(只 SELECT,不 DELETE)
//	go run ./scripts/port/cleanup_status --confirm      # 实际执行 DELETE
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

func main() {
	confirm := flag.Bool("confirm", false, "实际执行 DELETE;缺省为只读预演")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[cleanup] .env 未加载: %v", err)
	}
	cfg, err := config.Load(context.Background())
	if err != nil {
		applogger.Errorf("[cleanup_status] 配置加载失败: %v", err)
		os.Exit(1)
	}

	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[cleanup] 数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, _ := dbConn.GetDB().DB()
		_ = sqlDB.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	g := dbConn.GetDB().WithContext(ctx)

	applogger.Info("=================================================================")
	applogger.Info("= cleanup_port_status 启动")
	applogger.Infof("= 数据库: PostgreSQL @ %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	applogger.Info("=================================================================")

	// === Section 1: 总体命中数(按 interface_name 形态分组) ===
	applogger.Info("-----------------------------------------------------------------")
	applogger.Info("= Section 1: 命中总数 + 按形态分组")
	applogger.Info("-----------------------------------------------------------------")

	type ShapeRow struct {
		Shape string
		Cnt   int64
	}
	var shapes []ShapeRow
	g.Raw(`
		SELECT
		  CASE
		    WHEN interface_name ~ '^[0-9]+$'                THEN '纯数字 (5)'
		    WHEN interface_name ~ '^/[0-9]+$'               THEN '以斜杠开头 (/47)'
		    WHEN interface_name ~ '^[0-9]+/[0-9]+$'         THEN 'X/Y 短格式 (3/11)'
		    ELSE '其他'
		  END AS shape,
		  COUNT(*) AS cnt
		  FROM sys_device_port_status
		 WHERE interface_name ~ '^([0-9]+/)?[0-9]+$|^/[0-9]+$'
		 GROUP BY 1
		 ORDER BY cnt DESC
	`).Scan(&shapes)

	var total int64
	for _, s := range shapes {
		total += s.Cnt
		applogger.Infof("[cleanup] %-30s %6d 行", s.Shape, s.Cnt)
	}
	applogger.Infof("[cleanup] 总命中行数 = %d (matched),其他命名格式不会删", total)

	if total == 0 {
		applogger.Info("[cleanup] 无命中数据,流程结束")
		return
	}

	// === Section 2: 受影响设备数 + 采样(显示前 10 条) ===
	applogger.Info("-----------------------------------------------------------------")
	applogger.Info("= Section 2: 受影响范围 + 采样 (前 10 条)")
	applogger.Info("-----------------------------------------------------------------")

	var distinctDevs int64
	g.Raw(`SELECT COUNT(DISTINCT device_id) FROM sys_device_port_status WHERE interface_name ~ '^([0-9]+/)?[0-9]+$|^/[0-9]+$'`).Scan(&distinctDevs)
	applogger.Infof("[cleanup] 受影响设备数 = %d", distinctDevs)

	type SampleRow struct {
		ID            string
		DeviceID      string
		InterfaceName string
		AdminStatus   *string
		OperStatus    *string
		CollectedAt   time.Time
		CreatedAt     time.Time
	}
	var samples []SampleRow
	g.Raw(`
		SELECT id, device_id, interface_name, admin_status, oper_status, collected_at, created_at
		  FROM sys_device_port_status
		 WHERE interface_name ~ '^([0-9]+/)?[0-9]+$|^/[0-9]+$'
		 ORDER BY device_id, interface_name
		 LIMIT 10
	`).Scan(&samples)
	applogger.Infof("[cleanup] %-40s %-40s %-15s %-12s %-12s %-25s",
		"id", "device_id", "iface", "admin", "oper", "collected_at")
	for _, s := range samples {
		applogger.Infof("[cleanup] %-40s %-40s %-15s %-12s %-12s %-25s",
			s.ID, s.DeviceID, s.InterfaceName,
			nullableStr(s.AdminStatus), nullableStr(s.OperStatus),
			s.CollectedAt.Format("2006-01-02 15:04:05"))
	}

	// === Section 3: 决定是否 DELETE ===
	if !*confirm {
		applogger.Info("-----------------------------------------------------------------")
		applogger.Info("[cleanup] 预演模式(--confirm 未传),未执行 DELETE")
		applogger.Info("[cleanup] 如确认无误,执行: go run ./scripts/port/cleanup_status --confirm")
		applogger.Info("-----------------------------------------------------------------")
		return
	}

	applogger.Info("-----------------------------------------------------------------")
	applogger.Info("= Section 3: 执行 DELETE --confirm 已设置")
	applogger.Info("-----------------------------------------------------------------")
	start := time.Now()
	res := g.Exec(`
		DELETE FROM sys_device_port_status
		 WHERE interface_name ~ '^([0-9]+/)?[0-9]+$|^/[0-9]+$'
	`)
	elapsed := time.Since(start)
	if res.Error != nil {
		applogger.Errorf("[cleanup] DELETE 失败: %v", res.Error)
		os.Exit(1)
	}
	affected := res.RowsAffected
	applogger.Infof("[cleanup] DELETE 完成,影响行数 = %d,耗时 = %s", affected, elapsed)

	// === Section 4: 复盘核对 ===
	applogger.Info("-----------------------------------------------------------------")
	applogger.Info("= Section 4: 复盘核对(再 COUNT 一次)")
	applogger.Info("-----------------------------------------------------------------")
	var remaining int64
	g.Raw(`SELECT COUNT(*) FROM sys_device_port_status WHERE interface_name ~ '^([0-9]+/)?[0-9]+$|^/[0-9]+$'`).Scan(&remaining)
	applogger.Infof("[cleanup] DELETE 后剩余命中行数 = %d (期望 0)", remaining)

	var totalAfter int64
	g.Table("sys_device_port_status").Count(&totalAfter)
	applogger.Infof("[cleanup] sys_device_port_status DELETE 后总行数 = %d", totalAfter)
	applogger.Info("[cleanup] Done.")
}

func nullableStr(p *string) string {
	if p == nil {
		return "(NULL)"
	}
	if *p == "" {
		return "(空)"
	}
	return *p
}
