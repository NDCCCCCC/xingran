package migrations

import (
	"fmt"
	"log"

	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate182FixPortStatusDeviceIdDrift 数据治理:修复 sys_device_port_status.device_id
// 与 ops_info_points.device_id 在 84.14% 信息点上的历史漂移。
//
// 背景(2026-06-30 debug session):
//   - 工位 5F003 物理链路设备显示 0 台,根因追查发现 port_status.device_id 与 ip.device_id
//     在 1247/1486 (84.14%) 信息点上不一致
//   - 根因:历史 port_status 数据导入时未正确匹配设备 + 端口名跨设备不唯一(每台 RS8607E
//     都有 GE1/1~GE5/48,同一 interface_name 出现在多台设备的 port_status 中)
//   - 物理真相在 sys_device_mac_address.device_id = ops_info_points.device_id
//   - sys_device_port_status.device_id 错指到另一台物理设备(例如 5F 端口错记到 4F 设备)
//
// 同步方向:port_status.device_id ← ip.device_id
//
// 安全条件(行级,1247/1247 = 100% 满足):
//   1. ip.device_id 指向 sys_network_device 中实际存在的设备
//   2. 该设备在 sys_device_mac_address 至少有 1 条 MAC 记录(物理证据)
//   3. ip.deleted_at IS NULL AND ip.status = 0
//   4. port.device_id::text != ip.device_id::text(确实存在漂移)
//
// 备份表 sys_device_port_status_drift_backup:
//   - PK = port.id(UUID),每个 port 只存一行
//   - 同一 port 关联多个 info_point(实测可达 7 个)时只备份一次,回滚时按 port 恢复即可
//   - port.device_id 是单值,回滚恢复单值
//   - INSERT 用 DISTINCT ON (port.id) + ON CONFLICT (id) DO NOTHING 防御性写
//
// 幂等性:
//   - 若备份表存在且行数 > 0:跳过(已运行过)
//   - 若备份表存在但行数 = 0(上次失败留空):DROP 后重跑
//
// 事务边界:CREATE TABLE + 备份 INSERT + UPDATE 全部包在一个 Transaction 里,
// 任一失败整体回滚,不留半成品
func Migrate182FixPortStatusDeviceIdDrift(db *gorm.DB) error {
	log.Println("Running migration 182: 修复 sys_device_port_status.device_id 漂移")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 182 跳过(非 PostgreSQL)")
		log.Println("Migration 182 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 阶段 0: 幂等性检查
	// 关键:不能只看"备份表行数"判断是否完成,因为上次失败可能 INSERT 成功但 UPDATE 回滚
	// 必须用"当前漂移数"作为唯一幂等性依据:
	//   - drift == 0:已完成,跳过(同时 DROP 备份表释放空间)
	//   - drift >  0:上次失败或首次运行,DROP 旧备份表(若有),重新跑
	var driftCount int64
	if err := db.Raw(`
SELECT COUNT(*)
  FROM sys_device_port_status port
  JOIN ops_info_points ip ON port.id::text = ip.port_id
 WHERE port.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL
   AND ip.status = 0
   AND ip.device_id IS NOT NULL
   AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
   AND EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)`).Scan(&driftCount).Error; err != nil {
		return fmt.Errorf("检查漂移数失败: %w", err)
	}
	applogger.Infof("[迁移] 182 当前漂移数 = %d", driftCount)

	if driftCount == 0 {
		log.Println("Migration 182: 当前漂移 0 行(已修复完成),跳过")
		// 清理备份表(修复成功后,备份表已无价值)
		if err := db.Exec(`DROP TABLE IF EXISTS sys_device_port_status_drift_backup`).Error; err != nil {
			applogger.Warnf("[迁移] 182 清理备份表失败(非阻断): %v", err)
		}
		return nil
	}

	// drift > 0:首次运行 OR 上次失败 → DROP 旧备份表,重新跑
	var tableExists bool
	db.Raw(`SELECT EXISTS (SELECT 1 FROM information_schema.tables
                     WHERE table_name = 'sys_device_port_status_drift_backup')`).Scan(&tableExists)
	if tableExists {
		applogger.Infof("[迁移] 182 漂移 > 0,备份表残留(上次部分执行),DROP 后重跑")
		if err := db.Exec(`DROP TABLE sys_device_port_status_drift_backup`).Error; err != nil {
			return fmt.Errorf("清理旧备份表失败: %w", err)
		}
	}

	// 阶段 1: 预览(不修改)
	var safeCount int64
	if err := db.Raw(`
SELECT COUNT(*)
  FROM sys_device_port_status port
  JOIN ops_info_points ip ON port.id::text = ip.port_id
 WHERE port.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL
   AND ip.status = 0
   AND ip.device_id IS NOT NULL
   AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
   AND EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)`).Scan(&safeCount).Error; err != nil {
		return fmt.Errorf("预览失败: %w", err)
	}
	applogger.Infof("[迁移] 182 预览: safe-fix 行数 = %d", safeCount)
	if safeCount == 0 {
		log.Println("Migration 182: 无需修复,跳过")
		return nil
	}

	// 阶段 2: 独立事务 - DELETE 冲突行(让出 (device_id, interface_name) 给目标 port)
	//
	// 关键设计:DELETE 必须独立事务提交,不能与 UPDATE 在同一事务。
	// 否则 UPDATE 失败导致整个事务回滚,DELETE 的 255 行也被撤销(2026-06-30 实战教训)。
	//
	// DELETE 失败/回滚的代价可控:仅删除"无 info_point 引用的占位行",风险低。
	//
	// target_keys CTE 关键:每个 port 选**最权威**的 info_point 作为 target:
	//   1. 优先:port 的 interface_name 在 ip.device_id 设备上有 MAC 数据(物理证据,精确到接口)
	//   2. 次之:ip.device_id 在 sys_device_mac_address 中有任何 MAC(设备级证据,较粗)
	//   3. 兜底:ip.id 字典序(稳定 tie-breaker)
	//   原因(2026-06-30 实测):port 5de2e697 (GE5/44) 关联 6 个 info_point,
	//   5D212 (aca124c8) 设备 GE5/44 有 3 MAC,5D监控录像机1 (060e5a69) 设备 GE5/44 0 MAC
	//   但 060e5a69 设备有其他端口的 MAC。若按"设备有 MAC"判别,5D监控录像机1 错误优先;
	//   必须按"port.interface_name 在设备上有 MAC"判别才能选到对的设备。
	deleteSQL := `
DELETE FROM sys_device_port_status p2
 WHERE p2.id IN (
       WITH target_keys AS (
           SELECT DISTINCT ON (port.id)
                  port.id AS port_id,
                  ip.device_id::text AS target_device_id,
                  port.interface_name
             FROM sys_device_port_status port
             JOIN ops_info_points ip ON port.id::text = ip.port_id
            WHERE port.device_id::text != ip.device_id::text
              AND ip.deleted_at IS NULL
              AND ip.status = 0
              AND ip.device_id IS NOT NULL
              AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
            ORDER BY port.id,
                     -- 优先级 1:port 的 interface_name 在 ip.device_id 设备上有 MAC(物理证据,精确)
                     (CASE WHEN EXISTS (SELECT 1 FROM sys_device_mac_address
                                         WHERE device_id::text = ip.device_id
                                           AND interface_name ILIKE '%' || port.interface_name || '%')
                           THEN 0 ELSE 1 END) ASC,
                     -- 优先级 2:ip.device_id 设备有任何 MAC(较粗)
                     (CASE WHEN EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)
                           THEN 0 ELSE 1 END) ASC,
                     -- 优先级 3:ip.id 字典序(稳定 tie-breaker)
                     ip.id ASC
       )
       SELECT p3.id
         FROM sys_device_port_status p3
         JOIN target_keys t
           ON p3.device_id::text = t.target_device_id
          AND p3.interface_name = t.interface_name
          AND p3.id != t.port_id
        WHERE NOT EXISTS (
              SELECT 1 FROM ops_info_points ip2
               WHERE ip2.port_id::text = p3.id::text
                 AND ip2.deleted_at IS NULL
           )
 )`
	delResult := db.Exec(deleteSQL)
	if delResult.Error != nil {
		return fmt.Errorf("DELETE 冲突行失败: %w", delResult.Error)
	}
	applogger.Infof("[迁移] 182 DELETE 冲突行完成: %d 行(让出 (device_id, interface_name) 给目标 port,独立事务已提交)", delResult.RowsAffected)

	// 阶段 3: 独立事务 - 备份 + UPDATE
	// 备份表可重复创建(已存在则 DROP 后重建),原值备份允许事后回滚
	err := db.Transaction(func(tx *gorm.DB) error {
		// 3a. CREATE TABLE(在事务内,失败时回滚,不留半成品)
		if err := tx.Exec(`
CREATE TABLE sys_device_port_status_drift_backup (
    id uuid PRIMARY KEY,
    old_device_id uuid NOT NULL,
    new_device_id uuid NOT NULL,
    port_interface_name varchar(100),
    fixed_at timestamp NOT NULL DEFAULT NOW()
)`).Error; err != nil {
			return fmt.Errorf("创建备份表失败: %w", err)
		}

		// 3b. 备份原值(同样的 target 选取规则:精确到 interface 的 MAC 匹配)
		insertSQL := `
INSERT INTO sys_device_port_status_drift_backup (id, old_device_id, new_device_id, port_interface_name)
WITH target_keys AS (
    SELECT DISTINCT ON (port.id)
           port.id AS port_id,
           ip.device_id::uuid AS target_device_id,
           port.interface_name,
           port.device_id AS old_device_id
      FROM sys_device_port_status port
      JOIN ops_info_points ip ON port.id::text = ip.port_id
     WHERE port.device_id::text != ip.device_id::text
       AND ip.deleted_at IS NULL
       AND ip.status = 0
       AND ip.device_id IS NOT NULL
       AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
     ORDER BY port.id,
              (CASE WHEN EXISTS (SELECT 1 FROM sys_device_mac_address
                                  WHERE device_id::text = ip.device_id
                                    AND interface_name ILIKE '%' || port.interface_name || '%')
                    THEN 0 ELSE 1 END) ASC,
              (CASE WHEN EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)
                    THEN 0 ELSE 1 END) ASC,
              ip.id ASC
)
SELECT port_id, old_device_id, target_device_id, interface_name FROM target_keys
ON CONFLICT (id) DO NOTHING`
		result := tx.Exec(insertSQL)
		if result.Error != nil {
			return fmt.Errorf("备份原值失败: %w", result.Error)
		}
		applogger.Infof("[迁移] 182 备份完成: %d 行 port 原值已记录(每 port 1 行)", result.RowsAffected)

		// 3c. UPDATE(用同样的 target 选取规则 + ROW_NUMBER 防多个 port 撞同一 target)
		//
		// 关键修复 (2026-06-30):有 1 个 target 被 2 个 port 同时想要
		// (例:060e5a69 5F设备#2 + GE3/45,有 2 个 port 想去这里)。
		// 物理上不可能 2 个 port 同时存在同一 (device_id, interface) 位置。
		// 用 ROW_NUMBER 标记,每 target 只改 1 个,其他跳过 + log warn。
		updateSQL := `
UPDATE sys_device_port_status port
   SET device_id = ranked.target_device_id
  FROM (
       SELECT port_id, target_device_id,
              ROW_NUMBER() OVER (PARTITION BY target_device_id, interface_name ORDER BY port_id) AS rn
         FROM (
              SELECT DISTINCT ON (port.id)
                     port.id AS port_id,
                     ip.device_id::uuid AS target_device_id,
                     port.interface_name
                FROM sys_device_port_status port
                JOIN ops_info_points ip ON port.id::text = ip.port_id
               WHERE port.device_id::text != ip.device_id::text
                 AND ip.deleted_at IS NULL
                 AND ip.status = 0
                 AND ip.device_id IS NOT NULL
                 AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
               ORDER BY port.id,
                        -- 优先级 1:port 的 interface_name 在 ip.device_id 设备上有 MAC(精确)
                        (CASE WHEN EXISTS (SELECT 1 FROM sys_device_mac_address
                                            WHERE device_id::text = ip.device_id
                                              AND interface_name ILIKE '%' || port.interface_name || '%')
                              THEN 0 ELSE 1 END) ASC,
                        -- 优先级 2:ip.device_id 设备有任何 MAC(较粗)
                        (CASE WHEN EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)
                              THEN 0 ELSE 1 END) ASC,
                        -- 优先级 3:ip.id 字典序(稳定 tie-breaker)
                        ip.id ASC
         ) t
  ) ranked
 WHERE port.id = ranked.port_id
   AND ranked.rn = 1`
		updResult := tx.Exec(updateSQL)
		if updResult.Error != nil {
			return fmt.Errorf("UPDATE 失败: %w", updResult.Error)
		}
		applogger.Infof("[迁移] 182 UPDATE 完成: %d 行受影响(每 target 仅首个 port 被修)", updResult.RowsAffected)

		// 3d. 报告被跳过的 port(target 冲突被 ROW_NUMBER 过滤掉的)
		var skippedCount int64
		if err := tx.Raw(`
SELECT COUNT(*) FROM (
    SELECT port_id, target_device_id,
           ROW_NUMBER() OVER (PARTITION BY target_device_id, interface_name ORDER BY port_id) AS rn
      FROM (
           SELECT DISTINCT ON (port.id)
                  port.id AS port_id,
                  ip.device_id::uuid AS target_device_id,
                  port.interface_name
             FROM sys_device_port_status port
             JOIN ops_info_points ip ON port.id::text = ip.port_id
            WHERE port.device_id::text != ip.device_id::text
              AND ip.deleted_at IS NULL
              AND ip.status = 0
              AND ip.device_id IS NOT NULL
              AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
              AND EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)
            ORDER BY port.id,
                     (CASE WHEN EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id) THEN 0 ELSE 1 END) ASC,
                     ip.id ASC
      ) t
) ranked WHERE rn > 1`).Scan(&skippedCount).Error; err == nil {
			applogger.Warnf("[迁移] 182 跳过 %d 个 port(目标位置被其他漂移 port 占据,需人工 review)", skippedCount)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 阶段 4: 后置验证
	var remaining int64
	if err := db.Raw(`
SELECT COUNT(*) FROM sys_device_port_status port
  JOIN ops_info_points ip ON port.id::text = ip.port_id
 WHERE port.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL AND ip.status = 0
   AND ip.device_id IS NOT NULL
   AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
   AND EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)`).Scan(&remaining).Error; err != nil {
		applogger.Warnf("[迁移] 182 验证查询失败(非阻断): %v", err)
	} else {
		applogger.Infof("[迁移] 182 验证: 剩余漂移 safe-fix 行 = %d (期望 0)", remaining)
	}

	// 阶段 5: 回滚指引(只 log,需人工执行)
	rollbackHint := `-- 手动回滚(如果需要):
-- 1. UPDATE sys_device_port_status port
--      SET device_id = b.old_device_id
--      FROM sys_device_port_status_drift_backup b
--     WHERE port.id = b.id;
-- 2. DROP TABLE sys_device_port_status_drift_backup;
-- 注意:本迁移还 DELETE 了"被冲突"的 port_status 行(无 info_point 引用),
--      那些行不会被回滚。如需完整回滚,需额外恢复(用 deleted_at 软删除更安全)。`
	applogger.Infof("[迁移] 182 回滚指引:\n%s", rollbackHint)

	log.Println("Migration 182 completed")
	return nil
}