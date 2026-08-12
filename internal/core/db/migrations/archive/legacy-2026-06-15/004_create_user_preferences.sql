-- 创建用户个人设置表（PostgreSQL语法）
-- 命名规约: unique 约束显式命名 uni_<table>_<col> 与 GORM `uniqueIndex` tag 对齐
-- (避免 PG 自动命名 sys_user_preference_user_id_key 与 GORM 期望冲突)
CREATE TABLE IF NOT EXISTS sys_user_preference (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    theme VARCHAR(20) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'zh-CN',
    page_size INT DEFAULT 10,
    sidebar_collapsed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uni_sys_user_preference_user_id UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_id ON sys_user_preference(user_id);

-- 添加表和字段注释
COMMENT ON TABLE sys_user_preference IS '用户个人设置表';
COMMENT ON COLUMN sys_user_preference.id IS '主键ID';
COMMENT ON COLUMN sys_user_preference.user_id IS '用户ID';
COMMENT ON COLUMN sys_user_preference.theme IS '主题：light/dark';
COMMENT ON COLUMN sys_user_preference.language IS '语言：zh-CN/en-US';
COMMENT ON COLUMN sys_user_preference.page_size IS '默认分页大小';
COMMENT ON COLUMN sys_user_preference.sidebar_collapsed IS '侧边栏是否折叠';
COMMENT ON COLUMN sys_user_preference.created_at IS '创建时间';
COMMENT ON COLUMN sys_user_preference.updated_at IS '更新时间';
