-- ============================================
-- 仪表盘权限扩展数据库迁移
-- 文件: 050_add_dashboard_permissions.sql
-- 说明: 添加仪表盘基于部门的权限控制功能
-- ============================================

-- ============================================
-- 1. 添加仪表盘权限字段
-- ============================================

ALTER TABLE sys_dashboards
ADD COLUMN IF NOT EXISTS owner_dept_id VARCHAR(64),
ADD COLUMN IF NOT EXISTS scope VARCHAR(20) DEFAULT 'private',
ADD COLUMN IF NOT EXISTS dept_id VARCHAR(64),
ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT false;

-- ============================================
-- 2. 创建索引
-- ============================================

CREATE INDEX IF NOT EXISTS idx_dashboards_scope ON sys_dashboards(scope);
CREATE INDEX IF NOT EXISTS idx_dashboards_dept_id ON sys_dashboards(dept_id);
CREATE INDEX IF NOT EXISTS idx_dashboards_is_system ON sys_dashboards(is_system);
CREATE INDEX IF NOT EXISTS idx_dashboards_owner_dept_id ON sys_dashboards(owner_dept_id);

-- ============================================
-- 3. 更新现有数据（私有化现有仪表盘）
-- ============================================

UPDATE sys_dashboards
SET scope = 'private'
WHERE scope IS NULL;

-- ============================================
-- 4. 添加表和字段注释
-- ============================================

COMMENT ON COLUMN sys_dashboards.owner_dept_id IS '创建者部门ID';
COMMENT ON COLUMN sys_dashboards.scope IS '仪表盘可见范围: private=私有(仅自己), dept=部门可见, global=全局可见';
COMMENT ON COLUMN sys_dashboards.dept_id IS '关联部门ID（scope=dept时使用）';
COMMENT ON COLUMN sys_dashboards.is_system IS '是否为系统仪表盘（管理员创建的全局仪表盘）';

-- ============================================
-- 5. 更新菜单配置 - 仪表盘菜单改为统一入口
-- ============================================
-- 注意：如果需要手动更新菜单，可以执行以下语句（可选）

-- 菜单更新需要分步执行，使用 INSERT ... ON CONFLICT 代替 DO 块

-- 步骤 5.1: 检查并更新仪表盘列表菜单名称和路径
UPDATE sys_menu
SET menu_name = '仪表盘',
    path = '',
    component = 'dashboard-system/index',
    visible = 1
WHERE menu_name = '仪表盘列表';

-- 步骤 5.2: 删除不再需要的独立路由菜单
DELETE FROM sys_menu WHERE menu_name IN ('仪表盘查看', '仪表盘编辑');

-- ============================================
-- 6. 添加字典数据 - 仪表盘可见范围
-- ============================================

-- 步骤 6.1: 插入字典类型（如果不存在）
INSERT INTO sys_dict_type (id, dict_name, dict_type, status, created_by, updated_by)
SELECT gen_random_uuid(), '仪表盘可见范围', 'dashboard_scope', 0, 'system', 'system'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_dict_type WHERE dict_type = 'dashboard_scope'
);

-- 步骤 6.2: 插入字典数据
INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, created_by, updated_by)
SELECT
    gen_random_uuid(), 1, '私有', 'private', 'dashboard_scope', NULL, 'default', 'Y', 0, 'system', 'system'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_dict_data WHERE dict_type = 'dashboard_scope' AND dict_value = 'private'
);

INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, created_by, updated_by)
SELECT
    gen_random_uuid(), 2, '部门', 'dept', 'dashboard_scope', NULL, 'default', 'N', 0, 'system', 'system'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_dict_data WHERE dict_type = 'dashboard_scope' AND dict_value = 'dept'
);

INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, created_by, updated_by)
SELECT
    gen_random_uuid(), 3, '全局', 'global', 'dashboard_scope', NULL, 'default', 'N', 0, 'system', 'system'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_dict_data WHERE dict_type = 'dashboard_scope' AND dict_value = 'global'
);

-- ============================================
-- 迁移完成
-- ============================================
