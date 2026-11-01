package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate231PortWriteAuditOperatorIndex —— v1.51 DEBT-02：sys_port_write_audit
// operator 查询索引。
//
// 背景：审计页（Phase 127 PWX-04）五维过滤之一是 operator；表千万行级，
// 现有索引仅 (device_id, port_id, created_at) 复合与 created_at 单列，
// operator 等值过滤必然全表扫描。
//
// 索引选型：复合 (operator, created_at) 而非 operator 单列 ——
//   - 审计页 operator 过滤恒与 created_at 倒序分页组合（list 默认排序），
//     复合索引可同时满足等值过滤与排序消除（避免 filesort）；
//   - 单列 operator 是复合索引的最左前缀，单独建只增写放大无查询收益。
//
// 千万行级实机构建评估（真实执行留 DBA 窗口，代码先行）：
//   - PG：CREATE INDEX IF NOT EXISTS 取 ShareLock，阻塞该表写入（不阻塞读）。
//     千万行评估：常规硬件 3-8 分钟构建。生产建议改用
//     CREATE INDEX CONCURRENTLY（不阻塞写，耗时约 1.5-2 倍，需独立事务执行、
//     失败遗留 INVALID 索引需手工 DROP 后重试）——由 DBA 在维护窗口按
//     scripts/dba/231_operator_index_concurrently.sql（本 migration 同款语句的
//     CONCURRENTLY 变体）手工执行；应用侧 migration 以普通方式幂等兜底
//     （索引已存在则 HasIndex 跳过）。
//   - sqlite：全表重建锁库，dev/小规模库秒级，不适用千万级（sqlite 部署
//     非目标容量场景）。
//
// 幂等：HasIndex 探测，已存在跳过；失败非阻断（调用方 WARN，下次启动重试）。
func Migrate231PortWriteAuditOperatorIndex(db *gorm.DB) error {
	log.Println("Running migration 231: DEBT-02 sys_port_write_audit operator 复合索引")

	const indexName = "idx_port_write_audit_operator_created"
	if db.Migrator().HasIndex("sys_port_write_audit", indexName) {
		log.Println("Migration 231: operator 复合索引已存在，跳过")
		return nil
	}
	// 双方言：CREATE INDEX IF NOT EXISTS 语法 PG 与 sqlite 均支持
	if err := db.Exec("CREATE INDEX IF NOT EXISTS " + indexName +
		" ON sys_port_write_audit (operator, created_at)").Error; err != nil {
		return err
	}
	log.Println("Migration 231: operator 复合索引就绪")
	return nil
}
