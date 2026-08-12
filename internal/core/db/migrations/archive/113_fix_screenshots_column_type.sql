-- 修复 RPA 执行记录 screenshots 字段类型
-- 从 PostgreSQL 数组改为 JSON 文本存储，解决 GORM 扫描兼容性问题

-- 1. 先清空旧格式的截图数据（避免兼容问题）
UPDATE sys_rpa_executions SET screenshots = '[]' WHERE screenshots IS NOT NULL;

-- 2. 修改列类型
ALTER TABLE sys_rpa_executions
ALTER COLUMN screenshots TYPE TEXT USING CASE
    WHEN screenshots IS NULL THEN '[]'
    ELSE screenshots::TEXT
END;

-- 3. 设置默认值
ALTER TABLE sys_rpa_executions
ALTER COLUMN screenshots SET DEFAULT '[]';

-- 4. 添加注释
COMMENT ON COLUMN sys_rpa_executions.screenshots IS '截图列表，JSON 字符串数组格式';
