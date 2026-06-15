-- Migration: 115_migrate_dedicated_line_dict_values.sql
-- Description: 迁移专线表中的字典值，将中文名称转换为英文代码
-- Date: 2026-03-17
-- 说明：将 ops_dedicated_lines 表中的 line_type 和 isp 字段从中文标签转换为对应的字典值（dict_value）

-- 迁移 line_type 字段：将中文标签转换为英文代码
UPDATE ops_dedicated_lines
SET line_type = CASE
    -- 标准名称
    WHEN line_type = '互联网专线' THEN 'internet'
    WHEN line_type = '内网专线' THEN 'intranet'
    WHEN line_type = '云桌面专线' THEN 'cloud_desktop'
    WHEN line_type = 'MPLS VPN' THEN 'mpls'
    WHEN line_type = '光纤专线' THEN 'fiber'
    WHEN line_type = '租用专线' THEN 'leased_line'
    -- 可能的变体名称
    WHEN line_type = '互联网' THEN 'internet'
    WHEN line_type = '内网' THEN 'intranet'
    WHEN line_type = '云桌面' THEN 'cloud_desktop'
    WHEN line_type = 'MPLS' THEN 'mpls'
    WHEN line_type = '光纤' THEN 'fiber'
    WHEN line_type = '租用' THEN 'leased_line'
    -- 如果已经是英文代码，保持不变
    ELSE line_type
END
WHERE line_type IN (
    '互联网专线', '内网专线', '云桌面专线', 'MPLS VPN', '光纤专线', '租用专线',
    '互联网', '内网', '云桌面', 'MPLS', '光纤', '租用'
);

-- 迁移 isp 字段：将中文标签转换为英文代码
UPDATE ops_dedicated_lines
SET isp = CASE
    -- 标准名称
    WHEN isp = '电信' THEN 'telecom'
    WHEN isp = '移动' THEN 'mobile'
    WHEN isp = '联通' THEN 'unicom'
    WHEN isp = '广电' THEN 'broadcast'
    WHEN isp = '其他' THEN 'other'
    -- 可能的变体名称（带"中国"前缀）
    WHEN isp = '中国电信' THEN 'telecom'
    WHEN isp = '中国移动' THEN 'mobile'
    WHEN isp = '中国联通' THEN 'unicom'
    WHEN isp = '中国广电' THEN 'broadcast'
    -- 其他可能的变体
    WHEN isp = '电信公司' THEN 'telecom'
    WHEN isp = '移动公司' THEN 'mobile'
    WHEN isp = '联通公司' THEN 'unicom'
    WHEN isp = '广电公司' THEN 'broadcast'
    -- 如果已经是英文代码，保持不变
    ELSE isp
END
WHERE isp IN (
    '电信', '移动', '联通', '广电', '其他',
    '中国电信', '中国移动', '中国联通', '中国广电',
    '电信公司', '移动公司', '联通公司', '广电公司'
);

-- 添加注释说明字段值现在使用字典代码
COMMENT ON COLUMN ops_dedicated_lines.line_type IS '专线类型（字典代码，对应 sys_dict_data.dict_value，字典类型 ops_dedicated_line_type）';
COMMENT ON COLUMN ops_dedicated_lines.isp IS '运营商（字典代码，对应 sys_dict_data.dict_value，字典类型 ops_isp）';

-- 验证迁移结果（可选，用于检查是否还有未迁移的数据）
-- 这个查询会返回所有不在字典中的值
SELECT
    line_type,
    COUNT(*) as count
FROM ops_dedicated_lines
WHERE line_type NOT IN (
    SELECT dict_value
    FROM sys_dict_data
    WHERE dict_type = 'ops_dedicated_line_type'
    AND dict_value IS NOT NULL
    AND dict_value != ''
)
GROUP BY line_type;

SELECT
    isp,
    COUNT(*) as count
FROM ops_dedicated_lines
WHERE isp NOT IN (
    SELECT dict_value
    FROM sys_dict_data
    WHERE dict_type = 'ops_isp'
    AND dict_value IS NOT NULL
    AND dict_value != ''
)
GROUP BY isp;
