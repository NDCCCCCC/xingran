-- 删除错误的唯一约束 uni_dept_ou_mapping_ou
-- 问题：该约束阻止不同部门映射到同名的OU（如"业务科"出现在多个分支机构）
-- 解决：删除该约束，只需要 (dept_id, ad_config_id) 唯一即可

-- 删除错误的唯一约束
DO $$ BEGIN
    ALTER TABLE sys_dept_ou_mapping
    DROP CONSTRAINT IF EXISTS uni_dept_ou_mapping_ou;

    RAISE NOTICE 'Constraint uni_dept_ou_mapping_ou dropped successfully';
EXCEPTION
    WHEN undefined_object THEN
        RAISE NOTICE 'Constraint uni_dept_ou_mapping_ou does not exist, skipping';
END $$;

-- 验证当前约束
SELECT
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'sys_dept_ou_mapping'::regclass
  AND contype = 'u';
