-- Migration 126: 创建部门-OU映射表
-- Description: 为Phase 20 AD域控OU与部门映射功能创建映射表
-- Created: 2026-05-22

-- 创建部门-OU映射表
CREATE TABLE IF NOT EXISTS sys_dept_ou_mapping (
    -- 主键
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 外键
    dept_id UUID NOT NULL,
    ad_config_id UUID NOT NULL,

    -- OU信息
    ou_dn VARCHAR(500) NOT NULL,
    ou_name VARCHAR(255) NOT NULL,
    parent_ou_dn VARCHAR(500),

    -- 同步状态
    sync_enabled BOOLEAN DEFAULT true,
    sync_status VARCHAR(20) DEFAULT 'pending',
    last_sync_at TIMESTAMP,

    -- 审计字段
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_dept_ou_mapping_dept FOREIGN KEY (dept_id)
        REFERENCES sys_dept(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_dept_ou_mapping_ad_config FOREIGN KEY (ad_config_id)
        REFERENCES sys_ad_config(id)
        ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_dept_ou_mapping_dn
    ON sys_dept_ou_mapping(ou_dn);

CREATE INDEX IF NOT EXISTS idx_dept_ou_mapping_dept
    ON sys_dept_ou_mapping(dept_id);

CREATE INDEX IF NOT EXISTS idx_dept_ou_mapping_config
    ON sys_dept_ou_mapping(ad_config_id);

CREATE INDEX IF NOT EXISTS idx_dept_ou_mapping_status
    ON sys_dept_ou_mapping(sync_status, last_sync_at);

-- 创建唯一约束（一个AD配置下每个部门只能有一个映射）
CREATE UNIQUE INDEX IF NOT EXISTS uni_dept_ou_mapping_dept
    ON sys_dept_ou_mapping(dept_id, ad_config_id);

-- 创建唯一约束（一个AD配置下每个OU只能映射一个部门）
CREATE UNIQUE INDEX IF NOT EXISTS uni_dept_ou_mapping_ou
    ON sys_dept_ou_mapping(ad_config_id, ou_dn);

-- 添加表注释
COMMENT ON TABLE sys_dept_ou_mapping IS '系统部门与AD域控OU的映射关系表';

-- 添加列注释
COMMENT ON COLUMN sys_dept_ou_mapping.id IS '主键ID';
COMMENT ON COLUMN sys_dept_ou_mapping.dept_id IS '系统部门ID（外键）';
COMMENT ON COLUMN sys_dept_ou_mapping.ad_config_id IS 'AD配置ID（外键）';
COMMENT ON COLUMN sys_dept_ou_mapping.ou_dn IS 'AD域控OU的完整DN';
COMMENT ON COLUMN sys_dept_ou_mapping.ou_name IS 'OU名称';
COMMENT ON COLUMN sys_dept_ou_mapping.parent_ou_dn IS '父OU的DN';
COMMENT ON COLUMN sys_dept_ou_mapping.sync_enabled IS '是否启用同步';
COMMENT ON COLUMN sys_dept_ou_mapping.sync_status IS '同步状态：pending/synced/failed';
COMMENT ON COLUMN sys_dept_ou_mapping.last_sync_at IS '最后同步时间';
