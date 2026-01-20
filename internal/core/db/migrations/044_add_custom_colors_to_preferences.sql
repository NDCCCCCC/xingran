-- 添加自定义颜色字段到用户偏好设置表
-- Add custom color fields to user preferences table

-- 添加自定义主题色字段（HEX 格式，如 #4F46E5）
ALTER TABLE "sys_user_preference"
ADD COLUMN "custom_primary_color" VARCHAR(20);

-- 添加自定义侧边栏颜色字段（HEX 格式，如 #1E293B）
ALTER TABLE "sys_user_preference"
ADD COLUMN "custom_sidebar_color" VARCHAR(20);

-- 添加注释
COMMENT ON COLUMN "sys_user_preference"."custom_primary_color" IS '自定义主题色（HEX格式）';
COMMENT ON COLUMN "sys_user_preference"."custom_sidebar_color" IS '自定义侧边栏颜色（HEX格式）';
