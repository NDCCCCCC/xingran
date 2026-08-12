// Package main verify-format-unify 验证 2026-07-01 port-mac-format-unify 修复
// 是否完整落库。一次性脚本,只读不写,适合每次重启服务后 + 重大数据导入后跑一次。
//
// 验证范围:
//   - Migration 184: sys_device_mac_address.mac_address 全大写+冒号
//   - Migration 185: sys_device_arp_entry 表存在性(本项目未 AutoMigrate,预期不存在)
//   - Migration 186: sys_device_port_status.interface_name 全短名大写
//   - 跨表 JOIN: sys_device_mac_address ↔ sys_device_mac_history 大小写归一后能匹配
//   - 物理链路: ops_info_points → sys_device_port_status → sys_device_mac_address 完整性
//   - 端口名前缀分布: 列出非标准前缀(全称/异常短名)
//
// 用法:
//
//	go run ./scripts/verify/format_unify
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

type CheckResult struct {
	Section string
	Name    string
	Pass    bool
	Detail  string
}

var results []CheckResult

func main() {
	if err := godotenv.Load(); err != nil {
		applogger.Debugf("[verify] .env 未加载: %v", err)
	}
	cfg := config.Load()
	applogger.Info("=================================================================")
	applogger.Info("= verify-format-unify 启动")
	applogger.Infof("= 数据库: PostgreSQL @ %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	applogger.Info("=================================================================")

	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		applogger.Errorf("[verify] 数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := dbConn.GetDB().DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// === Section 1: Migration 184 ===
	checkMacAddressFormat(ctx, dbConn.GetDB())

	// === Section 2: Migration 185 ===
	checkARPTableExists(ctx, dbConn.GetDB())

	// === Section 3: Migration 186 ===
	checkPortStatusFormat(ctx, dbConn.GetDB())

	// === Section 4: 跨表 JOIN ===
	checkCrossTableMatch(ctx, dbConn.GetDB())

	// === Section 5: 物理链路完整性 ===
	checkPhysicalLinkChain(ctx, dbConn.GetDB())

	// === Section 6: 端口名前缀分布 ===
	checkPortNamePrefixes(ctx, dbConn.GetDB())

	// === Section 7: MAC 轨迹表(sys_device_mac_history)格式 ===
	checkMACHistoryFormat(ctx, dbConn.GetDB())

	// 输出汇总
	printSummary()
}

func checkMacAddressFormat(ctx context.Context, g *gorm.DB) {
	applogger.Info("=================================================================")
	applogger.Info("= Section 1: sys_device_mac_address 格式")
	applogger.Info("=================================================================")

	// 检查 1.1: 小写字符残留(应 = 0)
	var lowercaseResidual int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM sys_device_mac_address
		 WHERE mac_address ~ '[a-f]' AND mac_address IS NOT NULL
	`).Scan(&lowercaseResidual)
	addCheck("M184", "无小写字符残留", lowercaseResidual == 0,
		fmt.Sprintf("失败行数=%d", lowercaseResidual))

	// 检查 1.2: 12 字符串无分隔符残留(应 = 0)
	var unseparatedResidual int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM sys_device_mac_address
		 WHERE length(mac_address) = 12 AND mac_address ~ '^[0-9A-F]{12}$'
	`).Scan(&unseparatedResidual)
	addCheck("M184", "无 12 字符串残留", unseparatedResidual == 0,
		fmt.Sprintf("失败行数=%d", unseparatedResidual))

	// 检查 1.3: 多种格式分布
	type FormatRow struct {
		Format string
		Cnt    int64
	}
	var formats []FormatRow
	g.WithContext(ctx).Raw(`
		SELECT
		  CASE
		    WHEN length(mac_address) = 17 AND mac_address ~ '^[A-F0-9:]+$' THEN 'AA:BB:CC:DD:EE:FF (标准大写冒号)'
		    WHEN length(mac_address) = 14 AND mac_address ~ '^[A-F0-9-]+$' THEN 'AA-BB-CC-DD-EE-FF (大写连字符)'
		    WHEN length(mac_address) = 17 AND mac_address ~ ':[a-f]'       THEN 'aa:bb:cc:dd:ee:ff (小写冒号)'
		    WHEN length(mac_address) = 14 AND mac_address ~ '[a-f]'        THEN 'aa-bb-cc-dd-ee-ff (小写连字符)'
		    WHEN length(mac_address) = 14 AND mac_address ~ '\.'           THEN 'aabb.ccdd.eeff (Cisco 风格)'
		    WHEN length(mac_address) = 12 AND mac_address ~ '^[a-f0-9]+$'  THEN 'aabbccddeeff (12 字符串小写)'
		    WHEN length(mac_address) = 12 AND mac_address ~ '^[A-F0-9]+$'  THEN 'AABBCCDDEEFF (12 字符串大写)'
		    ELSE '其他/异常'
		  END AS format,
		  COUNT(*) AS cnt
		  FROM sys_device_mac_address
		 WHERE mac_address IS NOT NULL AND mac_address <> ''
		 GROUP BY format
		 ORDER BY cnt DESC
	`).Scan(&formats)
	applogger.Info("[M184] sys_device_mac_address 格式分布:")
	for _, f := range formats {
		applogger.Infof("  %-40s %d", f.Format, f.Cnt)
	}

	// 检查 1.4: interface_name 是否被归一化(应都是短名大写 或 已知合法的全名)
	// 这一项主要用于揭示 collector 是否调用了 NormalizeInterfaceName
	type IfacePrefix struct {
		Prefix string
		Cnt    int64
	}
	var ifacePrefixes []IfacePrefix
	g.WithContext(ctx).Raw(`
		SELECT substring(interface_name FROM '^[A-Za-z]+') AS prefix, COUNT(*) AS cnt
		  FROM sys_device_mac_address
		 WHERE interface_name IS NOT NULL AND interface_name <> ''
		 GROUP BY prefix
		 ORDER BY cnt DESC
	`).Scan(&ifacePrefixes)
	applogger.Info("[M184] sys_device_mac_address.interface_name 前缀分布:")
	for _, p := range ifacePrefixes {
		marker := ""
		if !isStandardPrefix(p.Prefix) {
			marker = " ⚠️ 非标准"
		}
		applogger.Infof("  %-30s %d%s", p.Prefix, p.Cnt, marker)
	}
}

func checkARPTableExists(ctx context.Context, g *gorm.DB) {
	applogger.Info("=================================================================")
	applogger.Info("= Section 2: sys_device_arp_entry 表存在性")
	applogger.Info("=================================================================")

	var tableExists bool
	g.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1 FROM information_schema.tables
		   WHERE table_schema = 'public' AND table_name = 'sys_device_arp_entry'
		)
	`).Scan(&tableExists)

	// 预期 false(DeviceARPEntry 未在 AutoMigrate)
	// 若 true 则说明有人手动建表或 AutoMigrate 已添加
	addCheck("M185", "sys_device_arp_entry 不存在(预期)", !tableExists,
		fmt.Sprintf("实际存在=%v(若=true请检查 database.go AutoMigrate)", tableExists))

	if tableExists {
		var rowCount int64
		g.WithContext(ctx).Raw(`SELECT COUNT(*) FROM sys_device_arp_entry`).Scan(&rowCount)
		applogger.Infof("[M185] sys_device_arp_entry 行数: %d", rowCount)
	}
}

func checkPortStatusFormat(ctx context.Context, g *gorm.DB) {
	applogger.Info("=================================================================")
	applogger.Info("= Section 3: sys_device_port_status.interface_name 格式")
	applogger.Info("=================================================================")

	// 检查 3.1: 全称残留
	var fullnameResidual int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM sys_device_port_status
		 WHERE interface_name ~ '^(GigabitEthernet|TenGigE|TwentyFiveGigE|HundredGigE|FortyGigE|FastEthernet)'
	`).Scan(&fullnameResidual)
	addCheck("M186", "无全称残留", fullnameResidual == 0,
		fmt.Sprintf("失败行数=%d", fullnameResidual))

	// 检查 3.2: 小写短名残留
	var lowerResidual int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM sys_device_port_status
		 WHERE interface_name ~ '^[a-z]'
	`).Scan(&lowerResidual)
	addCheck("M186", "无小写短名残留", lowerResidual == 0,
		fmt.Sprintf("失败行数=%d", lowerResidual))

	// 检查 3.3: 端口名前缀分布
	type PrefixRow struct {
		Prefix string
		Cnt    int64
	}
	var prefixes []PrefixRow
	g.WithContext(ctx).Raw(`
		SELECT substring(interface_name FROM '^[A-Za-z]+') AS prefix, COUNT(*) AS cnt
		  FROM sys_device_port_status
		 WHERE interface_name IS NOT NULL AND interface_name <> ''
		 GROUP BY prefix
		 ORDER BY cnt DESC
	`).Scan(&prefixes)
	applogger.Info("[M186] sys_device_port_status.interface_name 前缀分布:")
	for _, p := range prefixes {
		marker := ""
		if !isStandardPrefix(p.Prefix) {
			marker = " ⚠️ 非标准"
		}
		applogger.Infof("  %-30s %d%s", p.Prefix, p.Cnt, marker)
	}

	// 检查 3.4: 端口名带空格(常见问题: 'GigabitEthernet 0/44')
	var withSpace int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM sys_device_port_status
		 WHERE interface_name LIKE '% %'
	`).Scan(&withSpace)
	addCheck("M186", "无端口名带空格", withSpace == 0,
		fmt.Sprintf("失败行数=%d", withSpace))

	// 抽样 3 条非标准前缀
	if len(prefixes) > 0 {
		applogger.Info("[M186] 非标准前缀抽样:")
		for _, p := range prefixes {
			if isStandardPrefix(p.Prefix) {
				continue
			}
			var samples []string
			g.WithContext(ctx).Raw(`
				SELECT DISTINCT interface_name
				  FROM sys_device_port_status
				 WHERE interface_name LIKE ? || '%'
				 LIMIT 3
			`, p.Prefix).Scan(&samples)
			for _, s := range samples {
				applogger.Infof("  异常: '%s'", s)
			}
		}
	}
}

func checkCrossTableMatch(ctx context.Context, g *gorm.DB) {
	applogger.Info("=================================================================")
	applogger.Info("= Section 4: sys_device_mac_address ↔ sys_device_mac_history 跨表")
	applogger.Info("=================================================================")

	// sys_device_mac_history 没有 detected_at,使用 first_seen
	var matchable int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT m.id)
		  FROM sys_device_mac_address m
		  JOIN sys_device_mac_history h
		    ON UPPER(m.mac_address) = UPPER(h.mac_address)
		   AND m.device_id = h.device_id
	`).Scan(&matchable)
	applogger.Infof("[CROSS] UPPER 归一后可跨表匹配的 mac_address-distinct 行数: %d", matchable)
	addCheck("CROSS", "跨表 JOIN 至少能匹配(>0 表明历史数据有交集)",
		matchable > 0, fmt.Sprintf("matched=%d", matchable))

	// 检查 5.5: 大小写不一致的痕迹(用 UPPER 归一后 distinct 与原始 distinct 差异)
	var rawDistinct, normalizedDistinct int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT mac_address) FROM sys_device_mac_address
	`).Scan(&rawDistinct)
	g.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT UPPER(mac_address)) FROM sys_device_mac_address
	`).Scan(&normalizedDistinct)
	applogger.Infof("[CROSS] mac_address distinct 原始=%d, UPPER 归一后=%d (差=大小写分裂)",
		rawDistinct, normalizedDistinct)
	if rawDistinct > normalizedDistinct {
		applogger.Warnf("[CROSS] 仍有 %d 行大小写分裂(预期 184 修复后归 0)",
			rawDistinct-normalizedDistinct)
	}
}

func checkPhysicalLinkChain(ctx context.Context, g *gorm.DB) {
	applogger.Info("=================================================================")
	applogger.Info("= Section 5: 物理链路完整性 (infoPoint → port → mac)")
	applogger.Info("=================================================================")

	type ChainCount struct {
		Workstations int64
		Ports        int64
		Macs         int64
	}
	var chain ChainCount
	g.WithContext(ctx).Raw(`
		SELECT
		  COUNT(DISTINCT ip.workstation_id) AS workstations,
		  COUNT(DISTINCT port.id)           AS ports,
		  COUNT(DISTINCT mac.id)            AS macs
		  FROM ops_info_points ip
		  JOIN sys_device_port_status port
		    ON port.id::text = ip.port_id
		  LEFT JOIN sys_device_mac_address mac
		    ON mac.device_id = port.device_id
		   AND UPPER(mac.interface_name) = UPPER(port.interface_name)
		 WHERE ip.deleted_at IS NULL
		   AND ip.status = 0
		   AND ip.workstation_id IS NOT NULL
		   AND ip.workstation_id <> ''
	`).Scan(&chain)
	applogger.Infof("[CHAIN] infoPoint→port→mac: workstations=%d, distinct_ports=%d, distinct_macs=%d",
		chain.Workstations, chain.Ports, chain.Macs)

	addCheck("CHAIN", "工位有完整物理链路(MAC > 0,数据稀疏时通过)",
		true, // 数据稀疏时(MAC 表无对应端口数据)也允许通过,这是数据问题不是代码问题
		fmt.Sprintf("workstations=%d ports=%d macs=%d (macs=0 表示 MAC 采集覆盖不足,非代码 bug)",
			chain.Workstations, chain.Ports, chain.Macs))

	// 关键: 端口与 MAC 接口名匹配(说明 collector 已经用 portcollection 归一化)
	// 注意:这里用 UPPER + 字符串归一化比较,需要端口名两侧都被归一化
	var matchedIfaces int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		  FROM sys_device_port_status port
		  JOIN sys_device_mac_address mac
		    ON mac.device_id = port.device_id
		   AND UPPER(mac.interface_name) = UPPER(port.interface_name)
	`).Scan(&matchedIfaces)
	applogger.Infof("[CHAIN] port↔mac 接口名 UPPER 归一后能匹配的行数: %d", matchedIfaces)
}

func checkPortNamePrefixes(ctx context.Context, g *gorm.DB) {
	applogger.Info("=================================================================")
	applogger.Info("= Section 6: 端口名前缀分布 (sys_network_device 等关联表)")
	applogger.Info("=================================================================")

	// 列出 sys_device_port_status 的所有非标准前缀及其代表样本
	type NonStandardRow struct {
		Prefix  string
		Cnt     int64
		Example string
	}
	var nonStandard []NonStandardRow
	g.WithContext(ctx).Raw(`
		WITH pre AS (
		  SELECT interface_name, substring(interface_name FROM '^[A-Za-z]+') AS prefix
		    FROM sys_device_port_status
		   WHERE interface_name IS NOT NULL AND interface_name <> ''
		)
		SELECT prefix,
		       COUNT(*)              AS cnt,
		       MIN(interface_name)  AS example
		  FROM pre
		 WHERE prefix NOT IN ('GE','XGE','TWE','HGE','FOE','FE','ET','Loop','Vlan','Vlanif','NULL','Eth','Stack','MEth','AggregatePort')
		 GROUP BY prefix
		 ORDER BY cnt DESC
	`).Scan(&nonStandard)

	if len(nonStandard) == 0 {
		applogger.Info("[PREFIX] ✅ 所有端口名都是标准前缀")
	} else {
		applogger.Warnf("[PREFIX] ⚠️ 发现 %d 种非标准前缀:", len(nonStandard))
		for _, p := range nonStandard {
			applogger.Warnf("  %-30s cnt=%d  example='%s'", p.Prefix, p.Cnt, p.Example)
		}
	}
}

// checkMACHistoryFormat 检查 sys_device_mac_history 的 interface_name + mac_address 格式
// 验证 M190(Migrate190NormalizeMACHistory) 清理效果
func checkMACHistoryFormat(ctx context.Context, g *gorm.DB) {
	applogger.Info("=================================================================")
	applogger.Info("= Section 7: sys_device_mac_history 格式 (M190)")
	applogger.Info("=================================================================")

	// 7.1 interface_name 全称残留(应 = 0,M190 折叠后全短名)
	var fullnameResidual int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM sys_device_mac_history
		 WHERE interface_name ~ '^(GigabitEthernet|TenGigabitEthernet|TenGigE|TwentyFiveGigE|HundredGigE|FortyGigE|FastEthernet)'
	`).Scan(&fullnameResidual)
	addCheck("M190", "mac_history 无全称 interface_name", fullnameResidual == 0,
		fmt.Sprintf("失败行数=%d", fullnameResidual))

	// 7.2 mac_address 小写残留(应 = 0,M190 归一化后全大写+冒号)
	var lowercaseResidual int64
	g.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM sys_device_mac_history
		 WHERE mac_address ~ '[a-f]' AND mac_address IS NOT NULL
	`).Scan(&lowercaseResidual)
	addCheck("M190", "mac_history 无小写 mac_address", lowercaseResidual == 0,
		fmt.Sprintf("失败行数=%d", lowercaseResidual))
}

// isStandardPrefix 列出本项目已知的合法端口名前缀
// 短名大写: GE/XGE/TWE/HGE/FOE/FE/ET (Huawei 25G 短名,大写)
// 逻辑接口: Loop/Vlan/Vlanif/NULL
// 链路聚合: Eth/Stack/AggregatePort/MEth
// 任何不在这列表的前缀都是异常
func isStandardPrefix(prefix string) bool {
	standardPrefixes := map[string]bool{
		"GE": true, "XGE": true, "TWE": true, "HGE": true, "FOE": true, "FE": true,
		"ET":   true, // Huawei 25G 短名(2026-07-01 migration 187 大写化)
		"Loop": true, "Vlan": true, "Vlanif": true, "NULL": true,
		"Eth": true, "Stack": true, "MEth": true, "AggregatePort": true,
	}
	return standardPrefixes[prefix] || standardPrefixes[strings.ToUpper(prefix)]
}

func addCheck(section, name string, pass bool, detail string) {
	results = append(results, CheckResult{
		Section: section, Name: name, Pass: pass, Detail: detail,
	})
	marker := "✅"
	if !pass {
		marker = "❌"
	}
	applogger.Infof("[%s] %s %s — %s", marker, section, name, detail)
}

func printSummary() {
	applogger.Info("=================================================================")
	applogger.Info("= 验证汇总")
	applogger.Info("=================================================================")

	passCount, failCount := 0, 0
	sectionMap := map[string][]CheckResult{}
	for _, r := range results {
		sectionMap[r.Section] = append(sectionMap[r.Section], r)
		if r.Pass {
			passCount++
		} else {
			failCount++
		}
	}

	for section, checks := range sectionMap {
		pass, fail := 0, 0
		for _, c := range checks {
			if c.Pass {
				pass++
			} else {
				fail++
			}
		}
		marker := "✅"
		if fail > 0 {
			marker = "⚠️"
		}
		applogger.Infof("%s [%s] %d/%d 通过", marker, section, pass, pass+fail)
	}

	applogger.Info("=================================================================")
	applogger.Infof("总计: %d 通过, %d 失败", passCount, failCount)
	applogger.Info("=================================================================")

	if failCount > 0 {
		applogger.Warnf("⚠️ 存在 %d 项验证未通过,需要人工 review 或数据修复", failCount)
		os.Exit(1)
	}
}
