/**
 * Cache 常量定义
 */

export interface SelectOption {
  label: string;
  value: string;
}

// 缓存类型选项
export const TYPE_OPTIONS: SelectOption[] = [
  { label: "字符串", value: "string" },
  { label: "列表", value: "list" },
  { label: "哈希", value: "hash" },
  { label: "集合", value: "set" },
  { label: "有序集合", value: "zset" },
  { label: "其他", value: "other" }
];

// 操作选项
export const OPERATION_OPTIONS: SelectOption[] = [
  { label: "获取值", value: "get" },
  { label: "设置值", value: "set" },
  { label: "删除键", value: "del" },
  { label: "检查存在", value: "exists" },
  { label: "设置过期时间", value: "expire" },
  { label: "获取TTL", value: "ttl" }
];

// 缓存层级选项
export const LEVEL_OPTIONS: SelectOption[] = [
  { label: "全部", value: "all" },
  { label: "L1(内存)", value: "l1" },
  { label: "L2(Redis)", value: "l2" },
];

// 缓存层级标签配置
export const LEVEL_TAG_CONFIG: Record<string, { color: string; label: string }> = {
  l1: { color: "blue", label: "L1(内存)" },
  l2: { color: "green", label: "L2(Redis)" },
  both: { color: "purple", label: "L1+L2" },
};
