-- Migration 127: 为用户表添加AD相关字段
-- Description: 为Phase 20 AD域控OU与部门映射功能扩展用户表
-- Created: 2026-05-22

-- 为用户表添加AD相关字段（如果不存在）
DO $$
BEGIN
    -- 添加 ad_user_dn 字段（AD用户完整DN）
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'sys_user'
        AND column_name = 'ad_user_dn'
    ) THEN
        ALTER TABLE sys_user
        ADD COLUMN ad_user_dn VARCHAR(500);
    END IF;

    -- 添加 ad_ou_dn 字段（用户所在OU的DN）
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'sys_user'
        AND column_name = 'ad_ou_dn'
    ) THEN
        ALTER TABLE sys_user
        ADD COLUMN ad_ou_dn VARCHAR(500);
    END IF;

    -- 添加 ad_synced_at 字段（最后AD同步时间）
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'sys_user'
        AND column_name = 'ad_synced_at'
    ) THEN
        ALTER TABLE sys_user
        ADD COLUMN ad_synced_at TIMESTAMP;
    END IF;
END $$;

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_sys_user_ad_user_dn
    ON sys_user(ad_user_dn)
    WHERE ad_user_dn IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sys_user_ad_ou_dn
    ON sys_user(ad_ou_dn)
    WHERE ad_ou_dn IS NOT NULL;

-- 添加列注释
COMMENT ON COLUMN sys_user.ad_user_dn IS 'AD域控用户完整DN（例如：CN=张三,OU=部门,DC=company,DC=com）';
COMMENT ON COLUMN sys_user.ad_ou_dn IS '用户所在OU的DN（用于反向查找部门）';
COMMENT ON COLUMN sys_user.ad_synced_at IS '最后同步到AD的时间';
