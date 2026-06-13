CREATE TABLE IF NOT EXISTS sys_mac_oui_vendor (
    oui_prefix VARCHAR(6) PRIMARY KEY,  -- AABBCC格式（大写无分隔符）
    vendor_name VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mac_oui_vendor_updated ON sys_mac_oui_vendor(updated_at);
