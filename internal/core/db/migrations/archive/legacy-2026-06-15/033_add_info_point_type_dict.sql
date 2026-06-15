-- Migration: 033_add_info_point_type_dict.sql
-- Description: 添加信息点类型字典，替换硬编码
-- Date: 2026-01-14

-- 插入信息点类型字典
INSERT INTO sys_dict_type (id, dict_name, dict_type, status, remark, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '信息点类型',
    'ops_info_point_type',
    0,
    '信息点类型字典：网络信息点、电话信息点',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (dict_type) DO NOTHING;

-- 插入信息点类型字典数据
INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, remark, created_at, updated_at)
VALUES
    (
        gen_random_uuid(),
        1,
        '网络信息点',
        'network',
        'ops_info_point_type',
        NULL,
        'primary',
        true,
        0,
        '网络信息点',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        2,
        '电话信息点',
        'phone',
        'ops_info_point_type',
        NULL,
        'success',
        false,
        0,
        '电话信息点',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT DO NOTHING;

-- 删除旧的类型约束
ALTER TABLE ops_info_points DROP CONSTRAINT IF EXISTS chk_info_point_type;

-- 添加新的类型约束（包含网络和电话）
ALTER TABLE ops_info_points
    ADD CONSTRAINT chk_info_point_type
    CHECK (info_point_type IN ('network', 'phone'));
