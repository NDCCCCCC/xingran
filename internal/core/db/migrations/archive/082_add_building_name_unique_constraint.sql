-- 082: 为楼宇表添加唯一约束
-- 用于支持 ON CONFLICT DO UPDATE 操作

-- 添加 name 字段的唯一约束
ALTER TABLE ops_buildings ADD CONSTRAINT uq_ops_buildings_name UNIQUE (name);

-- 验证约束创建成功
SELECT conname, contype
FROM pg_constraint
WHERE conrelid = 'ops_buildings'::regclass;
