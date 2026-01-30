-- ========================================
-- 菜单系统重构：添加 meta 字段
-- ========================================
-- 目的：统一管理路由元数据（标题、图标、权限、缓存等）
-- 作者：Claude Code
-- 日期：2026-01-23
-- ========================================

-- ========================================
-- 第一步：添加 meta 字段
-- ========================================
ALTER TABLE sys_menu
ADD COLUMN IF NOT EXISTS meta jsonb;

-- ========================================
-- 第二步：为现有菜单生成默认 meta
-- ========================================
-- 使用 COALESCE 处理可能的 NULL 值
UPDATE sys_menu
SET meta = jsonb_build_object(
    'title', COALESCE(menu_name, ''),
    'icon', COALESCE(icon, ''),
    'hidden', CASE WHEN visible = 0 THEN true ELSE false END,
    'keepAlive', false,
    'affix', CASE WHEN path = 'dashboard' THEN true ELSE false END
)
WHERE meta IS NULL;

-- ========================================
-- 第三步：标准化路径字段
-- ========================================
-- 先更新所有 NULL 值为空字符串
UPDATE sys_menu
SET path = ''
WHERE path IS NULL;

-- 设置默认值
ALTER TABLE sys_menu
ALTER COLUMN path SET DEFAULT '';

-- 确保路径字段不为 NULL
ALTER TABLE sys_menu
ALTER COLUMN path SET NOT NULL;

-- 去掉前导斜杠（统一格式）
UPDATE sys_menu
SET path = LTRIM(path, '/')
WHERE path LIKE '/%';

-- ========================================
-- 第四步：创建 GIN 索引（用于 JSONB 查询）
-- ========================================
-- 支持高效的 meta 字段查询（如按 title、permissions 等查询）
CREATE INDEX IF NOT EXISTS idx_sys_menu_meta
ON sys_menu USING gin(meta);

-- ========================================
-- 第五步：添加注释
-- ========================================
COMMENT ON COLUMN sys_menu.meta IS '菜单元数据（JSONB）：包含标题、图标、权限、缓存等路由相关配置';

-- ========================================
-- 验证查询
-- ========================================
-- 检查 meta 字段是否创建成功
-- SELECT COUNT(*) FROM sys_menu WHERE meta IS NOT NULL;
-- SELECT id, menu_name, path, meta FROM sys_menu LIMIT 5;
