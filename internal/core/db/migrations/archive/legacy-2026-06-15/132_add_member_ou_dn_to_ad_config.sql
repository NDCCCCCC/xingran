-- Migration: Add member_ou_dn column to sys_ad_config
-- Purpose: Support specifying target OU for department groups (e.g., "本部部门分组")
-- Date: 2026-05-26

-- Add member_ou_dn column to sys_ad_config table
ALTER TABLE sys_ad_config
ADD COLUMN IF NOT EXISTS member_ou_dn VARCHAR(500);

-- Add comment for documentation
COMMENT ON COLUMN sys_ad_config.member_ou_dn IS '本部部门分组OU DN，用于创建和管理部门组（如：OU=本部部门分组,DC=example,DC=com）';

-- Create index for faster queries if needed (optional, since most queries are by ID)
-- CREATE INDEX idx_ad_config_member_ou ON sys_ad_config(member_ou_dn);
