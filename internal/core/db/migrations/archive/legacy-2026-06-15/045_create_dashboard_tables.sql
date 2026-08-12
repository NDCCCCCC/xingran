-- ============================================
-- 可自定义仪表盘模块数据库迁移
-- 文件: 045_create_dashboard_tables.sql
-- 说明: 创建可自定义仪表盘管理相关表
-- ============================================

-- 清理可能存在的旧表
DROP TABLE IF EXISTS sys_dashboard_versions CASCADE;
DROP TABLE IF EXISTS sys_dashboards CASCADE;

-- ============================================
-- 1. 仪表盘表 sys_dashboards
-- ============================================

CREATE TABLE sys_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    owner_id VARCHAR(64),
    is_default BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    template_scope VARCHAR(20),
    layout JSONB NOT NULL,
    refresh_interval INT DEFAULT 60,
    status INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INT DEFAULT 0
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_dashboards_owner ON sys_dashboards(owner_id);
CREATE INDEX IF NOT EXISTS idx_dashboards_status ON sys_dashboards(status);
CREATE INDEX IF NOT EXISTS idx_dashboards_is_default ON sys_dashboards(is_default);
CREATE INDEX IF NOT EXISTS idx_dashboards_is_template ON sys_dashboards(is_template);
CREATE INDEX IF NOT EXISTS idx_dashboards_deleted_at ON sys_dashboards(deleted_at);

-- 添加表和字段注释
COMMENT ON TABLE sys_dashboards IS '仪表盘表';
COMMENT ON COLUMN sys_dashboards.id IS '主键ID';
COMMENT ON COLUMN sys_dashboards.name IS '仪表盘名称';
COMMENT ON COLUMN sys_dashboards.description IS '描述';
COMMENT ON COLUMN sys_dashboards.owner_id IS '所有者ID';
COMMENT ON COLUMN sys_dashboards.is_default IS '是否为默认仪表盘';
COMMENT ON COLUMN sys_dashboards.is_template IS '是否为模板';
COMMENT ON COLUMN sys_dashboards.template_scope IS '模板作用域: global=全局, dept=部门, personal=个人';
COMMENT ON COLUMN sys_dashboards.layout IS '布局配置(JSON格式)';
COMMENT ON COLUMN sys_dashboards.refresh_interval IS '刷新间隔(秒)';
COMMENT ON COLUMN sys_dashboards.status IS '状态: 0=正常 1=停用';

-- ============================================
-- 2. 仪表盘版本表 sys_dashboard_versions
-- ============================================

CREATE TABLE sys_dashboard_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL,
    layout JSONB NOT NULL,
    comment VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64)
);

-- 添加外键约束
ALTER TABLE sys_dashboard_versions ADD CONSTRAINT fk_dashboard_version_dashboard
    FOREIGN KEY (dashboard_id) REFERENCES sys_dashboards(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_dashboard_versions_dashboard ON sys_dashboard_versions(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_versions_created_at ON sys_dashboard_versions(created_at);

-- 添加表和字段注释
COMMENT ON TABLE sys_dashboard_versions IS '仪表盘版本记录表';
COMMENT ON COLUMN sys_dashboard_versions.id IS '主键ID';
COMMENT ON COLUMN sys_dashboard_versions.dashboard_id IS '仪表盘ID';
COMMENT ON COLUMN sys_dashboard_versions.layout IS '布局配置(JSON格式)';
COMMENT ON COLUMN sys_dashboard_versions.comment IS '版本备注';
COMMENT ON COLUMN sys_dashboard_versions.created_at IS '创建时间';
COMMENT ON COLUMN sys_dashboard_versions.created_by IS '创建人';

-- ============================================
-- 3. 添加仪表盘菜单
-- ============================================

DO $$
DECLARE
    v_dashboard_menu_id UUID;
    v_count INT;
BEGIN
    -- 检查仪表盘管理菜单是否已存在
    SELECT id INTO v_dashboard_menu_id FROM sys_menu WHERE menu_name = '仪表盘管理' LIMIT 1;

    -- 如果不存在，创建仪表盘管理菜单（作为顶级菜单）
    IF v_dashboard_menu_id IS NULL THEN
        INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, created_by, updated_by)
        VALUES (gen_random_uuid(), '仪表盘管理', NULL, 5, 'dashboard', NULL, 'M', 1, 0, NULL, 'DashboardOutlined', 'system', 'system')
        RETURNING id INTO v_dashboard_menu_id;
    END IF;

    -- 检查子菜单是否已存在，避免重复
    SELECT COUNT(*) INTO v_count FROM sys_menu WHERE menu_name = '仪表盘列表';
    IF v_count = 0 THEN
        INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, created_by, updated_by)
        VALUES (
            gen_random_uuid(),
            '仪表盘列表',
            v_dashboard_menu_id,
            1,
            'list',
            'dashboard-system/index',
            'C',
            1,
            0,
            'system:dashboards:list',
            'UnorderedListOutlined',
            'system',
            'system'
        );
    END IF;

    SELECT COUNT(*) INTO v_count FROM sys_menu WHERE menu_name = '仪表盘查看';
    IF v_count = 0 THEN
        INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, created_by, updated_by)
        VALUES (
            gen_random_uuid(),
            '仪表盘查看',
            v_dashboard_menu_id,
            2,
            'view',
            'dashboard-system/view',
            'C',
            0,
            0,
            'system:dashboards:query',
            'system',
            'system'
        );
    END IF;

    SELECT COUNT(*) INTO v_count FROM sys_menu WHERE menu_name = '仪表盘编辑';
    IF v_count = 0 THEN
        INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, created_by, updated_by)
        VALUES (
            gen_random_uuid(),
            '仪表盘编辑',
            v_dashboard_menu_id,
            3,
            'edit',
            'dashboard-system/edit',
            'C',
            0,
            0,
            'system:dashboards:edit',
            'system',
            'system'
        );
    END IF;
END $$;

-- ============================================
-- 4. 添加字典数据 - Widget类型
-- ============================================

DO $$
DECLARE
    v_dict_count INT;
BEGIN
    -- 检查字典类型是否已存在
    SELECT COUNT(*) INTO v_dict_count FROM sys_dict_type WHERE dict_type = 'dashboard_widget_type';

    IF v_dict_count = 0 THEN
        INSERT INTO sys_dict_type (id, dict_name, dict_type, status, created_by, updated_by)
        VALUES (gen_random_uuid(), 'Widget类型', 'dashboard_widget_type', 0, 'system', 'system');

        -- 插入字典数据
        INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, created_by, updated_by)
        VALUES
            (gen_random_uuid(), 1, '统计卡片', 'stat-card', 'dashboard_widget_type', NULL, 'default', 'N', 0, 'system', 'system'),
            (gen_random_uuid(), 2, '图表', 'chart', 'dashboard_widget_type', NULL, 'default', 'N', 0, 'system', 'system'),
            (gen_random_uuid(), 3, '表格', 'table', 'dashboard_widget_type', NULL, 'default', 'N', 0, 'system', 'system'),
            (gen_random_uuid(), 4, '列表', 'list', 'dashboard_widget_type', NULL, 'default', 'N', 0, 'system', 'system'),
            (gen_random_uuid(), 5, '进度条', 'progress', 'dashboard_widget_type', NULL, 'default', 'N', 0, 'system', 'system'),
            (gen_random_uuid(), 6, '指标卡片', 'metric', 'dashboard_widget_type', NULL, 'default', 'N', 0, 'system', 'system');
    END IF;
END $$;

-- ============================================
-- 5. 添加字典数据 - 模板作用域
-- ============================================

DO $$
DECLARE
    v_dict_count INT;
BEGIN
    -- 检查字典类型是否已存在
    SELECT COUNT(*) INTO v_dict_count FROM sys_dict_type WHERE dict_type = 'dashboard_template_scope';

    IF v_dict_count = 0 THEN
        INSERT INTO sys_dict_type (id, dict_name, dict_type, status, created_by, updated_by)
        VALUES (gen_random_uuid(), '仪表盘模板作用域', 'dashboard_template_scope', 0, 'system', 'system');

        -- 插入字典数据
        INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, created_by, updated_by)
        VALUES
            (gen_random_uuid(), 1, '全局', 'global', 'dashboard_template_scope', NULL, 'default', 'N', 0, 'system', 'system'),
            (gen_random_uuid(), 2, '部门', 'dept', 'dashboard_template_scope', NULL, 'default', 'N', 0, 'system', 'system'),
            (gen_random_uuid(), 3, '个人', 'personal', 'dashboard_template_scope', NULL, 'default', 'N', 0, 'system', 'system');
    END IF;
END $$;

-- ============================================
-- 迁移完成
-- ============================================
