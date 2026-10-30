// Package main mac_purge_verify 是 mac_purge_meaningless 的验证工具,
// 用于在清理后抽样核对保留记录的语义正确性。
//
// 验证点:
//  1. moved / vlan_changed 总数 = 备份表中对应行数(零数据丢失)
//  2. first-appeared 数量 = backup 表中 distinct (device_id, mac_address) where event_type='appeared'
//  3. 抽样 5 条 moved 记录,显示前后接口名(应当不同)
//  4. 抽样 5 条 vlan_changed 记录,显示前后 VLAN
//  5. 抽样 5 条 first-appeared 记录,确认时间最早
//
// 用法:
//
//	go run ./scripts/mac/purge_verify -backup sys_device_mac_history_purge_backup_20260630_172543
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
)

func main() {
	backupTable := flag.String("backup", "", "备份表名(必填,如 sys_device_mac_history_purge_backup_20260630_172543)")
	flag.Parse()

	if *backupTable == "" {
		fmt.Fprintln(os.Stderr, "用法: go run ./scripts/mac/purge_verify -backup <table_name>")
		os.Exit(2)
	}

	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[mac_purge_verify] .env 未加载: %v", err)
	}
	cfg, err := config.Load(context.Background())
	if err != nil {
		applogger.Errorf("[mac_purge_verify] 配置加载失败: %v", err)
		os.Exit(1)
	}
	applogger.Info("=================================================================")
	applogger.Info("= mac_purge_verify 启动")
	applogger.Infof("= 数据库: PostgreSQL @ %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	applogger.Infof("= 备份表: %s", *backupTable)
	applogger.Info("=================================================================")

	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[mac_purge_verify] 数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := dbConn.GetDB().DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applogger.Info("[verify] === 1. moved 数量核对 ===")
	type CountRow struct {
		EventType string
		Backup    int64
		Current   int64
	}
	var movedCheck CountRow
	if err := dbConn.GetDB().WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT 'moved' AS event_type,
		       (SELECT COUNT(*) FROM %s WHERE event_type = 'moved') AS backup,
		       (SELECT COUNT(*) FROM sys_device_mac_history WHERE event_type = 'moved') AS current
	`, *backupTable)).Scan(&movedCheck).Error; err != nil {
		applogger.Errorf("[verify] moved 核对失败: %v", err)
	} else {
		applogger.Infof("[verify] moved: backup=%d, current=%d, 一致=%v",
			movedCheck.Backup, movedCheck.Current, movedCheck.Backup == movedCheck.Current)
	}

	applogger.Info("[verify] === 2. vlan_changed 数量核对 ===")
	var vlanCheck CountRow
	if err := dbConn.GetDB().WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT 'vlan_changed' AS event_type,
		       (SELECT COUNT(*) FROM %s WHERE event_type = 'vlan_changed') AS backup,
		       (SELECT COUNT(*) FROM sys_device_mac_history WHERE event_type = 'vlan_changed') AS current
	`, *backupTable)).Scan(&vlanCheck).Error; err != nil {
		applogger.Errorf("[verify] vlan_changed 核对失败: %v", err)
	} else {
		applogger.Infof("[verify] vlan_changed: backup=%d, current=%d, 一致=%v",
			vlanCheck.Backup, vlanCheck.Current, vlanCheck.Backup == vlanCheck.Current)
	}

	applogger.Info("[verify] === 3. first-appeared 数量核对 ===")
	type AppearedCheck struct {
		BackupDistinct int64
		CurrentRows    int64
	}
	var appearedCheck AppearedCheck
	if err := dbConn.GetDB().WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT
		    (SELECT COUNT(DISTINCT (device_id, mac_address)) FROM %s WHERE event_type = 'appeared') AS backup_distinct,
		    (SELECT COUNT(*) FROM sys_device_mac_history WHERE event_type = 'appeared') AS current_rows
	`, *backupTable)).Scan(&appearedCheck).Error; err != nil {
		applogger.Errorf("[verify] appeared 核对失败: %v", err)
	} else {
		applogger.Infof("[verify] first-appeared: backup-distinct(MAC+device)=%d, current-appeared-rows=%d, 一致=%v",
			appearedCheck.BackupDistinct, appearedCheck.CurrentRows,
			appearedCheck.BackupDistinct == appearedCheck.CurrentRows)
	}

	applogger.Info("[verify] === 4. moved 抽样 (5 条) ===")
	type MovedSample struct {
		DeviceID      string
		MAC           string
		InterfaceName string
		VLANID        *int
		FirstSeen     time.Time
		LastSeen      time.Time
	}
	var movedSamples []MovedSample
	if err := dbConn.GetDB().WithContext(ctx).
		Table("sys_device_mac_history").
		Select("device_id, mac_address, interface_name, vlan_id, first_seen, last_seen").
		Where("event_type = 'moved'").
		Order("first_seen DESC").
		Limit(5).
		Scan(&movedSamples).Error; err != nil {
		applogger.Warnf("[verify] moved 抽样失败: %v", err)
	} else {
		for i, s := range movedSamples {
			vlan := "NULL"
			if s.VLANID != nil {
				vlan = fmt.Sprintf("%d", *s.VLANID)
			}
			applogger.Infof("[verify]   [%d] device=%s mac=%s iface=%s vlan=%s first_seen=%s last_seen=%s",
				i+1, s.DeviceID, s.MAC, s.InterfaceName, vlan,
				s.FirstSeen.Format("2006-01-02 15:04:05"),
				s.LastSeen.Format("2006-01-02 15:04:05"))
		}
	}

	applogger.Info("[verify] === 5. vlan_changed 抽样 (5 条) ===")
	type VlanSample struct {
		DeviceID      string
		MAC           string
		InterfaceName string
		VLANID        *int
		FirstSeen     time.Time
	}
	var vlanSamples []VlanSample
	if err := dbConn.GetDB().WithContext(ctx).
		Table("sys_device_mac_history").
		Select("device_id, mac_address, interface_name, vlan_id, first_seen").
		Where("event_type = 'vlan_changed'").
		Order("first_seen DESC").
		Limit(5).
		Scan(&vlanSamples).Error; err != nil {
		applogger.Warnf("[verify] vlan_changed 抽样失败: %v", err)
	} else {
		for i, s := range vlanSamples {
			vlan := "NULL"
			if s.VLANID != nil {
				vlan = fmt.Sprintf("%d", *s.VLANID)
			}
			applogger.Infof("[verify]   [%d] device=%s mac=%s iface=%s vlan=%s first_seen=%s",
				i+1, s.DeviceID, s.MAC, s.InterfaceName, vlan,
				s.FirstSeen.Format("2006-01-02 15:04:05"))
		}
	}

	applogger.Info("[verify] === 6. first-appeared 抽样 (5 条) ===")
	type AppearedSample struct {
		DeviceID      string
		MAC           string
		InterfaceName string
		VLANID        *int
		FirstSeen     time.Time
	}
	var appearedSamples []AppearedSample
	if err := dbConn.GetDB().WithContext(ctx).
		Table("sys_device_mac_history").
		Select("device_id, mac_address, interface_name, vlan_id, first_seen").
		Where("event_type = 'appeared'").
		Order("first_seen DESC").
		Limit(5).
		Scan(&appearedSamples).Error; err != nil {
		applogger.Warnf("[verify] appeared 抽样失败: %v", err)
	} else {
		for i, s := range appearedSamples {
			vlan := "NULL"
			if s.VLANID != nil {
				vlan = fmt.Sprintf("%d", *s.VLANID)
			}
			applogger.Infof("[verify]   [%d] device=%s mac=%s iface=%s vlan=%s first_seen=%s",
				i+1, s.DeviceID, s.MAC, s.InterfaceName, vlan,
				s.FirstSeen.Format("2006-01-02 15:04:05"))
		}
	}

	applogger.Info("=================================================================")
	applogger.Info("[verify] 验证完成")
	applogger.Info("=================================================================")
}
