-- 增强文件表功能
-- 文件: 041_enhance_file_table.sql
-- 说明: 为sys_files表添加图片尺寸字段和扩展元数据字段，支持图片尺寸自动提取

-- ============================================
-- 1. 为sys_files表添加新字段
-- ============================================

-- 添加图片宽度字段
ALTER TABLE sys_files ADD COLUMN IF NOT EXISTS file_width INT;

-- 添加图片高度字段
ALTER TABLE sys_files ADD COLUMN IF NOT EXISTS file_height INT;

-- 添加扩展元数据字段（JSONB类型）
ALTER TABLE sys_files ADD COLUMN IF NOT EXISTS metadata JSONB;

-- ============================================
-- 2. 添加字段注释
-- ============================================

COMMENT ON COLUMN sys_files.file_width IS '图片宽度（像素），仅图片类型有值';
COMMENT ON COLUMN sys_files.file_height IS '图片高度（像素），仅图片类型有值';
COMMENT ON COLUMN sys_files.metadata IS '扩展元数据（JSON格式），可存储图片尺寸、EXIF信息等';

-- ============================================
-- 3. 添加索引优化查询
-- ============================================

-- 为宽高字段添加索引（用于按尺寸筛选）
CREATE INDEX IF NOT EXISTS idx_files_width ON sys_files(file_width) WHERE file_width IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_files_height ON sys_files(file_height) WHERE file_height IS NOT NULL;

-- 为元数据添加GIN索引（支持JSON查询）
CREATE INDEX IF NOT EXISTS idx_files_metadata ON sys_files USING GIN(metadata) WHERE metadata IS NOT NULL;

-- ============================================
-- 4. 添加业务类型分类索引
-- ============================================

-- 为业务类型添加索引，优化按分类查询
CREATE INDEX IF NOT EXISTS idx_files_business_type ON sys_files(business_type) WHERE is_deleted = false;

-- ============================================
-- 迁移完成
-- ============================================

-- 验证迁移
SELECT '041_enhance_file_table.sql migration completed' AS status;

-- 检查新增列
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'sys_files'
AND column_name IN ('file_width', 'file_height', 'metadata')
ORDER BY ordinal_position;

-- 检查索引
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'sys_files'
AND indexname LIKE 'idx_files_%';
