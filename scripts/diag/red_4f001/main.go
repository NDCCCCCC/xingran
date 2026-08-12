// Package main diag_4f001_red 诊断工位 4F001 (设备 CXHUB-151008W / 1CZ151008W) 在
// 对账看板显示红色的问题。
//
// 一次性诊断工具, 只读不写, 不修改任何数据库状态。
//
// 排查目标 (来自 .planning/debug/reconciliation-health-misjudgment.md):
//   - 三路径设备完全一致 (域控/资产/物理链路) 但 health 显示红色
//   - 用户报告健康详情: "物理有/责任人不一致(高危)"
//   - 预期: detection 走 Type A (物理 + 声明匹配, 健康), 实际走了非 A 分支
//
// 排查维度:
//  1. ops_asset 原始 user_id / nowuser_name (declared 端)
//  2. sys_workstation.user_id / user_name (工位所属人员)
//  3. sys_user 中 username/nickname='程步启' 的 id
//  4. ops_info_points (port_id → workstation_id 桥)
//  5. sys_device_mac_address (asset → port 桥的 MAC 端)
//  6. sys_device_port_status (asset → port 桥的 port 端)
//  7. reconciliation_physical_chain view (4F001 → 设备 → 物理用户)
//  8. reconciliation_normalized MV (detection 引擎的输入)
//  9. sys_data_reconciliation (detection 引擎的输出)
//
// 用法:
//
//	go run ./scripts/diag/red_4f001
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

const (
	targetDevicesn    = "1CZ151008W"
	targetWorkstation = "4F001"
	targetAssetMAC    = "B0:22:7A:2E:4A:4F" // 大写冒号, sys_device_mac_address 归一化后格式
	targetUserHint    = "程步启"
	targetAssetIP     = "10.62.8.1"
)

func main() {
	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[diag] .env 未加载: %v", err)
	}
	cfg := config.Load()
	applogger.Info("=================================================================")
	applogger.Info("= diag_4f001_red 启动: 排查工位 4F001 / 设备 1CZ151008W 红点")
	applogger.Infof("= 数据库: PostgreSQL @ %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	applogger.Info("=================================================================")

	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[diag] 数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, _ := dbConn.GetDB().DB()
		_ = sqlDB.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	g := dbConn.GetDB().WithContext(ctx)

	section("1. ops_asset (declared 端)", func() {
		type Row struct {
			ID          string
			DeviceSN    string
			DeviceName  *string
			MachineIP   *string
			MAC1        *string
			MAC2        *string
			UserID      *string
			NowUserName *string
			UserName    *string
			DeptName    *string
			Status      int
		}
		var rows []Row
		g.Raw(`
			SELECT id, devicesn, devicename, machine_ip, mac1, mac2,
			       user_id, nowuser_name, user_name, deptname, status
			  FROM ops_asset
			 WHERE devicesn = ?
			    OR (mac1 IS NOT NULL AND LOWER(REGEXP_REPLACE(mac1, '[.:\-]', '', 'g'))
			            = LOWER(REGEXP_REPLACE(?, '[.:\-]', '', 'g')))
			    OR (mac2 IS NOT NULL AND LOWER(REGEXP_REPLACE(mac2, '[.:\-]', '', 'g'))
			            = LOWER(REGEXP_REPLACE(?, '[.:\-]', '', 'g')))
			 LIMIT 5`,
			targetDevicesn, targetAssetMAC, targetAssetMAC).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[1] ops_asset 未找到设备 %s (devicesn/mac1/mac2)", targetDevicesn)
			return
		}
		for _, r := range rows {
			applogger.Infof("[1] id=%s devicesn=%s name=%v ip=%v", r.ID, r.DeviceSN, r.DeviceName, r.MachineIP)
			applogger.Infof("    mac1=%v mac2=%v", r.MAC1, r.MAC2)
			applogger.Infof("    user_id=%v nowuser_name=%v user_name=%v deptname=%v status=%d",
				r.UserID, r.NowUserName, r.UserName, r.DeptName, r.Status)
			// 关键诊断: user_id 是否为空 / 是否是 uuid 格式 / 是否能 join 到 sys_user
			if r.UserID != nil {
				applogger.Infof("    ⚠ user_id 类型检查: 长度=%d, 是否UUID格式=%v",
					len(*r.UserID), isUUIDLike(*r.UserID))
			} else {
				applogger.Warnf("    ⚠ user_id 为 NULL — detection 会认为 declared 无")
			}
		}
	})

	section("2. sys_workstation (工位所属人员)", func() {
		type Row struct {
			ID              string
			WorkstationName string
			UserID          *string
			UserName        *string
		}
		var rows []Row
		g.Raw(`
			SELECT id, name, user_id, user_name
			  FROM sys_workstation
			 WHERE name = ?
			    OR id::text IN (SELECT workstation_id FROM ops_info_points WHERE deleted_at IS NULL LIMIT 100)
			 LIMIT 50`,
			targetWorkstation).Scan(&rows)
		// 找 4F001 + 通过 info_points 关联的工作站
		var hits []Row
		for _, r := range rows {
			if r.WorkstationName == targetWorkstation {
				hits = append(hits, r)
			}
		}
		if len(hits) == 0 {
			// 简化: 用名字精确匹配
			g.Raw(`SELECT id, name, user_id, user_name FROM sys_workstation WHERE name = ?`,
				targetWorkstation).Scan(&hits)
		}
		if len(hits) == 0 {
			applogger.Warnf("[2] sys_workstation 未找到 %s", targetWorkstation)
			return
		}
		for _, r := range hits {
			applogger.Infof("[2] id=%s name=%s user_id=%v user_name=%v",
				r.ID, r.WorkstationName, r.UserID, r.UserName)
			if r.UserID != nil {
				applogger.Infof("    ⚠ user_id 类型检查: 长度=%d, 是否UUID格式=%v",
					len(*r.UserID), isUUIDLike(*r.UserID))
			} else {
				applogger.Warnf("    ⚠ user_id 为 NULL — detection 会认为 physical 无")
			}
		}
	})

	section("3. sys_user 程步启 (id 反查)", func() {
		type Row struct {
			ID       string
			Username string
			Nickname *string
			Status   int
		}
		var rows []Row
		g.Raw(`
			SELECT id, username, nickname, status
			  FROM sys_user
			 WHERE username = ? OR nickname = ?
			 LIMIT 10`,
			targetUserHint, targetUserHint).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[3] sys_user 未找到 username/nickname=%s", targetUserHint)
			return
		}
		for _, r := range rows {
			applogger.Infof("[3] id=%s username=%s nickname=%v status=%d",
				r.ID, r.Username, r.Nickname, r.Status)
		}
	})

	section("4. sys_device_mac_address (asset → MAC 链路)", func() {
		type Row struct {
			ID            string
			MACAddress    string
			InterfaceName string
			DeviceID      string
			CollectedAt   *time.Time
		}
		var rows []Row
		g.Raw(`
			SELECT id, mac_address, interface_name, device_id, collected_at
			  FROM sys_device_mac_address
			 WHERE LOWER(REGEXP_REPLACE(COALESCE(mac_address,''), '[.:\-]', '', 'g'))
			     = LOWER(REGEXP_REPLACE(?, '[.:\-]', '', 'g'))
			 ORDER BY collected_at DESC NULLS LAST
			 LIMIT 10`,
			targetAssetMAC).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[4] sys_device_mac_address 未找到 mac=%s", targetAssetMAC)
			return
		}
		for _, r := range rows {
			applogger.Infof("[4] id=%s mac=%s iface=%s device_id=%s collected=%v",
				r.ID, r.MACAddress, r.InterfaceName, r.DeviceID, r.CollectedAt)
		}
	})

	section("5. sys_device_port_status (port 端)", func() {
		// 通过上面 device_id 查询
		var deviceIDs []string
		g.Raw(`
			SELECT DISTINCT device_id FROM sys_device_mac_address
			 WHERE LOWER(REGEXP_REPLACE(COALESCE(mac_address,''), '[.:\-]', '', 'g'))
			     = LOWER(REGEXP_REPLACE(?, '[.:\-]', '', 'g'))`,
			targetAssetMAC).Scan(&deviceIDs)
		if len(deviceIDs) == 0 {
			applogger.Warnf("[5] sys_device_port_status 跳过 (无 device_id)")
			return
		}
		type Row struct {
			ID            string
			DeviceID      string
			InterfaceName string
		}
		var rows []Row
		g.Raw(`
			SELECT id, device_id, interface_name FROM sys_device_port_status
			 WHERE device_id IN ?
			 ORDER BY device_id LIMIT 30`,
			deviceIDs).Scan(&rows)
		applogger.Infof("[5] port 数: %d (device_ids=%v)", len(rows), deviceIDs)
		for _, r := range rows {
			applogger.Infof("    port id=%s device_id=%s iface=%s", r.ID, r.DeviceID, r.InterfaceName)
		}
	})

	section("6. ops_info_points (port → workstation)", func() {
		type Row struct {
			ID            string
			PortID        string
			WorkstationID string
			DeviceID      *string
			Status        int
		}
		var wsID string
		g.Raw(`SELECT id::text FROM sys_workstation WHERE name = ? LIMIT 1`, targetWorkstation).Scan(&wsID)
		if wsID == "" {
			applogger.Warnf("[6] ops_info_points 跳过 (未找到工位 %s)", targetWorkstation)
			return
		}
		var rows []Row
		g.Raw(`
			SELECT id, port_id, workstation_id, device_id, status
			  FROM ops_info_points
			 WHERE deleted_at IS NULL AND workstation_id = ?
			 LIMIT 20`, wsID).Scan(&rows)
		applogger.Infof("[6] workstation_id=%s, info_points 数: %d", wsID, len(rows))
		for _, r := range rows {
			applogger.Infof("    ip id=%s port_id=%s ws_id=%s device_id=%v status=%d",
				r.ID, r.PortID, r.WorkstationID, r.DeviceID, r.Status)
		}
	})

	section("7. reconciliation_physical_chain VIEW (整链路集成)", func() {
		type Row struct {
			AssetID          string
			AssetCode        string
			MacJoin          *string
			WorkstationID    *string
			PhysicalUserID   *string
			PhysicalUsername *string
		}
		var rows []Row
		g.Raw(`
			SELECT asset_id, asset_code, mac_join, workstation_id, physical_user_id, physical_username
			  FROM reconciliation_physical_chain
			 WHERE asset_code = ?
			    OR LOWER(REGEXP_REPLACE(COALESCE(mac_join,''), '[.:\-]', '', 'g'))
			     = LOWER(REGEXP_REPLACE(?, '[.:\-]', '', 'g'))
			 LIMIT 10`, targetDevicesn, targetAssetMAC).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[7] reconciliation_physical_chain 该设备未命中 (链路断裂!)")
			return
		}
		for _, r := range rows {
			applogger.Infof("[7] asset_id=%s asset_code=%s mac_join=%v", r.AssetID, r.AssetCode, r.MacJoin)
			applogger.Infof("    workstation_id=%v physical_user_id=%v physical_username=%v",
				r.WorkstationID, r.PhysicalUserID, r.PhysicalUsername)
			if r.PhysicalUserID != nil {
				applogger.Infof("    ⚠ physical_user_id 类型: 长度=%d UUID格式=%v",
					len(*r.PhysicalUserID), isUUIDLike(*r.PhysicalUserID))
			}
		}
	})

	section("8. reconciliation_normalized MV (detection 引擎输入)", func() {
		type Row struct {
			AssetID          string
			AssetCode        string
			AssetIP          *string
			MacJoin          *string
			AssetUserID      *string
			AssetUsername    *string
			WorkstationID    *string
			PhysicalUserID   *string
			PhysicalUsername *string
			AdUsername       *string
		}
		var rows []Row
		g.Raw(`
			SELECT asset_id, asset_code, asset_ip, mac_join,
			       asset_user_id, asset_username,
			       workstation_id, physical_user_id, physical_username,
			       ad_username
			  FROM reconciliation_normalized
			 WHERE asset_code = ?
			    OR LOWER(REGEXP_REPLACE(COALESCE(mac_join,''), '[.:\-]', '', 'g'))
			     = LOWER(REGEXP_REPLACE(?, '[.:\-]', '', 'g'))
			 LIMIT 10`, targetDevicesn, targetAssetMAC).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[8] reconciliation_normalized MV 该设备未命中")
			return
		}
		for _, r := range rows {
			applogger.Infof("[8] asset_id=%s asset_code=%s ip=%v mac=%v",
				r.AssetID, r.AssetCode, r.AssetIP, r.MacJoin)
			applogger.Infof("    asset_user_id=%v asset_username=%v",
				r.AssetUserID, r.AssetUsername)
			applogger.Infof("    workstation_id=%v physical_user_id=%v physical_username=%v",
				r.WorkstationID, r.PhysicalUserID, r.PhysicalUsername)
			applogger.Infof("    ad_username=%v", r.AdUsername)

			// === 关键: 模拟 detection ClassifySignals ===
			hasPhysical := r.PhysicalUserID != nil && *r.PhysicalUserID != ""
			hasDeclared := r.AssetUserID != nil && *r.AssetUserID != ""
			hasAD := r.AdUsername != nil && *r.AdUsername != ""
			match := hasPhysical && hasDeclared && *r.PhysicalUserID == *r.AssetUserID
			applogger.Infof("    >>> ClassifySignals: HasPhysical=%v HasDeclared=%v HasAD=%v Match=%v",
				hasPhysical, hasDeclared, hasAD, match)
			var t string
			if hasPhysical && hasDeclared && match {
				if hasAD && *r.AdUsername != *r.PhysicalUsername {
					t = "F"
				} else {
					t = "A"
				}
			} else if hasPhysical && !hasDeclared {
				t = "B"
			} else if hasPhysical && hasDeclared && !match {
				t = "C"
			} else if !hasPhysical && hasDeclared {
				t = "D"
			} else if !hasPhysical && !hasDeclared {
				t = "E"
			} else {
				t = "F"
			}
			applogger.Infof("    >>> ClassifyType → %s (应 A 才健康)", t)
		}
	})

	section("9. sys_data_reconciliation (detection 引擎输出, 最近 24h)", func() {
		type Row struct {
			AssetID          string
			ConflictType     string
			Severity         string
			DetectedAt       time.Time
			AssetUsername    *string
			PhysicalUsername *string
		}
		var rows []Row
		g.Raw(`
			SELECT r.asset_id, r.conflict_type, r.severity, r.detected_at,
			       r.asset_username, r.physical_username
			  FROM sys_data_reconciliation r
			 WHERE r.deleted_at IS NULL
			   AND r.detected_at > NOW() - INTERVAL '24 hours'
			   AND r.asset_id IN (
			       SELECT id FROM ops_asset WHERE devicesn = ?
			   )
			 ORDER BY r.detected_at DESC LIMIT 20`, targetDevicesn).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[9] sys_data_reconciliation 该资产最近 24h 无记录 (Type A 健康不入主表)")
			return
		}
		for _, r := range rows {
			applogger.Infof("[9] asset=%s type=%s severity=%s at=%v",
				r.AssetID, r.ConflictType, r.Severity, r.DetectedAt)
			applogger.Infof("    asset_username=%v physical_username=%v",
				r.AssetUsername, r.PhysicalUsername)
		}
	})

	section("10. ops_workstation_device 4F001 关联资产 (健康计算真实来源)", func() {
		type Row struct {
			WorkstationID string
			AssetID       string
			DeviceSN      *string
		}
		var wsID string
		g.Raw(`SELECT id::text FROM sys_workstation WHERE workstation_name = ? LIMIT 1`, targetWorkstation).Scan(&wsID)
		if wsID == "" {
			applogger.Warnf("[10] sys_workstation 未找到 %s", targetWorkstation)
			return
		}
		applogger.Infof("[10] 4F001 workstation_id=%s", wsID)
		var rows []Row
		g.Raw(`
			SELECT wsd.workstation_id::text, wsd.asset_id, a.devicesn
			  FROM ops_workstation_device wsd
			  LEFT JOIN ops_asset a ON a.id::text = wsd.asset_id AND a.deleted_at IS NULL
			 WHERE wsd.workstation_id = ? AND wsd.deleted_at IS NULL`, wsID).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[10] ops_workstation_device 未找到 4F001 的关联资产")
			return
		}
		applogger.Infof("[10] 4F001 关联资产数 (HealthScore.Total): %d", len(rows))
		for _, r := range rows {
			applogger.Infof("    ws=%s asset=%s devicesn=%v",
				r.WorkstationID, r.AssetID, r.DeviceSN)
		}
	})

	section("11. sys_data_reconciliation 4F001 关联资产实际冲突 (健康分母)", func() {
		type Row struct {
			AssetID         string
			ConflictType    string
			Severity        string
			DetectedAt      time.Time
			ResolvedAt      *time.Time
			ExceptionRuleID *string
		}
		var wsID string
		g.Raw(`SELECT id::text FROM sys_workstation WHERE workstation_name = ? LIMIT 1`, targetWorkstation).Scan(&wsID)
		if wsID == "" {
			applogger.Warnf("[11] sys_data_reconciliation 跳过 (无 4F001 id)")
			return
		}
		var rows []Row
		// wsd.asset_id 是 varchar(36) 存 UUID, sys_data_reconciliation.asset_id 是 uuid
		// 需要 wsd.asset_id::uuid 强转
		g.Raw(`
			SELECT r.asset_id::text, r.conflict_type, r.severity, r.detected_at, r.resolved_at, r.exception_rule_id
			  FROM sys_data_reconciliation r
			 WHERE r.deleted_at IS NULL
			   AND r.asset_id IN (
			       SELECT wsd.asset_id::uuid FROM ops_workstation_device wsd
			        WHERE wsd.workstation_id = ? AND wsd.deleted_at IS NULL)
			 ORDER BY r.detected_at DESC LIMIT 30`, wsID).Scan(&rows)
		if len(rows) == 0 {
			applogger.Warnf("[11] 4F001 关联资产在 sys_data_reconciliation 无记录 (全 Type A 健康!)")
			applogger.Warnf("      HealthScore 应 = 100 绿色, 红色 = 服务端 bug 或缓存陈旧")
			return
		}
		applogger.Infof("[11] sys_data_reconciliation 4F001 关联资产的冲突记录数: %d", len(rows))
		openCount := 0
		for _, r := range rows {
			open := r.ResolvedAt == nil
			if open {
				openCount++
			}
			applogger.Infof("    asset=%s type=%s severity=%s resolved=%v exception=%v at=%v",
				r.AssetID, r.ConflictType, r.Severity, r.ResolvedAt, r.ExceptionRuleID, r.DetectedAt)
		}
		applogger.Infof("[11] 未解决 (resolved_at IS NULL) 记录数: %d — 这是 HealthScore 分母的 drift/conflict/noData 来源", openCount)
	})

	section("12. SUMMARY - 红点根因定位", func() {
		applogger.Info("==========================================================")
		applogger.Info("= 综合判断 (看上面各 section 输出来判断):")
		applogger.Info("= 1) Section 8 MV 是 Type A 健康 — 说明 detection 没问题")
		applogger.Info("= 2) Section 10 ops_workstation_device 是 health 真实关联表")
		applogger.Info("=    - 若 assets 数与 4F001 实际设备数不一致 → 关联表有 stale 数据")
		applogger.Info("= 3) Section 11 sys_data_reconciliation 是 health score 分母")
		applogger.Info("=    - 若有 open 记录 → 红点的真因是这些冲突")
		applogger.Info("=    - 若无 open 记录 → 红点 = 服务端 bug / Redis 缓存陈旧")
		applogger.Info("==========================================================")
	})
}

func section(name string, fn func()) {
	applogger.Info("=================================================================")
	applogger.Infof("= %s", name)
	applogger.Info("=================================================================")
	fn()
}

func isUUIDLike(s string) bool {
	// 简单检查 36 字符 + 8-4-4-4-12 格式
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// 取消 fmt 引用以防 unused
var _ = fmt.Sprintf
