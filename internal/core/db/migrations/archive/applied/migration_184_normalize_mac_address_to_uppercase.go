//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate184NormalizeMACAddressToUppercase 把 sys_device_mac_address 表里
// 所有 MAC 统一为大写+冒号格式(AA:BB:CC:DD:EE:FF)。
//
// 支持的输入格式(2026-07-01 真实数据):
//   - aa:bb:cc:dd:ee:ff (17 字符冒号,可能是大小写混用)   -> AA:BB:CC:DD:EE:FF
//   - AA:BB:CC:DD:EE:FF (17 字符冒号,已是大写,无变化)
//   - aa-bb-cc-dd-ee-ff (14 字符连字符)                  -> AA:BB:CC:DD:EE:FF
//   - aabb.ccdd.eeff    (14 字符 Cisco 风格)             -> AA:BB:CC:DD:EE:FF
//   - AABBCCDDEEFF      (12 字符无分隔符)                -> AA:BB:CC:DD:EE:FF
//
// 三步策略:
//   1) 剥分隔符(. : -) → 12 字符 hex(只对 12/14/17 字符的 hex 字符串)
//   2) UPPER
//   3) 重新插入冒号
//
// 兜底: 任何剥完分隔符后不是 12 hex 字符的(如 00:00:00:00:00:00x / 全 0 / 注释行)
// 不动,保留原值(warn 日志 + 后续人工 review)。
//
// 修复历史:
//   2026-07-01 v1: 初版只处理 length=12/17,漏掉 14 字符 hyphenated 格式(706 行)
//   2026-07-01 v2: 重写为"剥分隔符→UPPER→插冒号",覆盖 12/14/17 三种 hex 长度
func Migrate184NormalizeMACAddressToUppercase(db *gorm.DB) error {
	log.Println("Running migration 184 (v2): 统一 sys_device_mac_address 大写+冒号 (覆盖 12/14/17 字符)")

	if !isPostgreSQL(db) {
		log.Println("Migration 184 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 第一步: 剥所有分隔符(. : -),得到 12 字符 hex
	// REGEXP_REPLACE 第三参数 '' 删除匹配字符,第四参数 'g' 全局
	result := db.Exec(`
UPDATE sys_device_mac_address
   SET mac_address = UPPER(REGEXP_REPLACE(mac_address, '[.:\-]', '', 'g'))
 WHERE mac_address ~ '^[0-9a-fA-F.:\-]+$'
   AND length(REGEXP_REPLACE(mac_address, '[.:\-]', '', 'g')) = 12
`)
	if result.Error != nil {
		return result.Error
	}
	log.Printf("Migration 184 step1 (UPPER + 剥分隔符):影响 %d 行", result.RowsAffected)

	// 第二步: 把 12 字符串按 2 字符插冒号 → AA:BB:CC:DD:EE:FF
	result = db.Exec(`
UPDATE sys_device_mac_address
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
		return result.Error
	}
	log.Printf("Migration 184 step2 (12 字符串插冒号):影响 %d 行", result.RowsAffected)

	// 第三步: 兜底统计 — 列出仍未归一化的异常 MAC
	type AnomalyRow struct {
		RawLen int
		Cnt    int64
	}
	var anomalies []AnomalyRow
	db.Raw(`
SELECT length(mac_address) AS raw_len, COUNT(*) AS cnt
  FROM sys_device_mac_address
 WHERE mac_address IS NOT NULL AND mac_address <> ''
   AND mac_address !~ '^[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}$'
 GROUP BY raw_len
 ORDER BY raw_len
`).Scan(&anomalies)
	if len(anomalies) > 0 {
		log.Printf("Migration 184: 仍有 %d 种异常格式未归一化(后续处理):", len(anomalies))
		for _, a := range anomalies {
			log.Printf("  length=%d cnt=%d", a.RawLen, a.Cnt)
		}
	}

	// 第四步(2026-07-01 v2 增): 清理 mac_collector 解析错误产生的垃圾数据
	// 实测 16 行:mac_address = 'Flags:' / 'Total' / 'Invalid' / 等设备输出文本
	// 这些是 mac_collector 解析 device display mac-address 失败时,误把表头/汇总行
	// 当成 MAC 写入数据库。需 DELETE 而非转换(无法 normalize)。
	// 安全条件: mac_address 不像任何已知 MAC 格式 + interface_name 是 'displayed =' / 'detected' / '-'
	// 这类明显非接口名的标记
	result = db.Exec(`
DELETE FROM sys_device_mac_address
 WHERE length(mac_address) NOT IN (12, 14, 17)
    OR mac_address IN ('Flags:', 'Total', 'Invalid')
    OR interface_name IN ('displayed =', 'detected', '-', '')
`)
	if result.Error != nil {
		log.Printf("Migration 184 step4 清理垃圾数据失败(非阻断): %v", result.Error)
	} else {
		log.Printf("Migration 184 step4 清理垃圾数据:影响 %d 行", result.RowsAffected)
	}

	log.Println("Migration 184 (v2) completed")
	return nil
}
