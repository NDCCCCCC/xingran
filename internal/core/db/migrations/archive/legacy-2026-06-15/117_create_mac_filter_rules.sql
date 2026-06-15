-- ============================================
-- MAC 地址过滤规则表
-- ============================================
-- 用于配置不同设备类型和厂商的 MAC 地址过滤阈值和 LLDP 过滤开关
-- 支持优先级解析：最具体规则优先（厂商+设备类型 > 设备类型 > 默认）

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

-- 创建索引以提升查询性能
CREATE INDEX idx_mac_filter_rules_device_type ON sys_mac_filter_rules(device_type);
CREATE INDEX idx_mac_filter_rules_vendor ON sys_mac_filter_rules(vendor);
CREATE INDEX idx_mac_filter_rules_priority ON sys_mac_filter_rules(priority DESC);
CREATE INDEX idx_mac_filter_rules_deleted_at ON sys_mac_filter_rules(deleted_at);

-- 插入默认规则
-- 默认交换机规则（任意厂商）
INSERT INTO sys_mac_filter_rules (id, rule_name, device_type, vendor, mac_threshold, enable_lldp_filter, priority, is_system, remark)
VALUES (gen_random_uuid(), '默认交换机规则', 'switch', NULL, 10, TRUE, 0, TRUE, '交换机默认MAC数阈值为10，启用LLDP过滤');

-- 默认路由器规则（任意厂商）
INSERT INTO sys_mac_filter_rules (id, rule_name, device_type, vendor, mac_threshold, enable_lldp_filter, priority, is_system, remark)
VALUES (gen_random_uuid(), '默认路由器规则', 'router', NULL, 500, TRUE, 0, TRUE, '路由器默认MAC数阈值为500，启用LLDP过滤');

-- 默认防火墙规则（任意厂商）
INSERT INTO sys_mac_filter_rules (id, rule_name, device_type, vendor, mac_threshold, enable_lldp_filter, priority, is_system, remark)
VALUES (gen_random_uuid(), '默认防火墙规则', 'firewall', NULL, 100, TRUE, 0, TRUE, '防火墙默认MAC数阈值为100，启用LLDP过滤');

-- 默认负载均衡器规则（任意厂商）
INSERT INTO sys_mac_filter_rules (id, rule_name, device_type, vendor, mac_threshold, enable_lldp_filter, priority, is_system, remark)
VALUES (gen_random_uuid(), '默认负载均衡器规则', 'loadbalancer', NULL, 50, TRUE, 0, TRUE, '负载均衡器默认MAC数阈值为50，启用LLDP过滤');

-- 默认无线接入点规则（任意厂商）
INSERT INTO sys_mac_filter_rules (id, rule_name, device_type, vendor, mac_threshold, enable_lldp_filter, priority, is_system, remark)
VALUES (gen_random_uuid(), '默认无线接入点规则', 'ap', NULL, 100, TRUE, 0, TRUE, '无线接入点默认MAC数阈值为100，启用LLDP过滤');
