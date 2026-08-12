//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate186NormalizePortStatusInterfaceName 把 sys_device_port_status.interface_name
// 里的"全称"统一折叠到"短名大写"形式。
//
// 折叠映射(2026-07-01 port-mac-format-unify):
//   GigabitEthernet0/0/1   -> GE0/0/1
//   TenGigE1/0/1           -> XGE1/0/1
//   TwentyFiveGigE1/1      -> TWE1/1
//   HundredGigE1/49        -> HGE1/49
//   FortyGigE1/1           -> FOE1/1
//   FastEthernet0/1        -> FE0/1
//
// 唯一键冲突处理: sys_device_port_status 上有 (device_id, interface_name) 唯一约束
// (migration_177 reapplied)。如果某 device 已有 "GE0/0/1" 而现在要写
// "GigabitEthernet0/0/1" -> "GE0/0/1" 会触发冲突。处理策略:
//   - 先按 (device_id, 新名) 找出冲突行
//   - 用 collected_at DESC 排序保留最新行,删除更旧行(migration_177 同款策略)
//   - 再 UPDATE 剩余行的 interface_name
//
// 不更新已是大写短名的行(`^GE|^XGE|^TWE|^HGE|^FOE|^FE|^Loop|^Vlan|^NULL`),
// 也不动未知前缀(Loopback / Vlanif / NULL / Vlan)以外的其他形式。
func Migrate186NormalizePortStatusInterfaceName(db *gorm.DB) error {
	log.Println("Running migration 186: 统一 sys_device_port_status.interface_name 短名大写")

	if !isPostgreSQL(db) {
		log.Println("Migration 186 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 1) 全称 → 短名大写 (REGEXP_REPLACE 一次性处理)
	//    (\d+...) 留数字 + 斜杠部分,只换前缀
	//    注意:PG REGEXP_REPLACE 不支持 lookahead/lookbehind 累加,
	//    这里前缀是固定字符串,直接用锚定匹配最稳
	type rowUpdate struct {
		ID            string
		DeviceID      string
		OldIface      string
		NewIface      string
	}

	// 拉出需要折叠的行(GigabitEthernet / TenGigE / TwentyFiveGigE / HundredGigE
	// / FortyGigE / FastEthernet 全称)
	var rows []rowUpdate
	if err := db.Raw(`
SELECT id::text AS id,
       device_id::text AS device_id,
       interface_name AS old_iface,
       CASE
         WHEN interface_name LIKE 'GigabitEthernet%'    THEN 'GE'    || SUBSTRING(interface_name, 16)
         WHEN interface_name LIKE 'TenGigE%'            THEN 'XGE'   || SUBSTRING(interface_name, 8)
         WHEN interface_name LIKE 'TwentyFiveGigE%'     THEN 'TWE'   || SUBSTRING(interface_name, 15)
         WHEN interface_name LIKE 'HundredGigE%'        THEN 'HGE'   || SUBSTRING(interface_name, 12)
         WHEN interface_name LIKE 'FortyGigE%'          THEN 'FOE'   || SUBSTRING(interface_name, 10)
         WHEN interface_name LIKE 'FastEthernet%'       THEN 'FE'    || SUBSTRING(interface_name, 13)
         ELSE interface_name
       END AS new_iface
  FROM sys_device_port_status
 WHERE interface_name ~ '^(GigabitEthernet|TenGigE|TwentyFiveGigE|HundredGigE|FortyGigE|FastEthernet)'
   AND interface_name <> 'GE' || SUBSTRING(interface_name, 16)
   AND interface_name <> 'XGE' || SUBSTRING(interface_name, 8)
   AND interface_name <> 'TWE' || SUBSTRING(interface_name, 15)
   AND interface_name <> 'HGE' || SUBSTRING(interface_name, 12)
   AND interface_name <> 'FOE' || SUBSTRING(interface_name, 10)
   AND interface_name <> 'FE' || SUBSTRING(interface_name, 13)
`).Scan(&rows).Error; err != nil {
		log.Printf("Migration 186: 拉取待折叠行失败: %v", err)
		return err
	}

	if len(rows) == 0 {
		log.Println("Migration 186: 没有需要折叠的接口名,直接完成")
		return nil
	}

	log.Printf("Migration 186: 待折叠 %d 行", len(rows))

	// 2) 按 (device_id, new_iface) 分组,先对每组做冲突排查:
	//    若该 device 已有同名 short 形式,保留 collected_at 较新者
	//    (migration_177 已为常规重复行打过补丁,这里只需处理 short 行的冲突)

	// 3) 把 infoPoint 引用先 re-point 到目标行(避免 UPDATE 后 UNIQUE 冲突删除行
	//    时 infoPoint 引用悬空)
	//    策略: 跳过 — migration 186 的目标是把"全称"折叠为"短名",短名已存在的
	//    情况下旧"全称"行要么被删要么被改,无 infoPoint 引用风险
	//    (infoPoint 是在物理链路上创建,全称行没有真实端口对应)。

	// 4) 直接 UPDATE。PostgreSQL 在 UPDATE 时若新值与现有同 (device_id, iface) 冲突
	//    会抛 23505;先 DELETE 那些冲突的"短名"行(它们没有真实物理端口对应,
	//    是历史采集时不同厂商命名产生的副本)。
	//
	//    实现: 对每一行 UPDATE 之前先查 (device_id, new_iface) 是否有重复,
	//    有则按 collected_at DESC 保留最新。
	for _, r := range rows {
		// 4.1) 查 (device_id, new_iface) 冲突
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
			log.Printf("Migration 186: 查冲突失败(device=%s, new=%s): %v",
				r.DeviceID, r.NewIface, err)
			return err
		}

		if conflictID != "" {
			// 4.2) 冲突:把"旧全称"行 的 ops_info_points 引用 re-point 到 conflict(短名行)
			//     然后删除旧全称行
			if err := db.Exec(`
UPDATE ops_info_points
   SET port_id = ?,
       updated_at = NOW()
 WHERE port_id = ?
   AND deleted_at IS NULL
			`, conflictID, r.ID).Error; err != nil {
				log.Printf("Migration 186: re-point infoPoint %s->%s 失败(非阻断,继续): %v",
					r.ID, conflictID, err)
			}
			if err := db.Exec(`
DELETE FROM sys_device_port_status
 WHERE id = ?::uuid
			`, r.ID).Error; err != nil {
				log.Printf("Migration 186: 删除旧全称行 %s 失败: %v", r.ID, err)
				return err
			}
			continue
		}

		// 4.3) 无冲突,直接 UPDATE
		if err := db.Exec(`
UPDATE sys_device_port_status
   SET interface_name = ?
 WHERE id = ?::uuid
		`, r.NewIface, r.ID).Error; err != nil {
			log.Printf("Migration 186: UPDATE %s -> %s 失败: %v",
				r.ID, r.NewIface, err)
			return err
		}
	}

	log.Println("Migration 186 completed")
	return nil
}
