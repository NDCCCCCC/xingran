-- =============================================
-- 验证码背景图管理表
-- =============================================

-- 验证码背景图表
CREATE TABLE IF NOT EXISTS sys_captcha_background (
    id VARCHAR(36) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INT NOT NULL DEFAULT 0,

    -- 基本信息
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    file_width INT NOT NULL,
    file_height INT NOT NULL,
    file_md5 VARCHAR(32),

    -- 验证码配置
    piece_shape VARCHAR(20) NOT NULL DEFAULT 'circle',
    difficulty_level INT NOT NULL DEFAULT 1,
    allowed_shapes JSONB,

    -- 使用统计
    use_count BIGINT DEFAULT 0,
    last_used_at TIMESTAMP,
    sort_order INT DEFAULT 0,

    -- 状态管理
    status INT NOT NULL DEFAULT 1,
    remark VARCHAR(500)
);

-- 索引
CREATE INDEX idx_captcha_bg_status ON sys_captcha_background(status, deleted_at);
CREATE INDEX idx_captcha_bg_shape ON sys_captcha_background(piece_shape);
CREATE INDEX idx_captcha_bg_difficulty ON sys_captcha_background(difficulty_level);
CREATE INDEX idx_captcha_bg_sort ON sys_captcha_background(sort_order);
CREATE INDEX idx_captcha_bg_used ON sys_captcha_background(last_used_at DESC);
CREATE INDEX idx_captcha_bg_md5 ON sys_captcha_background(file_md5);

-- 注释
COMMENT ON TABLE sys_captcha_background IS '验证码背景图管理表';
COMMENT ON COLUMN sys_captcha_background.id IS '主键ID';
COMMENT ON COLUMN sys_captcha_background.file_name IS '文件名';
COMMENT ON COLUMN sys_captcha_background.file_path IS '存储路径';
COMMENT ON COLUMN sys_captcha_background.file_size IS '文件大小(字节)';
COMMENT ON COLUMN sys_captcha_background.file_width IS '图片宽度';
COMMENT ON COLUMN sys_captcha_background.file_height IS '图片高度';
COMMENT ON COLUMN sys_captcha_background.file_md5 IS '文件MD5值';
COMMENT ON COLUMN sys_captcha_background.piece_shape IS '拼图形状: circle/square/star/heart';
COMMENT ON COLUMN sys_captcha_background.difficulty_level IS '难度级别: 1-简单 2-中等 3-困难';
COMMENT ON COLUMN sys_captcha_background.allowed_shapes IS '允许的拼图形状(JSON数组)';
COMMENT ON COLUMN sys_captcha_background.use_count IS '使用次数';
COMMENT ON COLUMN sys_captcha_background.last_used_at IS '最后使用时间';
COMMENT ON COLUMN sys_captcha_background.sort_order IS '排序序号';
COMMENT ON COLUMN sys_captcha_background.status IS '状态: 0-禁用 1-启用';
COMMENT ON COLUMN sys_captcha_background.remark IS '备注';

-- =============================================
-- 验证码配置项扩展
-- =============================================

-- 背景图模式: auto=自动生成 custom=仅自定义 mixed=混合模式
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark)
VALUES (
    '验证码背景图模式',
    'sys.account.captchaBackgroundMode',
    'mixed',
    'Y',
    1,
    '背景图模式: auto=自动生成 custom=仅自定义图片 mixed=混合模式'
) ON CONFLICT (config_key) DO NOTHING;

-- 默认拼图形状
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark)
VALUES (
    '验证码默认拼图形状',
    'sys.account.captchaPieceShape',
    'circle',
    'Y',
    1,
    '默认拼图形状: circle=圆形 square=方形 star=星形 heart=心形'
) ON CONFLICT (config_key) DO NOTHING;

-- 默认难度级别
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark)
VALUES (
    '验证码默认难度',
    'sys.account.captchaDifficulty',
    '1',
    'Y',
    1,
    '难度级别: 1=简单 2=中等 3=困难'
) ON CONFLICT (config_key) DO NOTHING;

-- 缓存池大小
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark)
VALUES (
    '验证码缓存池大小',
    'sys.account.captchaCachePoolSize',
    '50',
    'Y',
    1,
    '每种形状和难度预生成的验证码数量'
) ON CONFLICT (config_key) DO NOTHING;

-- 存储路径
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark)
VALUES (
    '验证码图片存储路径',
    'sys.account.captchaStoragePath',
    './uploads/captcha/backgrounds',
    'Y',
    1,
    '背景图存储路径'
) ON CONFLICT (config_key) DO NOTHING;

-- 最大文件大小
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark)
VALUES (
    '验证码图片最大大小',
    'sys.account.captchaMaxFileSize',
    '2097152',
    'Y',
    1,
    '单张图片最大大小(字节)，默认2MB'
) ON CONFLICT (config_key) DO NOTHING;

-- 允许的图片格式
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark)
VALUES (
    '验证码允许的图片格式',
    'sys.account.captchaAllowedFormats',
    'jpg,jpeg,png',
    'Y',
    1,
    '允许的图片格式，逗号分隔'
) ON CONFLICT (config_key) DO NOTHING;

-- =============================================
-- 存储目录创建
-- =============================================

-- 创建存储目录（需要在服务器上执行）
-- mkdir -p ./uploads/captcha/backgrounds
-- chmod 755 ./uploads/captcha/backgrounds
