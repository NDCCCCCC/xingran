//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate190NormalizeMACHistory 把 sys_device_mac_history 表里的 mac_address 与
// interface_name 历史数据归一化。
//
// 背景(2026-07-01 port-mac-format-unify):
//   - MAC 轨迹表此前从 BuildMACStateMap 写入,该函数只归一化 MAC 未归一化 InterfaceName
//     (已修),导致历史 interface_name 全是全称(GigabitEthernet0/0/1)。
//   - mac_address 也可能有小写/点分隔/连字符等格式未归一化的历史值。
//   - M184/M186/M187 只清理 sys_device_mac_address 与 sys_device_port_status,
//     sys_device_mac_history 此前无清理迁移。
//
// 归一化(与 pkg/normalize 对齐):
//   - mac_address → 大写+冒号 (AA:BB:CC:DD:EE:FF),复用 M184 同款"剥分隔符→UPPER→插冒号"
//   - interface_name → 大写短名 (GE0/0/1),复用 M187 Step1 同款 CASE WHEN 折叠
//
// 安全: sys_device_mac_history 无 (device_id, interface_name) 唯一约束(历史记录表),
// 直接 UPDATE 无冲突;不动 first_seen(分区键),不触发分区表 UPDATE 限制。
func Migrate190NormalizeMACHistory(db *gorm.DB) error {
	log.Println("Running migration 190: 归一化 sys_device_mac_history 的 mac_address + interface_name")

	if !isPostgreSQL(db) {
		log.Println("Migration 190 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// ========== Step 1: mac_address → 大写+冒号 ==========
	// 1.1 剥所有分隔符(. : -) + UPPER(仅对 12/14/17 字符 hex 串)
	result := db.Exec(`
UPDATE sys_device_mac_history
   SET mac_address = UPPER(REGEXP_REPLACE(mac_address, '[.:\-]', '', 'g'))
 WHERE mac_address ~ '^[0-9a-fA-F.:\-]+$'
   AND length(REGEXP_REPLACE(mac_address, '[.:\-]', '', 'g')) = 12
`)
	if result.Error != nil {
		log.Printf("Migration 190 step1.1 (UPPER+剥分隔符) 失败: %v", result.Error)
		return result.Error
	}
	log.Printf("Migration 190 step1.1 影响 %d 行", result.RowsAffected)

	// 1.2 12 字符串插冒号 → AA:BB:CC:DD:EE:FF
	result = db.Exec(`
UPDATE sys_device_mac_history
   SET mac_address = (
        SUBSTRING(mac_address, 1, 2)  || ':' ||
        SUBSTRING(mac_address, 3, 2)  || ':' ||
        SUBSTRING(mac_address, 5, 2)  || ':' ||
        SUBSTRING(mac_address, 7, 2)  || ':' ||
        SUBSTRING(mac_address, 9, 2)  || ':' ||
        SUBSTRING(mac_address, 11, 2)
       )
 WHERE length(mac_address) = 12
   AND mac_address ~ '^[0-9A-F]{12}$'
`)
	if result.Error != nil {
		log.Printf("Migration 190 step1.2 (插冒号) 失败: %v", result.Error)
		return result.Error
	}
	log.Printf("Migration 190 step1.2 影响 %d 行", result.RowsAffected)

	// ========== Step 2: interface_name → 大写短名 ==========
	// 2.1 先去所有空格(华为/锐捷部分输出 "GigabitEthernet 1/0/1" 含空格)
	result = db.Exec(`
UPDATE sys_device_mac_history
   SET interface_name = REGEXP_REPLACE(interface_name, '\s+', '', 'g')
 WHERE interface_name ~ '\s'
`)
	if result.Error != nil {
		log.Printf("Migration 190 step2.1 (去空格) 失败(非阻断): %v", result.Error)
	} else {
		log.Printf("Migration 190 step2.1 (去空格) 影响 %d 行", result.RowsAffected)
	}

	// 2.2 全称 → 短名(CASE WHEN,覆盖华为/Cisco 全称,与 M187 Step1 对齐)
	result = db.Exec(`
UPDATE sys_device_mac_history
   SET interface_name = CASE
     WHEN interface_name ILIKE 'GigabitEthernet%'    THEN 'GE'   || SUBSTRING(interface_name, 16)
     WHEN interface_name ILIKE 'TenGigabitEthernet%' THEN 'XGE'  || SUBSTRING(interface_name, 19)
     WHEN interface_name ILIKE 'TenGigE%'            THEN 'XGE'  || SUBSTRING(interface_name, 8)
     WHEN interface_name ILIKE 'TwentyFiveGigE%'     THEN 'TWE'  || SUBSTRING(interface_name, 15)
     WHEN interface_name ILIKE 'HundredGigE%'        THEN 'HGE'  || SUBSTRING(interface_name, 12)
     WHEN interface_name ILIKE 'FortyGigE%'          THEN 'FOE'  || SUBSTRING(interface_name, 10)
     WHEN interface_name ILIKE 'FastEthernet%'       THEN 'FE'   || SUBSTRING(interface_name, 13)
     WHEN interface_name ILIKE 'Loopback%'           THEN 'Loop' || SUBSTRING(interface_name, 9)
     ELSE interface_name
   END
 WHERE interface_name ILIKE 'GigabitEthernet%'
    OR interface_name ILIKE 'TenGigabitEthernet%'
    OR interface_name ILIKE 'TenGigE%'
    OR interface_name ILIKE 'TwentyFiveGigE%'
    OR interface_name ILIKE 'HundredGigE%'
    OR interface_name ILIKE 'FortyGigE%'
    OR interface_name ILIKE 'FastEthernet%'
    OR interface_name ILIKE 'Loopback%'
`)
	if result.Error != nil {
		log.Printf("Migration 190 step2.2 (全称→短名) 失败: %v", result.Error)
		return result.Error
	}
	log.Printf("Migration 190 step2.2 (全称→短名) 影响 %d 行", result.RowsAffected)

	// ========== Step 3: DELETE 垃圾行(mac_address 非标准格式) ==========
	// 设备 display mac-address 输出的表头/汇总行/注释行曾误解析入库:
	//   mac_address = 'Flags:' / 'Total' / '#' / 'Invalid' / 'forwarding...' 等
	// 这些非 hex 字符串无法归一化(只能 DELETE)。根源已在 parseMACLine 加 isCanonicalMAC
	// 校验(防新垃圾),本步清历史已入库的(实测 sys_device_mac_history_2026_07 有大量此类行)。
	result = db.Exec(`
DELETE FROM sys_device_mac_history
 WHERE mac_address !~ '^[0-9A-F]{2}(:[0-9A-F]{2}){5}$'
`)
	if result.Error != nil {
		log.Printf("Migration 190 step3 (删垃圾) 失败(非阻断): %v", result.Error)
	} else {
		log.Printf("Migration 190 step3 (删 mac_address 非标准垃圾) 影响 %d 行", result.RowsAffected)
	}

	log.Println("Migration 190 completed")
	return nil
}
