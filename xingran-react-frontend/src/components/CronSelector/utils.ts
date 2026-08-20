/**
 * Cron表达式工具函数
 * 提供表达式解析、生成、验证、预览等功能
 */

import cronValidate from "cron-validate";
import later from "@breejs/later";
import type { CronConfig, CronFieldConfig, CronFieldType } from "./constants";
import { WEEK_DAY_NAMES, MONTH_NAMES } from "./constants";

// ============ Cron配置转表达式 ============

/**
 * 将CronConfig对象转换为cron表达式字符串
 * @param config Cron配置对象
 * @returns cron表达式字符串
 */
export function cronConfigToExpression(config: CronConfig): string {
  const parts = [
    fieldConfigToCronPart(config.second, "second"),
    fieldConfigToCronPart(config.minute, "minute"),
    fieldConfigToCronPart(config.hour, "hour"),
    fieldConfigToCronPart(config.day, "day"),
    fieldConfigToCronPart(config.month, "month"),
    fieldConfigToCronPart(config.week, "week"),
  ];
  return parts.join(" ");
}

/**
 * 将字段配置转换为cron表达式片段
 * @param fieldConfig 字段配置
 * @param fieldType 字段类型
 * @returns cron表达式片段
 */
function fieldConfigToCronPart(fieldConfig: CronFieldConfig, fieldType: CronFieldType): string {
  const { periodType } = fieldConfig;

  // 每单位 - 使用 *
  if (periodType === "every") {
    return "*";
  }

  // 指定特定值 - 使用逗号分隔，如 0,5,10
  if (periodType === "specific" && fieldConfig.specific && fieldConfig.specific.length > 0) {
    const sorted = [...fieldConfig.specific].sort((a, b) => a - b);
    return sorted.join(",");
  }

  // 范围 - 使用连字符，如 5-10
  if (
    periodType === "range" &&
    fieldConfig.rangeStart !== undefined &&
    fieldConfig.rangeEnd !== undefined
  ) {
    return `${fieldConfig.rangeStart}-${fieldConfig.rangeEnd}`;
  }

  // 周期 - 使用斜杠，如 0/5
  if (periodType === "cycle") {
    const start = fieldConfig.cycleStart ?? (fieldType === "day" || fieldType === "month" ? 1 : 0);
    const interval = fieldConfig.cycleInterval ?? 1;
    return `${start}/${interval}`;
  }

  return "*";
}

// ============ 表达式转Cron配置 ============

/**
 * 将cron表达式字符串解析为CronConfig对象
 * @param expression cron表达式字符串
 * @returns Cron配置对象
 */
export function expressionToCronConfig(expression: string): CronConfig {
  const parts = expression.trim().split(/\s+/);
  // 处理可能只有5个字段的情况（省略秒）
  if (parts.length === 5) {
    parts.unshift("0"); // 添加默认秒字段
  }
  if (parts.length !== 6) {
    // 返回默认配置
    return getDefaultCronConfig();
  }

  return {
    second: parseCronPart(parts[0], "second"),
    minute: parseCronPart(parts[1], "minute"),
    hour: parseCronPart(parts[2], "hour"),
    day: parseCronPart(parts[3], "day"),
    month: parseCronPart(parts[4], "month"),
    week: parseCronPart(parts[5], "week"),
  };
}

/**
 * 解析cron表达式片段
 * @param part 表达式片段
 * @param fieldType 字段类型
 * @returns 字段配置对象
 */
function parseCronPart(part: string, fieldType: CronFieldType): CronFieldConfig {
  // 问号 - 仅用于日和周字段，表示不指定
  if (part === "?") {
    return { type: fieldType, periodType: "every" };
  }

  // 星号 - 每单位
  if (part === "*") {
    return { type: fieldType, periodType: "every" };
  }

  // 斜杠 - 周期，如 0/5 或 */5
  if (part.includes("/")) {
    const [startStr, intervalStr] = part.split("/");
    const start = startStr === "*" ? 0 : parseInt(startStr);
    const interval = parseInt(intervalStr);
    return {
      type: fieldType,
      periodType: "cycle",
      cycleStart: start,
      cycleInterval: interval,
    };
  }

  // 连字符 - 范围，如 5-10 或 MON-FRI
  if (part.includes("-")) {
    const [startStr, endStr] = part.split("-");
    const start = parseFieldValue(startStr, fieldType);
    const end = parseFieldValue(endStr, fieldType);
    return {
      type: fieldType,
      periodType: "range",
      rangeStart: start,
      rangeEnd: end,
    };
  }

  // 逗号 - 指定多个值，如 0,5,10 或 MON,WED,FRI
  if (part.includes(",")) {
    const values = part.split(",").map((v) => parseFieldValue(v, fieldType));
    return { type: fieldType, periodType: "specific", specific: values };
  }

  // 单个值
  const value = parseFieldValue(part, fieldType);
  return { type: fieldType, periodType: "specific", specific: [value] };
}

/**
 * 解析字段值（处理数字和星期/月份缩写）
 */
function parseFieldValue(value: string, fieldType: CronFieldType): number {
  const num = parseInt(value);
  if (!isNaN(num)) {
    return num;
  }

  // 处理星期缩写（SUN, MON, TUE...）
  if (fieldType === "week") {
    const dayIndex = ["SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"].indexOf(value.toUpperCase());
    if (dayIndex !== -1) {
      return dayIndex + 1; // 转换为1-7
    }
  }

  // 处理月份缩写（JAN, FEB...）
  if (fieldType === "month") {
    const monthIndex = [
      "JAN",
      "FEB",
      "MAR",
      "APR",
      "MAY",
      "JUN",
      "JUL",
      "AUG",
      "SEP",
      "OCT",
      "NOV",
      "DEC",
    ].indexOf(value.toUpperCase());
    if (monthIndex !== -1) {
      return monthIndex + 1;
    }
  }

  return 0;
}

// ============ 表达式验证 ============

/**
 * 验证cron表达式是否有效
 * @param expression cron表达式
 * @returns 是否有效
 */
export function validateCronExpression(expression: string): boolean {
  if (!expression || expression.trim() === "") {
    return false;
  }

  try {
    // cron-validate 支持 Quartz 格式，包括 '?' 符号
    const result = cronValidate(expression);

    if (!result.isValid) {
      return false;
    }

    // 检查是否是支持的格式 (5或6个字段)
    const parts = expression.trim().split(/\s+/);
    const fieldCount = parts.length;

    // 支持5字段(标准)或6字段(带秒)格式
    return fieldCount === 5 || fieldCount === 6;
  } catch {
    return false;
  }
}

// ============ 执行时间计算 ============

/**
 * 获取未来执行时间
 * @param expression cron表达式
 * @param count 获取的次数
 * @returns 执行时间数组
 */
export function getNextRunTimes(expression: string, count: number = 5): Date[] {
  try {
    // @breejs/later 需要标准 cron 格式，将 Quartz 的 '?' 替换为 '*'
    const normalizedExpression = expression.replace(/\?/g, "*");

    // 设置使用本地时间而非 UTC 时间
    later.date.localTime();

    // 判断表达式是否包含秒字段（6字段格式包含秒，5字段格式不包含秒）
    const parts = normalizedExpression.trim().split(/\s+/);
    const hasSeconds = parts.length === 6;

    // 解析 cron 表达式，必须正确传递 hasSeconds 参数
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    const schedule = later.parse.cron(normalizedExpression, hasSeconds);

    // 使用 later.schedule(schedule).next(count) 一次性获取多次执行时间
    // @breejs/later 实际运行时始终返回 Date[] (单值用 Date 但 API 签名是不定 union),
    // 这里用 Array.isArray() 守卫,运行时转换 TS 7 的类型不能跨赋值语句自动收窄。
    const rawTimes: Date | Date[] = later.schedule(schedule).next(count);
    const times: Date[] = Array.isArray(rawTimes) ? rawTimes : [];

    // 确保返回的是 Date 数组
    return Array.isArray(times) ? times : [];
  } catch (e) {
    console.error("getNextRunTimes error:", e);
    return [];
  }
}

// ============ 中文描述生成 ============

/**
 * 将cron表达式转换为中文描述
 * @param expression cron表达式
 * @returns 中文描述
 */
export function cronToChinese(expression: string): string {
  try {
    const config = expressionToCronConfig(expression);
    const parts: string[] = [];

    // 处理秒
    if (config.second.periodType === "specific" && config.second.specific?.[0] === 0) {
      // 默认每分钟第0秒，不显示
    } else if (config.second.periodType === "every") {
      parts.push("每秒");
    } else if (config.second.periodType === "cycle") {
      parts.push(`每${config.second.cycleInterval}秒`);
    }

    // 处理分
    if (config.minute.periodType === "every") {
      if (parts.length === 0) parts.push("每分钟");
    } else if (config.minute.periodType === "specific" && config.minute.specific?.length === 1) {
      parts.push(`第${config.minute.specific[0]}分`);
    } else if (config.minute.periodType === "cycle") {
      const start = config.minute.cycleStart ?? 0;
      const interval = config.minute.cycleInterval ?? 1;
      parts.push(`从${start}分开始每${interval}分钟`);
    }

    // 处理时
    if (config.hour.periodType === "every") {
      if (parts.length === 0) parts.push("每小时");
    } else if (config.hour.periodType === "specific" && config.hour.specific?.length === 1) {
      parts.push(`${config.hour.specific[0]}点`);
    } else if (config.hour.periodType === "cycle") {
      const start = config.hour.cycleStart ?? 0;
      const interval = config.hour.cycleInterval ?? 1;
      parts.push(`从${start}点开始每${interval}小时`);
    } else if (
      config.hour.periodType === "specific" &&
      config.hour.specific &&
      config.hour.specific.length > 1
    ) {
      parts.push(`${config.hour.specific.join("、")}点`);
    }

    // 处理日
    if (config.day.periodType === "every") {
      if (parts.length === 0) parts.push("每天");
    } else if (config.day.periodType === "specific" && config.day.specific?.length === 1) {
      parts.push(`每月${config.day.specific[0]}号`);
    } else if (config.day.periodType === "cycle") {
      const start = config.day.cycleStart ?? 1;
      const interval = config.day.cycleInterval ?? 1;
      parts.push(`从${start}号开始每${interval}天`);
    }

    // 处理周
    if (
      config.week.periodType === "specific" &&
      config.week.specific &&
      config.week.specific.length > 0
    ) {
      const days = config.week.specific.map((d) => WEEK_DAY_NAMES[d - 1] || d).join("、");
      parts.push(`每${days}`);
    }

    // 处理月
    if (config.month.periodType === "specific" && config.month.specific?.length === 1) {
      parts.push(`每年${MONTH_NAMES[config.month.specific[0] - 1]}`);
    } else if (config.month.periodType === "cycle") {
      const start = config.month.cycleStart ?? 1;
      const interval = config.month.cycleInterval ?? 1;
      parts.push(`从${MONTH_NAMES[start - 1]}开始每${interval}个月`);
    }

    if (parts.length === 0) {
      return "每分钟执行";
    }

    return parts.join("") + "执行";
  } catch {
    return "无法解析的表达式";
  }
}

// ============ 默认配置 ============

/**
 * 获取默认的Cron配置（每天早上9点）
 */
export function getDefaultCronConfig(): CronConfig {
  return {
    second: { type: "second", periodType: "specific", specific: [0] },
    minute: { type: "minute", periodType: "specific", specific: [0] },
    hour: { type: "hour", periodType: "specific", specific: [9] },
    day: { type: "day", periodType: "every" },
    month: { type: "month", periodType: "every" },
    week: { type: "week", periodType: "every" },
  };
}

/**
 * 获取每分钟的Cron配置
 */
export function getEveryMinuteCronConfig(): CronConfig {
  return {
    second: { type: "second", periodType: "specific", specific: [0] },
    minute: { type: "minute", periodType: "every" },
    hour: { type: "hour", periodType: "every" },
    day: { type: "day", periodType: "every" },
    month: { type: "month", periodType: "every" },
    week: { type: "week", periodType: "every" },
  };
}
