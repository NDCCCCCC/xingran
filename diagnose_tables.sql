-- 数据库表诊断脚本
-- 执行此脚本来确认表是否存在以及搜索路径配置

-- 1. 检查当前数据库
SELECT current_database() as current_db;

-- 2. 检查搜索路径
SHOW search_path;

-- 3. 检查表是否存在（在所有 schema 中）
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_name IN (
    'sys_config',
    'sys_user_preference',
    'sys_rpa_executions',
    'sys_rpa_workers'
)
ORDER BY table_schema, table_name;

-- 4. 检查 public schema 中的表
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
    AND table_name LIKE 'sys_%'
ORDER BY table_name
LIMIT 20;

-- 5. 如果表在非 public schema，设置搜索路径
-- SET search_path = public, your_schema;
