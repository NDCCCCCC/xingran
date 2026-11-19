package migrations

import (
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Migrate254PortWriteAuditNameSnapshot —— 端口写审计行增加设备/端口名称快照列
// （v1.57，审计页 UUID → 名称可读性优化的二期）。
//
// 背景：审计页此前直接展示 device_id / port_id UUID，审计员无法直接阅读。
// 一期在读时用 sys_network_device / sys_device_port_status 补名，但设备改名、
// 端口改描述或采集行被 CleanOldRecords 清理后，历史审计的"当时名称"就丢了。
// 审计是 append-only 证据链，名称必须固化在写入时刻，故新增四列快照：
//
//	device_name_snapshot      设备名称
//	device_ip_snapshot        设备管理 IP（区分重名设备）
//	interface_name_snapshot   接口名（GE0/0/1）
//	port_description_snapshot 端口描述
//
// 历史回填：本迁移按当前表内容一次性回填存量行（best effort，只填空值）。
// 回填值是"迁移时刻"的名称，早于该时刻的真实名称无法重建 —— 这是可接受
// 的降级：回填后仍比裸 UUID 可读，且查询层在快照为空时继续实时兜底。
//
// 幂等：HasColumn 守卫（列）+ UPDATE ... WHERE snapshot = ” 守卫（回填），
// 重复执行无副作用。ADD COLUMN 均为可空 TEXT，PG/SQLite 双方言安全。
//
// 失败语义：非阻断（调用方 WARN，下次启动重试）；列缺失时查询层回退实时补名，
// 功能可用但失去历史名称固化能力。
func Migrate254PortWriteAuditNameSnapshot(db *gorm.DB) error {
	log.Println("Running migration 254: sys_port_write_audit 名称快照列")

	if !db.Migrator().HasTable(&models.PortWriteAudit{}) {
		log.Println("Migration 254: sys_port_write_audit 表不存在，跳过")
		return nil
	}

	columns := []struct {
		field string
		name  string
	}{
		{"DeviceNameSnapshot", "device_name_snapshot"},
		{"DeviceIPSnapshot", "device_ip_snapshot"},
		{"InterfaceNameSnapshot", "interface_name_snapshot"},
		{"PortDescriptionSnapshot", "port_description_snapshot"},
	}
	for _, c := range columns {
		if db.Migrator().HasColumn(&models.PortWriteAudit{}, c.name) {
			continue
		}
		if err := db.Migrator().AddColumn(&models.PortWriteAudit{}, c.field); err != nil {
			return err
		}
		log.Printf("Migration 254: 列 %s 添加完成", c.name)
	}

	// 历史回填（best effort）：分批 UPDATE，每批 5000 行，最多 200 批（100 万行），
	// 避免千万行表在启动期一次性长事务锁表。剩余行由查询层实时补名兜底，
	// 也可由 DBA 手工重跑本迁移继续推进。
	//
	// 方言：UPDATE ... WHERE id IN (SELECT ... LIMIT n) 在 PG 与 SQLite 均合法。
	// 子查询显式 JOIN 两表 —— 设备侧**不过滤 deleted_at**：已软删设备的名称为
	// 历史事实，必须回填；端口采集行缺失的审计行保持空串。
	const batchSize = 5000
	const maxBatches = 200
	backfilled := int64(0)
	for i := 0; i < maxBatches; i++ {
		res := db.Exec(`
UPDATE sys_port_write_audit
SET device_name_snapshot = COALESCE((SELECT d.device_name FROM sys_network_device d WHERE d.id = sys_port_write_audit.device_id), ''),
    device_ip_snapshot = COALESCE((SELECT d.ip_address FROM sys_network_device d WHERE d.id = sys_port_write_audit.device_id), ''),
    interface_name_snapshot = COALESCE((SELECT p.interface_name FROM sys_device_port_status p WHERE p.id = sys_port_write_audit.port_id), ''),
    port_description_snapshot = COALESCE((SELECT p.description FROM sys_device_port_status p WHERE p.id = sys_port_write_audit.port_id), '')
WHERE id IN (
    SELECT id FROM sys_port_write_audit
    WHERE (device_name_snapshot = '' OR device_name_snapshot IS NULL)
      AND EXISTS (SELECT 1 FROM sys_network_device d2 WHERE d2.id = sys_port_write_audit.device_id)
    LIMIT ?
)`, batchSize)
		if res.Error != nil {
			log.Printf("Migration 254: 历史回填第 %d 批失败（不阻断启动，查询层实时补名兜底）: %v", i+1, res.Error)
			break
		}
		backfilled += res.RowsAffected
		if res.RowsAffected < batchSize {
			break
		}
	}
	if backfilled > 0 {
		log.Printf("Migration 254: 历史回填 %d 行（剩余空值行由查询层实时补名）", backfilled)
	}
	log.Println("Migration 254: 名称快照列与历史回填就绪")
	return nil
}
