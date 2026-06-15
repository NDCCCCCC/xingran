-- Migration: RPA 选择器学习表
-- Version: 103
-- Description: 添加 RPA 选择器学习相关的数据表

-- 选择器成功记录表
CREATE TABLE IF NOT EXISTS sys_rpa_selector_success (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_url VARCHAR(500) NOT NULL,
    element_id VARCHAR(200) NOT NULL,
    selector TEXT NOT NULL,
    selector_type VARCHAR(50),
    success_count INTEGER DEFAULT 1,
    avg_duration BIGINT DEFAULT 0,
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_selector_success_page_url ON sys_rpa_selector_success(page_url);
CREATE INDEX IF NOT EXISTS idx_selector_success_element_id ON sys_rpa_selector_success(element_id);
CREATE INDEX IF NOT EXISTS idx_selector_success_page_element ON sys_rpa_selector_success(page_url, element_id);
CREATE INDEX IF NOT EXISTS idx_selector_success_selector ON sys_rpa_selector_success(selector);
CREATE INDEX IF NOT EXISTS idx_selector_success_last_used ON sys_rpa_selector_success(last_used_at);

-- 选择器失败记录表
CREATE TABLE IF NOT EXISTS sys_rpa_selector_failure (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_url VARCHAR(500) NOT NULL,
    element_id VARCHAR(200) NOT NULL,
    selector TEXT NOT NULL,
    error_type VARCHAR(50),
    error_message TEXT,
    failure_count INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolved_with TEXT
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_selector_failure_page_url ON sys_rpa_selector_failure(page_url);
CREATE INDEX IF NOT EXISTS idx_selector_failure_element_id ON sys_rpa_selector_failure(element_id);
CREATE INDEX IF NOT EXISTS idx_selector_failure_page_element ON sys_rpa_selector_failure(page_url, element_id);
CREATE INDEX IF NOT EXISTS idx_selector_failure_error_type ON sys_rpa_selector_failure(error_type);
CREATE INDEX IF NOT EXISTS idx_selector_failure_created_at ON sys_rpa_selector_failure(created_at);

-- 添加表注释
COMMENT ON TABLE sys_rpa_selector_success IS 'RPA 选择器成功记录表';
COMMENT ON TABLE sys_rpa_selector_failure IS 'RPA 选择器失败记录表';
