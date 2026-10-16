//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate177ReapplyPortStatusUnique 重新应用 sys_device_port_status 唯一约束
//
// 背景:
//   - 早期 migration 010_add_port_status_unique_constraint.sql 已被 archive,从未实际部署
//   - models.DevicePortStatus 当前也未声明 uniqueIndex
//   - 后果: port 采集使用 clause.OnConflict{device_id, interface_name} 做 UPSERT,但因 DB
//     无唯一约束,PG 不会触发 ON CONFLICT,每次采集都 INSERT 新行,导致同一 (device_id,
//     interface_name) 出现多行 port_status 记录
//   - 数据完整性破坏: 同一物理端口分散到多条 UUID,infoPoint.port_id 引用漂移,
//     ops_info_points→port_status→sys_device_mac_address 反查链间歇性失败
//
// 修复策略:
//   1. 把每组 (device_id, interface_name) 多行合并为一行(保留 collected_at 最新;
//      若有 ops_info_points 引用其中任一行,优先保留被引用的那行,避免悬挂引用)
//   2. 把 infoPoint.port_id 重新指向合并后保留的 UUID
//   3. 删除被合并的旧行
//   4. 添加 UNIQUE (device_id, interface_name) 约束
//   5. 给 models.DevicePortStatus 加 uniqueIndex:uniq_device_interface tag,
//      确保后续 AutoMigrate 走 ADD path 而非 DROP+ADD
func Migrate177ReapplyPortStatusUnique(db *gorm.DB) error {
	log.Println("Running migration 177: Reapply sys_device_port_status unique constraint")

	if !db.Migrator().HasTable("sys_device_port_status") {
		log.Println("Table sys_device_port_status does not exist yet, skipping migration 177")
		return nil
	}

	// 检查唯一约束 / 唯一索引是否已存在
	// GORM `uniqueIndex` tag 会创建 unique INDEX(在 pg_class),而非 pg_constraint
	// 入口;两种形态任一存在即跳过 ADD CONSTRAINT(都已强制唯一性)
	var uniqueExists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint
			 WHERE conname = 'uniq_device_interface'
			   AND conrelid = 'sys_device_port_status'::regclass
		) OR EXISTS (
			SELECT 1 FROM pg_class c
			 WHERE c.relname = 'uniq_device_interface'
			   AND c.relkind = 'i'
			   AND c.relnamespace = 'public'::regnamespace
			   AND EXISTS (
			       SELECT 1 FROM pg_index i
			        WHERE i.indexrelid = c.oid
			          AND i.indisunique
			   )
		)
	`).Scan(&uniqueExists).Error
	if err != nil {
		log.Printf("Migration 177: 检查唯一性存在性失败 (忽略,继续): %v", err)
	}

	// Step 1: 为每组 (device_id, interface_name) 选出"保留行"——优先选被 infoPoint 引用的
	//         其次选 collected_at 最新的
	type portPickRow struct {
		DeviceID      string `gorm:"column:device_id"`
		InterfaceName string `gorm:"column:interface_name"`
		KeepID        string `gorm:"column:keep_id"`
		DropIDs       string `gorm:"column:drop_ids"`
		DropCount     int    `gorm:"column:drop_count"`
	}
	var picks []portPickRow
	if err := db.Raw(`
		WITH ranked AS (
			SELECT
				p.device_id,
				p.interface_name,
				p.id,
				p.collected_at,
				ROW_NUMBER() OVER (
					PARTITION BY p.device_id, p.interface_name
					ORDER BY
						-- 优先:有 infoPoint 引用
						CASE WHEN EXISTS (
							SELECT 1 FROM ops_info_points ip
							WHERE ip.port_id::text = p.id::text
							  AND ip.deleted_at IS NULL
						) THEN 0 ELSE 1 END,
						-- 其次:collected_at 最新
						p.collected_at DESC NULLS LAST,
						-- 再后:created_at 最新
						p.created_at DESC NULLS LAST,
						-- 兜底:id 字典序最大
						p.id DESC
				) AS rn
			FROM sys_device_port_status p
		),
		keep_set AS (
			SELECT device_id, interface_name, id AS keep_id
			FROM ranked WHERE rn = 1
		),
		drop_set AS (
			SELECT device_id, interface_name,
			       string_agg(id::text, ',' ORDER BY id::text) AS drop_ids,
			       COUNT(*) AS drop_count
			FROM ranked WHERE rn > 1
			GROUP BY device_id, interface_name
		)
		SELECT k.device_id, k.interface_name, k.keep_id, d.drop_ids, d.drop_count
		  FROM keep_set k
		  JOIN drop_set d
		    ON d.device_id = k.device_id AND d.interface_name = k.interface_name
	`).Scan(&picks).Error; err != nil {
		log.Printf("Migration 177: 计算保留行失败: %v", err)
		return err
	}

	totalMerged := 0
	totalRepointed := 0
	for _, p := range picks {
		if p.DropCount == 0 {
			continue
		}
		totalMerged++

		// Step 2: 把 infoPoint 引用从 drop_ids 列表里的 UUID 改到 keep_id
		// GORM 不支持 string list IN,用 ANY 字符串匹配需要 array 转换,简化用单值循环
		// drop_ids 形如 "uuid1,uuid2,uuid3"
		res := db.Exec(`
			UPDATE ops_info_points
			   SET port_id = ?::uuid,
			       updated_at = NOW()
			 WHERE port_id::text = ANY (string_to_array(?, ','))
			   AND deleted_at IS NULL
		`, p.KeepID, p.DropIDs)
		if res.Error != nil {
			log.Printf("Migration 177: re-point infoPoint for (%s, %s) failed (non-fatal,continue): %v",
				p.DeviceID, p.InterfaceName, res.Error)
		} else {
			totalRepointed += int(res.RowsAffected)
		}
	}

	log.Printf("Migration 177: 合并 %d 组 (device_id, interface_name) 重复行,重新指向 %d 条 infoPoint 引用",
		totalMerged, totalRepointed)

	// Step 3: 删除重复行(只保留 ranked.rn = 1)
	if err := db.Exec(`
		WITH ranked AS (
			SELECT id,
			       ROW_NUMBER() OVER (
			           PARTITION BY device_id, interface_name
			           ORDER BY
			               CASE WHEN EXISTS (
			                   SELECT 1 FROM ops_info_points ip
			                   WHERE ip.port_id::text = sys_device_port_status.id::text
			                     AND ip.deleted_at IS NULL
			               ) THEN 0 ELSE 1 END,
			               collected_at DESC NULLS LAST,
			               created_at DESC NULLS LAST,
			               id DESC
			       ) AS rn
			FROM sys_device_port_status
		)
		DELETE FROM sys_device_port_status
		 WHERE id IN (SELECT id FROM ranked WHERE rn > 1)
	`).Error; err != nil {
		log.Printf("Migration 177: 删除重复行失败: %v", err)
		return err
	}

	// Step 4: 添加唯一约束(若不存在)
	if !uniqueExists {
		if err := db.Exec(`
			ALTER TABLE sys_device_port_status
			ADD CONSTRAINT uniq_device_interface
			UNIQUE (device_id, interface_name)
		`).Error; err != nil {
			log.Printf("Migration 177: 添加唯一约束失败: %v", err)
			return err
		}
		log.Println("Migration 177: 已添加 uniq_device_interface 唯一约束")
	} else {
		log.Println("Migration 177: uniq_device_interface 唯一性已存在(约束或 unique index),跳过 ADD CONSTRAINT")
	}

	log.Println("Migration 177 completed successfully")
	return nil
}
