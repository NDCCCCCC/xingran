/**
 * NetworkDevice 工具函数
 */

import type { SelectOption } from "./constants";

// 统一时间格式化函数从 @/utils/datetime 导入
export { formatDateTime } from "@/utils/datetime";

/**
 * 根据值获取选项标签
 */
export function getOptionLabel(
  options: SelectOption[],
  value: string | number
): string | undefined {
  return options.find((o) => o.value === value)?.label;
}

/**
 * 获取状态颜色
 */
export function getStatusColor(status: number): string {
  const colorMap: Record<number, string> = {
    0: "success",
    1: "error",
    2: "default",
  };
  return colorMap[status] || "default";
}
