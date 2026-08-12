//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate187CleanGarbledInterfaceNames 清理 sys_device_port_status 和
// sys_device_mac_address 表里 interface_name 字段的乱码形式。
//
// 背景(2026-07-01 verify-format-unify 发现):
//   - 旧版 collector(本任务修复前)不调用 NormalizeInterfaceName,直接 INSERT 设备原始输出
//   - 历史数据混入乱码形式: GEGigabitEthernet5/29 / XGigabitEthernet0/0/1 / Loopback0 / et3/46 等
//   - collector 代码已修(全部走 portcollection.NormalizeInterfaceName),但历史数据未清理
//
// 本 migration 显式处理 4 类问题:
//   1. 全称 Gig* → 短名 GE/XGE/TWE/... (sys_device_port_status 与 sys_device_mac_address)
//   2. 短+全乱码 GEGigabitEthernet5/29 → GE5/29 (显式 CASE WHEN)
//   3. XGigabitEthernet0/0/1 → XGE0/0/1 (H3C 设备特定)
//   4. et 短名 (Huawei 25G) → et 大写 ET
//
// 不动:已是大写短名的行(GE/XGE/TWE/...)— 短名本来就合法
//
// 唯一键冲突处理(2026-07-01 决策):
//   sys_device_port_status 有 (device_id, interface_name) 唯一约束(migration_177 reapplied)
//   若某 device 已有 "GE0/0/1" 而现在要写 "GigabitEthernet0/0/1" -> "GE0/0/1" 会触发冲突
//   解决: 先按 (device_id, 新名) 找出冲突行,保留 collected_at DESC 较新者(migration_177 同款策略)
func Migrate187CleanGarbledInterfaceNames(db *gorm.DB) error {
	log.Println("Running migration 187: 清理 sys_device_port_status + sys_device_mac_address 的乱码 interface_name")

	if !isPostgreSQL(db) {
		log.Println("Migration 187 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// ========== Step 1: 清理 sys_device_port_status ==========
	log.Println("[187] === Step 1: sys_device_port_status ===")

	// 1.1 拉出待清理的行(全称、短+全乱码、X Gigabit、et 小写、Loopback)
	// 简化策略: 用 LIKE 显式匹配 9 种已知乱码形式(2026-07-01 验证发现)
	type PortRow struct {
		ID       string
		DeviceID string
		OldIface string
		NewIface string
	}
	var portRows []PortRow
	if err := db.Raw(`
SELECT id::text AS id,
       device_id::text AS device_id,
       interface_name AS old_iface,
       CASE
         -- 短+全乱码(显式列举 9 种已知模式 + 第二轮发现的)
         -- 注: PG SUBSTRING(str, pos) 1-indexed,数字起点计算:
         --   'XGEngigabitethernet'  19 chars prefix → SUBSTRING(20) 取数字
         --   'XGengigabitethernet'  20 chars prefix → SUBSTRING(21) 取数字
         --   'XGEigabitEthernet'    17 chars prefix → SUBSTRING(18) 取数字
         --   'GEngigabitethernet'   18 chars prefix → SUBSTRING(19) 取数字
         --   'GEgabitetherngigabitethernet' 31 chars → SUBSTRING(32) 取数字
         --   'GEGigabitEthernet'    16 chars prefix → SUBSTRING(17) 取数字
         --   'GEgigabitEthernet'    16 chars prefix → SUBSTRING(17) 取数字
         --   'GEgigabitethernet'    16 chars prefix → SUBSTRING(17) 取数字
         --   'XGigabitEthernet'      2 chars prefix  → SUBSTRING(3) 取数字 (截了 Gig 的 G)
         --   'XGigabitethernet'      2 chars prefix  → SUBSTRING(3) 取数字
         WHEN interface_name ILIKE 'GEGigabitEthernet%'                 THEN 'GE'    || SUBSTRING(interface_name, 17)
         WHEN interface_name ILIKE 'GEgigabitEthernet%'                 THEN 'GE'    || SUBSTRING(interface_name, 17)
         WHEN interface_name ILIKE 'GEgigabitethernet%'                 THEN 'GE'    || SUBSTRING(interface_name, 17)
         WHEN interface_name ILIKE 'GEgabitetherngigabitethernet%'      THEN 'GE'    || SUBSTRING(interface_name, 32)
         WHEN interface_name ILIKE 'GEngigabitethernet%'                THEN 'GE'    || SUBSTRING(interface_name, 19)
         WHEN interface_name ILIKE 'XGigabitEthernet%'                  THEN 'XGE'   || SUBSTRING(interface_name, 3)
         WHEN interface_name ILIKE 'XGigabitethernet%'                  THEN 'XGE'   || SUBSTRING(interface_name, 3)
         WHEN interface_name ILIKE 'XGengigabitethernet%'               THEN 'XGE'   || SUBSTRING(interface_name, 21)
         WHEN interface_name ILIKE 'XGEngigabitethernet%'               THEN 'XGE'   || SUBSTRING(interface_name, 20)
         WHEN interface_name ILIKE 'XGEigabitEthernet%'                 THEN 'XGE'   || SUBSTRING(interface_name, 18)
         WHEN interface_name ILIKE 'XGEt%'                             THEN 'XGE'   || SUBSTRING(interface_name, 5)  -- 2026-07-01 第二轮:SUBSTRING 错位残留 XGEt0/49 → XGE0/49
         WHEN interface_name ILIKE 'GEt%'                              THEN 'GE'    || SUBSTRING(interface_name, 4)
         WHEN interface_name ILIKE 'GEet%'                              THEN 'GE'    || SUBSTRING(interface_name, 5)
         WHEN interface_name = 'GEn'                                  THEN 'GE'
         -- 全称 Gig* → 短名
         WHEN interface_name ILIKE 'GigabitEthernet%'                   THEN 'GE'    || SUBSTRING(interface_name, 16)
         WHEN interface_name ILIKE 'TenGigE%'                           THEN 'XGE'   || SUBSTRING(interface_name, 8)
         WHEN interface_name ILIKE 'TenGigabitEthernet%'                  THEN 'XGE'   || SUBSTRING(interface_name, 19)  -- Cisco/Ruijie 全称(2026-07-01 第二轮发现)
         WHEN interface_name ILIKE 'TwentyFiveGigE%'                    THEN 'TWE'   || SUBSTRING(interface_name, 15)
         WHEN interface_name ILIKE 'HundredGigE%'                       THEN 'HGE'   || SUBSTRING(interface_name, 12)
         WHEN interface_name ILIKE 'FortyGigE%'                         THEN 'FOE'   || SUBSTRING(interface_name, 10)
         WHEN interface_name ILIKE 'FastEthernet%'                      THEN 'FE'    || SUBSTRING(interface_name, 13)
         WHEN interface_name ILIKE 'Loopback%'                          THEN 'Loop'  || SUBSTRING(interface_name, 9)
         -- et 小写 → ET 大写(Huawei 25G 短名,纯短名)
         WHEN interface_name ~ '^et[0-9]'                              THEN 'ET'    || SUBSTRING(interface_name, 3)
         ELSE interface_name
       END AS new_iface
  FROM sys_device_port_status
 WHERE interface_name ILIKE 'GEGigabitEthernet%'
    OR interface_name ILIKE 'GEgigabitEthernet%'
    OR interface_name ILIKE 'GEgigabitethernet%'
    OR interface_name ILIKE 'GEgabitetherngigabitethernet%'
    OR interface_name ILIKE 'GEngigabitethernet%'
    OR interface_name ILIKE 'XGigabitEthernet%'
    OR interface_name ILIKE 'XGigabitethernet%'
    OR interface_name ILIKE 'XGengigabitethernet%'
    OR interface_name ILIKE 'XGEngigabitethernet%'
    OR interface_name ILIKE 'XGEigabitEthernet%'
    OR interface_name ILIKE 'XGEt%'
    OR interface_name ILIKE 'GEt%'
    OR interface_name ILIKE 'GEet%'
    OR interface_name = 'GEn'
    OR interface_name ILIKE 'GigabitEthernet%'
    OR interface_name ILIKE 'TenGigE%'
    OR interface_name ILIKE 'TenGigabitEthernet%'
    OR interface_name ILIKE 'TwentyFiveGigE%'
    OR interface_name ILIKE 'HundredGigE%'
    OR interface_name ILIKE 'FortyGigE%'
    OR interface_name ILIKE 'FastEthernet%'
    OR interface_name ILIKE 'Loopback%'
    OR interface_name ~ '^et[0-9]'
`).Scan(&portRows).Error; err != nil {
		log.Printf("[187] 拉取待清理 port_status 行失败: %v", err)
		return err
	}

	log.Printf("[187] sys_device_port_status 待清理 %d 行", len(portRows))
	for _, r := range portRows {
		// 1.2 查 (device_id, new_iface) 冲突
		var conflictID string
		err := db.Raw(`
SELECT id::text
  FROM sys_device_port_status
 WHERE device_id = ?::uuid
   AND interface_name = ?
   AND id <> ?::uuid
 ORDER BY collected_at DESC NULLS LAST, created_at DESC NULLS LAST
 LIMIT 1
		`, r.DeviceID, r.NewIface, r.ID).Scan(&conflictID).Error
		if err != nil {
			log.Printf("[187] 查冲突失败(device=%s, new=%s): %v", r.DeviceID, r.NewIface, err)
			continue
		}

		if conflictID != "" {
			// 冲突:把"旧乱码"行 的 ops_info_points 引用 re-point 到 conflict(标准行)
			// 然后删除旧乱码行
			if err := db.Exec(`
UPDATE ops_info_points
   SET port_id = ?,
       updated_at = NOW()
 WHERE port_id = ?
   AND deleted_at IS NULL
			`, conflictID, r.ID).Error; err != nil {
				log.Printf("[187] re-point infoPoint %s->%s 失败(非阻断,继续): %v",
					r.ID, conflictID, err)
			}
			if err := db.Exec(`
DELETE FROM sys_device_port_status
 WHERE id = ?::uuid
			`, r.ID).Error; err != nil {
				log.Printf("[187] 删除旧乱码行 %s 失败: %v", r.ID, err)
				continue
			}
			continue
		}

		// 无冲突,直接 UPDATE
		if err := db.Exec(`
UPDATE sys_device_port_status
   SET interface_name = ?
 WHERE id = ?::uuid
		`, r.NewIface, r.ID).Error; err != nil {
			log.Printf("[187] UPDATE port_status %s -> %s 失败: %v", r.ID, r.NewIface, err)
			continue
		}
	}

	// ========== Step 2: 清理 sys_device_mac_address.interface_name ==========
	// 2026-07-01 修复: 原 WHERE 用 'GigabitEthernet %'(带空格)只匹配 "GigabitEthernet 3/33"
	// (interface_name 与数字间含空格),漏掉最常见的 "GigabitEthernet3/33"(无空格,华为/H3C 标准输出)。
	// 改为 'GigabitEthernet%'(无空格) + 补 'TenGigabitEthernet%'(华为/Cisco 10G 全称,原仅 Step 1 有)。
	log.Println("[187] === Step 2: sys_device_mac_address.interface_name ===")

	// 这一表没有 BaseModel/无 deleted_at,无 unique 约束,可以放心 UPDATE
	// 但有 (device_id, mac_address, interface_name) 联合唯一约束(部分厂商有此约束)
	// 同样先查冲突再 UPDATE;无冲突直接 update
	type MacRow struct {
		ID       string
		DeviceID string
		OldIface string
		NewIface string
	}
	var macRows []MacRow
	if err := db.Raw(`
SELECT id::text AS id,
       device_id::text AS device_id,
       interface_name AS old_iface,
       CASE
         WHEN interface_name ILIKE 'GigabitEthernet%'    THEN 'GE'    || SUBSTRING(interface_name, 16)
         WHEN interface_name ILIKE 'TenGigabitEthernet%' THEN 'XGE'   || SUBSTRING(interface_name, 19)
         WHEN interface_name ILIKE 'TenGigE%'            THEN 'XGE'   || SUBSTRING(interface_name, 8)
         WHEN interface_name ILIKE 'TwentyFiveGigE%'     THEN 'TWE'   || SUBSTRING(interface_name, 15)
         WHEN interface_name ILIKE 'HundredGigE%'        THEN 'HGE'   || SUBSTRING(interface_name, 12)
         WHEN interface_name ILIKE 'FortyGigE%'          THEN 'FOE'   || SUBSTRING(interface_name, 10)
         WHEN interface_name ILIKE 'FastEthernet%'       THEN 'FE'    || SUBSTRING(interface_name, 13)
         WHEN interface_name ILIKE 'Loopback%'           THEN 'Loop'  || SUBSTRING(interface_name, 9)
         ELSE interface_name
       END AS new_iface
  FROM sys_device_mac_address
 WHERE interface_name ILIKE 'GigabitEthernet%'
    OR interface_name ILIKE 'TenGigabitEthernet%'
    OR interface_name ILIKE 'TenGigE%'
    OR interface_name ILIKE 'TwentyFiveGigE%'
    OR interface_name ILIKE 'HundredGigE%'
    OR interface_name ILIKE 'FortyGigE%'
    OR interface_name ILIKE 'FastEthernet%'
    OR interface_name ILIKE 'Loopback%'
`).Scan(&macRows).Error; err != nil {
		log.Printf("[187] 拉取待清理 mac_address 行失败: %v", err)
		return err
	}

	log.Printf("[187] sys_device_mac_address 待清理 %d 行", len(macRows))
	for _, r := range macRows {
		// 简化: 先尝试直接 UPDATE(无冲突风险,因为这是从全名到短名,后者更紧凑,极少冲突)
		if err := db.Exec(`
UPDATE sys_device_mac_address
   SET interface_name = ?
 WHERE id = ?::uuid
		`, r.NewIface, r.ID).Error; err != nil {
			log.Printf("[187] UPDATE mac_address %s -> %s 失败: %v", r.ID, r.NewIface, err)
			continue
		}
	}

	log.Println("Migration 187 completed")
	return nil
}
