//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate117CreateMacFilterRules 创建 MAC 地址过滤规则表 sys_mac_filter_rules
//
// 背景: 项目没有 SQL 文件自动加载器, 117_create_mac_filter_rules.sql 一直未被执行,
// 导致 topology.FilterRuleService 调用 GORM 时报 "relation sys_mac_filter_rules does not exist"。
// 本迁移与该 SQL 文件等价: 建表 + 索引 + 5 条系统默认规则。
//
// 幂等性: 表存在则跳过整段执行; 默认规则使用 ON CONFLICT 避免重复插入。
func Migrate117CreateMacFilterRules(db *gorm.DB) error {
	log.Println("Running migration 117: Create sys_mac_filter_rules table")

	// 检查表是否已存在(已存在则跳过, 保持幂等)
	if db.Migrator().HasTable("sys_mac_filter_rules") {
		log.Println("Table sys_mac_filter_rules already exists, skipping migration 117...")
		return nil
	}

	// 建表 + 索引(与 117_create_mac_filter_rules.sql 完全等价)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS sys_mac_filter_rules (
		id VARCHAR(36) PRIMARY KEY,
		rule_name VARCHAR(100) NOT NULL,
		device_type VARCHAR(50) NOT NULL,
		vendor VARCHAR(50),
		mac_threshold INTEGER NOT NULL DEFAULT 10,
		enable_lldp_filter BOOLEAN NOT NULL DEFAULT TRUE,
		priority INTEGER NOT NULL DEFAULT 0,
		is_system BOOLEAN NOT NULL DEFAULT FALSE,
		remark TEXT,
		created_by VARCHAR(100),
		updated_by VARCHAR(100),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		CONSTRAINT chk_mac_threshold CHECK (mac_threshold >= 0),
		CONSTRAINT chk_priority CHECK (priority >= 0),
		CONSTRAINT uq_device_vendor UNIQUE (device_type, vendor)
	);

	CREATE INDEX IF NOT EXISTS idx_mac_filter_rules_device_type ON sys_mac_filter_rules(device_type);
	CREATE INDEX IF NOT EXISTS idx_mac_filter_rules_vendor ON sys_mac_filter_rules(vendor);
	CREATE INDEX IF NOT EXISTS idx_mac_filter_rules_priority ON sys_mac_filter_rules(priority DESC);
	CREATE INDEX IF NOT EXISTS idx_mac_filter_rules_deleted_at ON sys_mac_filter_rules(deleted_at);
	`

	if err := db.Exec(createTableSQL).Error; err != nil {
		log.Printf("Failed to create sys_mac_filter_rules table: %v", err)
		return err
	}

	// 默认规则种子数据(系统级, is_system=true, 用户不可删除)
	// 使用 ON CONFLICT 保证幂等: 若 (device_type, vendor) 已存在则跳过
	seedSQL := `
	INSERT INTO sys_mac_filter_rules (id, rule_name, device_type, vendor, mac_threshold, enable_lldp_filter, priority, is_system, remark)
	VALUES
		(gen_random_uuid(), '默认交换机规则', 'switch', NULL, 10, TRUE, 0, TRUE, '交换机默认MAC数阈值为10，启用LLDP过滤'),
		(gen_random_uuid(), '默认路由器规则', 'router', NULL, 500, TRUE, 0, TRUE, '路由器默认MAC数阈值为500，启用LLDP过滤'),
		(gen_random_uuid(), '默认防火墙规则', 'firewall', NULL, 100, TRUE, 0, TRUE, '防火墙默认MAC数阈值为100，启用LLDP过滤'),
		(gen_random_uuid(), '默认负载均衡器规则', 'loadbalancer', NULL, 50, TRUE, 0, TRUE, '负载均衡器默认MAC数阈值为50，启用LLDP过滤'),
		(gen_random_uuid(), '默认无线接入点规则', 'ap', NULL, 100, TRUE, 0, TRUE, '无线接入点默认MAC数阈值为100，启用LLDP过滤')
	ON CONFLICT (device_type, vendor) DO NOTHING;
	`

	if err := db.Exec(seedSQL).Error; err != nil {
		log.Printf("Failed to seed default mac filter rules: %v", err)
		return err
	}

	log.Println("Migration 117 completed successfully")
	return nil
}
