//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate185NormalizeARPEntriesToUppercase 把 sys_device_arp_entry 表里
// 所有小写 MAC 统一提升为大写+冒号格式(AA:BB:CC:DD:EE:FF)。
//
// 背景(2026-07-01 port-mac-format-unify):
//   - 旧 arp_collector 只换分隔符(.:-)→ 冒号,不动大小写,导致
//     3 种格式混存(00:e0:fc:12:34:56 / 00:E0:FC:12:34:56 / 00e0fc123456)
//   - 写入分裂 + 大小写分裂,跨表 LIKE 漏查
//
// 表名修正(2026-07-01 hotfix #1): GORM 模型 `models.DeviceARPEntry` 显式
// TableName() 返回 "sys_device_arp_entry"(**单数**),不是
// `sys_device_arp_entries`(GORM 默认复数规则不适用此处)。
// 修复前错误引用复数表名导致 SQLSTATE 42P01 relation does not exist。
//
// 表存在性兜底(2026-07-01 hotfix #2): 实测发现 `models.DeviceARPEntry` **未注册到
// `internal/core/db/database.go::AutoMigrate`**(2026-07-01 grep 验证),意味着
// `sys_device_arp_entry` 表在本仓库生产 DB 中**根本不存在**(若运行过 arp 采集
// 也会因 GORM 缺表报错)。盲目 UPDATE 会再抛 42P01,改成 "HasTable 检查 → 跳过"。
//
// 实现: 复用 migration_184 的两步策略
//   1) UPPER 提大小写
//   2) length=12 的 12 字符串插冒号
func Migrate185NormalizeARPEntriesToUppercase(db *gorm.DB) error {
	log.Println("Running migration 185: 统一 sys_device_arp_entry 大写+冒号")

	if !isPostgreSQL(db) {
		log.Println("Migration 185 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// hotfix #2: 表存在性兜底。DeviceARPEntry 未在 AutoMigrate 列表,生产 DB 无此表。
	// HasTable 返回 false → 直接 no-op,不再尝试 UPDATE 触发 42P01。
	if !db.Migrator().HasTable("sys_device_arp_entry") {
		log.Println("Migration 185: sys_device_arp_entry 表不存在(DeviceARPEntry 未在 AutoMigrate 中),跳过")
		return nil
	}

	// 第一步: UPPER 转换 length=12 或 17 的小写行
	result := db.Exec(`
UPDATE sys_device_arp_entry
   SET mac_address = UPPER(mac_address)
 WHERE mac_address ~ '[a-f]'
   AND length(mac_address) IN (12, 17)
`)
	if result.Error != nil {
		return result.Error
	}
	log.Printf("Migration 185: UPPER 完成,影响 %d 行(小写 12/17 字符串)", result.RowsAffected)

	// 第二步: 12 字符串插冒号
	result = db.Exec(`
UPDATE sys_device_arp_entry
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
	log.Printf("Migration 185: 12 字符串插冒号完成,影响 %d 行", result.RowsAffected)

	log.Println("Migration 185 completed")
	return nil
}
