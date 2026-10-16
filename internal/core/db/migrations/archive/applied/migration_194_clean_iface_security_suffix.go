//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate194CleanIfaceSecuritySuffix 清理三表 interface_name 的尾部字母垃圾后缀
// (GE1/0/4SECURITY → GE1/0/4),并幂等重跑去空格 / 全称转短名 / MAC 大写冒号,
// 覆盖生产曾跑旧二进制的残留。
//
// 背景(2026-07-02 mac-iface-security-suffix):
//   - 华为 display mac-address 输出 security 类型 MAC 时,Learned-From 列接口名粘连
//     security 标记(无空格),normalize.InterfaceName 的守卫曾直接 ToUpper 放行尾部,
//     产生 GE1/0/4SECURITY / GE1/0/4security 入库。
//   - normalize 已修(pkg/normalize/iface.go 守卫命中后用 shortIfaceBodyPattern 剥离尾部),
//     本迁移清理历史已入库脏数据。
//   - 同时幂等清理带空格 / 全称 / 小写 MAC(覆盖生产曾跑 17459ec9 之前二进制的残留,
//     与 M184/M187/M190 互补,重跑无副作用)。
//
// 三表: sys_device_mac_address / sys_device_mac_history / sys_device_port_status
//
// 安全:
//   - mac_address / mac_history 无 (device, interface_name) 唯一约束,直接 UPDATE。
//   - port_status 有 (device_id, interface_name) 唯一约束,剥离后缀产生冲突极罕见
//     (真实接口名本就该是无后缀形式),直接 UPDATE,冲突记日志。
//   - Step C 的 UPPER 只作用于物理接口前缀(GE/XGE/...),不影响逻辑接口(Vlanif/Loop 等)。
func Migrate194CleanIfaceSecuritySuffix(db *gorm.DB) error {
	log.Println("Running migration 194: 清理 interface_name SECURITY 后缀 + 幂等重归一化")

	if !isPostgreSQL(db) {
		log.Println("Migration 194 skipped (non-PostgreSQL dialect)")
		return nil
	}

	ifaceTables := []string{"sys_device_mac_address", "sys_device_mac_history", "sys_device_port_status"}

	for _, table := range ifaceTables {
		log.Printf("[194] === %s ===", table)

		// Step A: 去空格(覆盖 "GigabitEthernet 0/2" 带空格残留)
		r := db.Exec(`UPDATE ` + table + ` SET interface_name = REGEXP_REPLACE(interface_name, '\s+', '', 'g') WHERE interface_name ~ '\s'`)
		if r.Error != nil {
			log.Printf("[194] %s 去空格失败(非阻断): %v", table, r.Error)
		} else {
			log.Printf("[194] %s 去空格影响 %d 行", table, r.RowsAffected)
		}

		// Step B: 全称 → 短名(幂等,与 M187/M190 同款 CASE WHEN)
		r = db.Exec(`
UPDATE ` + table + `
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
		if r.Error != nil {
			log.Printf("[194] %s 全称→短名失败(非阻断): %v", table, r.Error)
		} else {
			log.Printf("[194] %s 全称→短名影响 %d 行", table, r.RowsAffected)
		}

		// Step C: 剥离尾部字母垃圾(GE1/0/4SECURITY → GE1/0/4)— 本次核心新增
		// 匹配"物理接口短名+数字+数字段"主体后跟字母,保留主体并 UPPER(物理前缀全大写安全)。
		//   - GE1/0/4SECURITY / GE1/0/4security / ge1/0/4security → GE1/0/4
		//   - GE0/0/1(无后缀)不命中 WHERE,不动
		//   - GE0/0/1.5(子接口,数字结尾)不命中 WHERE,不动
		//   - GEet5/34(GE 后非数字)不命中守卫,不动(与 normalize 一致)
		r = db.Exec(`
UPDATE ` + table + `
   SET interface_name = UPPER(REGEXP_REPLACE(interface_name, '^((?:GE|XGE|TWE|HGE|FOE|FE|ET)[0-9][0-9/.:]*).*$', '\1', 'i'))
 WHERE interface_name ~* '^(GE|XGE|TWE|HGE|FOE|FE|ET)[0-9][0-9/.:]*[A-Za-z]'
`)
		if r.Error != nil {
			log.Printf("[194] %s 剥离尾部字母失败(非阻断): %v", table, r.Error)
		} else {
			log.Printf("[194] %s 剥离尾部字母影响 %d 行", table, r.RowsAffected)
		}
	}

	// ========== Step D: mac_address / mac_history MAC → 大写冒号(幂等) ==========
	for _, macTable := range []string{"sys_device_mac_address", "sys_device_mac_history"} {
		// D.1 剥分隔符(. : -) + UPPER(仅 12 字符 hex)
		r := db.Exec(`
UPDATE ` + macTable + `
   SET mac_address = UPPER(REGEXP_REPLACE(mac_address, '[.:\-]', '', 'g'))
 WHERE mac_address ~ '^[0-9a-fA-F.:\-]+$'
   AND length(REGEXP_REPLACE(mac_address, '[.:\-]', '', 'g')) = 12
`)
		if r.Error != nil {
			log.Printf("[194] %s MAC UPPER 失败(非阻断): %v", macTable, r.Error)
		} else {
			log.Printf("[194] %s MAC UPPER 影响 %d 行", macTable, r.RowsAffected)
		}

		// D.2 12 字符串插冒号 → AA:BB:CC:DD:EE:FF
		r = db.Exec(`
UPDATE ` + macTable + `
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
		if r.Error != nil {
			log.Printf("[194] %s MAC 插冒号失败(非阻断): %v", macTable, r.Error)
		} else {
			log.Printf("[194] %s MAC 插冒号影响 %d 行", macTable, r.RowsAffected)
		}
	}

	log.Println("Migration 194 completed")
	return nil
}
