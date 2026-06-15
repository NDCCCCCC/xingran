-- 添加部门外部机构标识字段
-- 用于区分部门是否是拥有职场（楼宇、机房等）的外部机构

-- 添加 is_external_org 字段到 sys_dept 表
ALTER TABLE sys_dept
ADD COLUMN IF NOT EXISTS is_external_org SMALLINT DEFAULT 0;

-- 添加注释
COMMENT ON COLUMN sys_dept.is_external_org IS '是否为外部机构：0=否（内部部门），1=是（拥有职场的外部机构）';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_sys_dept_external_org ON sys_dept(is_external_org) WHERE is_external_org = 1;
