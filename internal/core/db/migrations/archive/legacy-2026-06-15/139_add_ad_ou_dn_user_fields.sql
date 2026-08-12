-- Migration 139: Add ad_ou_dn and ad_synced_at columns to sys_user
-- Description: AD login department sync writes to these columns
-- Note: ad_dn column already exists (stores full user DN)
-- Created: 2026-05-27

-- Add columns if they don't exist (GORM AutoMigrate will also handle this, but explicit is safer)
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_ou_dn TEXT;
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_synced_at TIMESTAMPTZ;

-- Create index for OU DN lookups (used in AD sync queries)
CREATE INDEX IF NOT EXISTS idx_sys_user_ad_ou_dn ON sys_user (ad_ou_dn) WHERE ad_ou_dn IS NOT NULL;
