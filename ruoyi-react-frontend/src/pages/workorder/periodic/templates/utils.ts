/**
 * Periodic Template Utilities
 * 周期性工单模板工具函数
 */

export const VARIABLE_HELP_CONTENT = {
  title: "工单标题支持以下变量：",
  variables: [
    { code: "{date}", description: "当前日期 (2025-01-04)" },
    { code: "{datetime}", description: "日期时间 (2025-01-04 15:30:45)" },
    { code: "{year}", description: "年份 (2025)" },
    { code: "{month}", description: "月份 (01)" },
    { code: "{day}", description: "日 (04)" },
    { code: "{weekday}", description: "星期 (Monday)" },
    { code: "{hour}", description: "小时 (15)" },
    { code: "{minute}", description: "分钟 (30)" },
  ] as const,
};
