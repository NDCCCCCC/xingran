-- 检查工位表唯一索引是否存在
SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND indexname = 'sys_workstation_floor_name_idx';
