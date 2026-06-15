-- Migration 005: Expand user preferences table
-- 扩展用户个人设置表，支持完整的主题和布局配置
-- Author: Claude Code
-- Date: 2026-01-18

-- 添加新字段到 sys_user_preference 表
ALTER TABLE sys_user_preference
ADD COLUMN IF NOT EXISTS theme_style VARCHAR(20) DEFAULT 'minimal',
ADD COLUMN IF NOT EXISTS layout_type VARCHAR(20) DEFAULT 'classic',
ADD COLUMN IF NOT EXISTS layout_density VARCHAR(20) DEFAULT 'comfortable',
ADD COLUMN IF NOT EXISTS sidebar_width INT DEFAULT 240,
ADD COLUMN IF NOT EXISTS sidebar_collapsed_width INT DEFAULT 64;

-- 添加字段注释
COMMENT ON COLUMN sys_user_preference.theme_style IS '主题风格：minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet';
COMMENT ON COLUMN sys_user_preference.layout_type IS '布局类型：classic/hybrid/innovative';
COMMENT ON COLUMN sys_user_preference.layout_density IS '布局密度：compact/comfortable/spacious';
COMMENT ON COLUMN sys_user_preference.sidebar_width IS '侧边栏展开宽度（像素）';
COMMENT ON COLUMN sys_user_preference.sidebar_collapsed_width IS '侧边栏折叠宽度（像素）';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_user_preference_theme
ON sys_user_preference(user_id, theme, theme_style);

-- 添加检查约束 - 确保数据完整性
ALTER TABLE sys_user_preference
DROP CONSTRAINT IF EXISTS chk_theme_style,
ADD CONSTRAINT chk_theme_style
CHECK (theme_style IN ('minimal', 'glassmorphism', 'neumorphism', 'flat2.0', 'luxury-quiet'));

ALTER TABLE sys_user_preference
DROP CONSTRAINT IF EXISTS chk_layout_type,
ADD CONSTRAINT chk_layout_type
CHECK (layout_type IN ('classic', 'hybrid', 'innovative'));

ALTER TABLE sys_user_preference
DROP CONSTRAINT IF EXISTS chk_layout_density,
ADD CONSTRAINT chk_layout_density
CHECK (layout_density IN ('compact', 'comfortable', 'spacious'));

ALTER TABLE sys_user_preference
DROP CONSTRAINT IF EXISTS chk_page_size,
ADD CONSTRAINT chk_page_size
CHECK (page_size IN (5, 10, 20, 50, 100));

-- 为现有用户设置默认值（对于 NULL 值）
UPDATE sys_user_preference
SET theme_style = 'minimal'
WHERE theme_style IS NULL;

UPDATE sys_user_preference
SET layout_type = 'classic'
WHERE layout_type IS NULL;

UPDATE sys_user_preference
SET layout_density = 'comfortable'
WHERE layout_density IS NULL;

UPDATE sys_user_preference
SET sidebar_width = 240
WHERE sidebar_width IS NULL;

UPDATE sys_user_preference
SET sidebar_collapsed_width = 64
WHERE sidebar_collapsed_width IS NULL;

-- 设置字段为 NOT NULL（在设置默认值后）
ALTER TABLE sys_user_preference
ALTER COLUMN theme_style SET NOT NULL,
ALTER COLUMN layout_type SET NOT NULL,
ALTER COLUMN layout_density SET NOT NULL,
ALTER COLUMN sidebar_width SET NOT NULL,
ALTER COLUMN sidebar_collapsed_width SET NOT NULL;
