/**
 * 定时调度「运行方式」换算（纯函数，零依赖、可单测）
 *
 * 设计背景（Phase 158 / REC-02）：
 *   员工不该看到「cron」这种技术字符串，但后端调度器只认 cron 表达式。
 *   本文件是两者之间**唯一**的换算口径：
 *     简单选项（每天 / 每周 / 每隔 N 分钟）  ⟷  cron 表达式
 *
 * 为什么手写而不引 cron-parser：
 *   1. 零新增 npm 依赖（硬约束）；
 *   2. 后端用 robfig/cron 口径（5 位「分 时 日 月 周」，秒可选），
 *      第三方库的口径与解析细节可能与后端不一致 —— 手写可精确对齐
 *      「每天 09:00 = 0 9 * * *」这类约定，避免前后端两套真相。
 *
 * 互逆性约定（三个简单预设）：
 *   cronToPreset(presetToCron(p, opts)).preset === p
 *   presetToCron(cronToPreset(cron).preset, cronToPreset(cron).opts) === cron
 */

/** 简单预设（员工可见的三选一） */
export type SimplePreset = "daily" | "weekly" | "interval";

/** 运行方式（简单预设 + 自定义） */
export type SchedulePreset = SimplePreset | "custom";

/** 每天：几点几分 */
export interface DailyOptions {
  hour: number;
  minute: number;
}

/** 每周：星期几（1=周一 … 7=周日）+ 几点几分 */
export interface WeeklyOptions {
  weekday: number;
  hour: number;
  minute: number;
}

/** 每隔 N 分钟 */
export interface IntervalOptions {
  minutes: number;
}

/** 三个简单预设的选项联合类型 */
export type PresetOptions = DailyOptions | WeeklyOptions | IntervalOptions;

/** cron → 展示用的简单预设描述 */
export interface SimplePresetDescription {
  preset: SimplePreset;
  /** 中文业务标签（列表页「运行方式」列直接展示） */
  label: string;
  /** 回填到表单的选项 */
  opts: PresetOptions;
}

/** cron → 展示用的自定义描述 */
export interface CustomPresetDescription {
  preset: "custom";
  label: string;
  /** 原始 cron 字符串（仅「高级设置」里可见） */
  cron: string;
}

export type CronDescription = SimplePresetDescription | CustomPresetDescription;

// ---------------------------------------------------------------------------
// 默认值与文案常量（员工可见的中文口径集中在这里）
// ---------------------------------------------------------------------------

/** 每天的默认时间：09:00 */
export const DEFAULT_HOUR = 9;
/** 每天/每周的默认分钟：00 */
export const DEFAULT_MINUTE = 0;
/** 每周的默认星期：周一 */
export const DEFAULT_WEEKDAY = 1;
/** 「每隔 N 分钟」的默认间隔：15 分钟 */
export const DEFAULT_INTERVAL_MINUTES = 15;
/** 「每隔 N 分钟」的输入步长：5 */
export const INTERVAL_STEP = 5;
/** 「每隔 N 分钟」的取值下界 */
export const INTERVAL_MIN_MINUTES = 5;
/** 「每隔 N 分钟」的取值上界 */
export const INTERVAL_MAX_MINUTES = 120;

/** 自定义规则的展示文案 */
export const CUSTOM_LABEL = "自定义";

/** 星期几中文名（索引 0 = 周一 … 6 = 周日） */
export const WEEKDAY_NAMES = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"] as const;

/** 星期几下拉选项（value 与 WeeklyOptions.weekday 同构：1=周一 … 7=周日） */
export const WEEKDAY_OPTIONS = WEEKDAY_NAMES.map((label, index) => ({
  label,
  value: index + 1,
}));

/** 星期几 → 中文名（1=周一 … 7=周日；非法值回落「周一」） */
export function weekdayLabel(weekday: number): string {
  const index = weekday - 1;
  if (!Number.isInteger(index) || index < 0 || index >= WEEKDAY_NAMES.length) {
    return WEEKDAY_NAMES[0];
  }
  return WEEKDAY_NAMES[index];
}

// ---------------------------------------------------------------------------
// 内部工具
// ---------------------------------------------------------------------------

/** 两位补零（时间展示用） */
function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

/** 是否是纯数字字段（不接受 * / , - 之类的复合写法） */
function isPlainInt(field: string): boolean {
  return /^\d{1,2}$/.test(field);
}

/** 取值域内取整，超界或非法时回落默认值 */
function clampInt(value: unknown, min: number, max: number, fallback: number): number {
  const num = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(num)) return fallback;
  const rounded = Math.trunc(num);
  if (rounded < min || rounded > max) return fallback;
  return rounded;
}

// ---------------------------------------------------------------------------
// 正向：简单选项 → cron
// ---------------------------------------------------------------------------

/**
 * presetToCron —— 简单选项换算为 cron 表达式。
 *
 * @param preset 简单预设（daily / weekly / interval）
 * @param opts   选项；缺省时用默认值（每天 09:00 / 每周一 09:00 / 每隔 15 分钟）
 * @returns 5 位 cron 表达式（分 时 日 月 周），与后端 robfig/cron 口径一致
 *
 * 示例：
 *   presetToCron("daily", { hour: 9, minute: 0 })      → "0 9 * * *"
 *   presetToCron("weekly", { weekday: 1, hour: 9, minute: 0 }) → "0 9 * * 1"
 *   presetToCron("interval", { minutes: 15 })          → 间隔 15 分钟（斜杠步长写法）
 */
export function presetToCron(preset: SimplePreset, opts: Partial<PresetOptions> = {}): string {
  switch (preset) {
    case "daily": {
      const o = opts as Partial<DailyOptions>;
      const hour = clampInt(o.hour, 0, 23, DEFAULT_HOUR);
      const minute = clampInt(o.minute, 0, 59, DEFAULT_MINUTE);
      return `${minute} ${hour} * * *`;
    }
    case "weekly": {
      const o = opts as Partial<WeeklyOptions>;
      const weekday = clampInt(o.weekday, 1, 7, DEFAULT_WEEKDAY);
      const hour = clampInt(o.hour, 0, 23, DEFAULT_HOUR);
      const minute = clampInt(o.minute, 0, 59, DEFAULT_MINUTE);
      // cron 的星期位：0=周日 … 6=周六；业务口径 1=周一 … 7=周日 → 取模 7
      return `${minute} ${hour} * * ${weekday % 7}`;
    }
    case "interval": {
      const o = opts as Partial<IntervalOptions>;
      const minutes = clampInt(o.minutes, 1, 1440, DEFAULT_INTERVAL_MINUTES);
      return `*/${minutes} * * * *`;
    }
    default:
      // 理论不可达（类型已约束）；保底回落每天 09:00，避免产出非法表达式
      return presetToCron("daily");
  }
}

// ---------------------------------------------------------------------------
// 反向：cron → 展示用
// ---------------------------------------------------------------------------

/**
 * cronToPreset —— cron 表达式换算为展示用描述。
 *
 * 命中三个简单预设时返回 `{ preset, label, opts }`；否则返回
 * `{ preset: "custom", label: "自定义", cron }`。
 *
 * 「运行方式」列直接用返回的 label（中文），员工看不到 cron 字符串。
 */
export function cronToPreset(cron: string): CronDescription {
  const raw = (cron ?? "").trim();
  const fields = raw.split(/\s+/).filter(Boolean);

  // 只认 5 位（分 时 日 月 周）；6 位（带秒）与描述符一律当自定义
  if (fields.length !== 5) {
    return { preset: "custom", label: CUSTOM_LABEL, cron: raw };
  }

  const [minuteField, hourField, domField, monthField, dowField] = fields;

  // 每隔 N 分钟： */N * * * *
  if (
    /^\*\/\d+$/.test(minuteField) &&
    hourField === "*" &&
    domField === "*" &&
    monthField === "*" &&
    dowField === "*"
  ) {
    const minutes = Number(minuteField.slice(2));
    if (Number.isInteger(minutes) && minutes > 0) {
      return {
        preset: "interval",
        label: `每隔 ${minutes} 分钟`,
        opts: { minutes },
      };
    }
  }

  const minuteOk = isPlainInt(minuteField);
  const hourOk = isPlainInt(hourField);

  // 每天： M H * * *
  if (minuteOk && hourOk && domField === "*" && monthField === "*" && dowField === "*") {
    const minute = Number(minuteField);
    const hour = Number(hourField);
    if (minute >= 0 && minute <= 59 && hour >= 0 && hour <= 23) {
      return {
        preset: "daily",
        label: `每天 ${pad2(hour)}:${pad2(minute)}`,
        opts: { hour, minute },
      };
    }
  }

  // 每周： M H * * D
  if (minuteOk && hourOk && domField === "*" && monthField === "*" && isPlainInt(dowField)) {
    const minute = Number(minuteField);
    const hour = Number(hourField);
    const dow = Number(dowField);
    if (minute >= 0 && minute <= 59 && hour >= 0 && hour <= 23 && dow >= 0 && dow <= 7) {
      // cron 星期位 0=周日 → 业务口径 7=周日
      const weekday = dow === 0 ? 7 : dow;
      return {
        preset: "weekly",
        label: `每${weekdayLabel(weekday)} ${pad2(hour)}:${pad2(minute)}`,
        opts: { weekday, hour, minute },
      };
    }
  }

  return { preset: "custom", label: CUSTOM_LABEL, cron: raw };
}

/**
 * describeRunMode —— cron 表达式 → 中文运行方式文案（列表页「运行方式」列）。
 *
 * 命中简单预设给业务文案，其余一律「自定义」。
 */
export function describeRunMode(cron: string): string {
  return cronToPreset(cron).label;
}

// ---------------------------------------------------------------------------
// 时间文本 ↔ 时分（表单里的「时间」输入用 HH:mm 文本）
// ---------------------------------------------------------------------------

/**
 * formatClockTime —— 时分 → "HH:mm" 文本（表单回填用）。
 *
 * @param hour   0-23
 * @param minute 0-59
 */
export function formatClockTime(hour: number, minute: number): string {
  const h = clampInt(hour, 0, 23, DEFAULT_HOUR);
  const m = clampInt(minute, 0, 59, DEFAULT_MINUTE);
  return `${pad2(h)}:${pad2(m)}`;
}

/**
 * parseClockTime —— "HH:mm" 文本 → 时分（非法文本回落默认 09:00）。
 *
 * @param value 形如 "09:00" 的时间文本
 */
export function parseClockTime(value: string): { hour: number; minute: number } {
  const matched = /^(\d{1,2}):(\d{1,2})$/.exec((value ?? "").trim());
  if (!matched) {
    return { hour: DEFAULT_HOUR, minute: DEFAULT_MINUTE };
  }
  return {
    hour: clampInt(Number(matched[1]), 0, 23, DEFAULT_HOUR),
    minute: clampInt(Number(matched[2]), 0, 59, DEFAULT_MINUTE),
  };
}

// ---------------------------------------------------------------------------
// 形状校验（仅「高级设置」里手写表达式时用）
// ---------------------------------------------------------------------------

/** 单个 cron 字段允许的字符集（数字 / * / , / -） */
const CRON_FIELD_PATTERN = /^[0-9*/,-]+$/;

/** 各字段的取值域（5 位口径：分 时 日 月 周） */
const CRON_FIELD_RANGES: Array<[number, number]> = [
  [0, 59], // 分
  [0, 23], // 时
  [1, 31], // 日
  [1, 12], // 月
  [0, 7], // 星期
];

/**
 * isValidCronShape —— cron 表达式的**基本形状**校验。
 *
 * 只做形状检查（字段个数 + 字符集 + 纯数字字段的取值域），
 * 不做完整语义解析 —— 真正的合法性由后端 robfig/cron 判定并返回可读错误。
 *
 * @param cron 待校验的表达式（5 位或 6 位带秒）
 */
export function isValidCronShape(cron: string): boolean {
  const raw = (cron ?? "").trim();
  if (!raw) return false;

  const fields = raw.split(/\s+/).filter(Boolean);
  if (fields.length !== 5 && fields.length !== 6) return false;
  if (!fields.every((field) => CRON_FIELD_PATTERN.test(field))) return false;

  // 6 位（带秒）时整体右移一位：秒 分 时 日 月 周
  const ranged = fields.length === 6 ? fields.slice(1) : fields;
  for (let i = 0; i < ranged.length; i += 1) {
    const field = ranged[i];
    if (!isPlainInt(field)) continue; // 复合写法（* / , -）不做取值域判定
    const [min, max] = CRON_FIELD_RANGES[i];
    const value = Number(field);
    if (value < min || value > max) return false;
  }
  return true;
}
