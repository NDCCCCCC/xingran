-- 缓存时间配置数据
-- 用于配置不同缓存项的过期时间（单位：分钟）
-- 可以通过参数管理页面动态修改

-- 清除旧的缓存配置（如果存在）
DELETE FROM sys_config WHERE config_key LIKE 'cache.%';

-- 部门管理缓存配置
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES
    (gen_random_uuid(), '部门树缓存时间', 'cache.dept.tree', '30', 'Y', 1, '部门树结构数据的缓存时间（分钟），默认30分钟', NOW(), NOW()),
    (gen_random_uuid(), '部门列表缓存时间', 'cache.dept.list', '30', 'Y', 1, '部门列表数据的缓存时间（分钟），默认30分钟', NOW(), NOW()),
    (gen_random_uuid(), '部门选择器缓存时间', 'cache.dept.select', '30', 'Y', 1, '部门选择器数据的缓存时间（分钟），默认30分钟', NOW(), NOW());

-- 角色管理缓存配置
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES
    (gen_random_uuid(), '角色菜单缓存时间', 'cache.role.menus', '30', 'Y', 1, '角色菜单权限数据的缓存时间（分钟），默认30分钟', NOW(), NOW()),
    (gen_random_uuid(), '角色部门缓存时间', 'cache.role.depts', '30', 'Y', 1, '角色部门权限数据的缓存时间（分钟），默认30分钟', NOW(), NOW());

-- 字典管理缓存配置
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES
    (gen_random_uuid(), '字典类型缓存时间', 'cache.dict.type', '60', 'Y', 1, '字典类型数据的缓存时间（分钟），默认60分钟', NOW(), NOW()),
    (gen_random_uuid(), '字典数据缓存时间', 'cache.dict.data', '30', 'Y', 1, '字典数据内容的缓存时间（分钟），默认30分钟', NOW(), NOW());

-- 用户管理缓存配置
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES
    (gen_random_uuid(), '用户列表缓存时间', 'cache.user.list', '10', 'Y', 1, '用户列表数据的缓存时间（分钟），默认10分钟', NOW(), NOW()),
    (gen_random_uuid(), '用户详情缓存时间', 'cache.user.byid', '30', 'Y', 1, '用户详情数据的缓存时间（分钟），默认30分钟', NOW(), NOW());

-- 菜单管理缓存配置
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES
    (gen_random_uuid(), '菜单树缓存时间', 'cache.menu.tree', '30', 'Y', 1, '菜单树结构数据的缓存时间（分钟），默认30分钟', NOW(), NOW()),
    (gen_random_uuid(), '菜单路由缓存时间', 'cache.menu.router', '30', 'Y', 1, '菜单路由数据的缓存时间（分钟），默认30分钟', NOW(), NOW());

COMMENT ON COLUMN sys_config.config_key IS '配置键，缓存配置以 cache. 开头';
COMMENT ON COLUMN sys_config.config_value IS '配置值，缓存时间单位为分钟';
COMMENT ON COLUMN sys_config.is_system IS '是否系统内置：0=否，1=是';
