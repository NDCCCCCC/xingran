-- Phase 32 P2-A4 source-tracking:
--   Original commit: 5467ba78
--   Created: 2026-06-09
--   Note: Conflicts with 027_cleanup_duplicate_indexes.sql — both share prefix 027. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- 用户列配置表
CREATE TABLE IF NOT EXISTS sys_user_column_config (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    page_key VARCHAR(100) NOT NULL,
    column_key VARCHAR(100) NOT NULL,
    visible BOOLEAN DEFAULT TRUE,
    display_order INT DEFAULT 0,
    width INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INT DEFAULT 0,
    CONSTRAINT uk_user_page_column UNIQUE (user_id, page_key, column_key)
);

-- 创建索引
CREATE INDEX idx_user_page ON sys_user_column_config(user_id, page_key);

-- 添加表注释
COMMENT ON TABLE sys_user_column_config IS '用户列配置表';
COMMENT ON COLUMN sys_user_column_config.user_id IS '用户ID';
COMMENT ON COLUMN sys_user_column_config.page_key IS '页面标识（如 asset.list）';
COMMENT ON COLUMN sys_user_column_config.column_key IS '列标识';
COMMENT ON COLUMN sys_user_column_config.visible IS '是否可见';
COMMENT ON COLUMN sys_user_column_config.display_order IS '显示顺序';
COMMENT ON COLUMN sys_user_column_config.width IS '列宽度（像素）';
