/**
 * 资产对账模块 —— 数据字典标签解析工具(字典驱动)
 *
 * 背景:对账模块多处渲染枚举文本(冲突类型 / 严重度 / 例外动作),
 *   早期把文案硬编码在组件里 —— 管理员在字典管理页新增枚举项后,
 *   前端既显示不出新项,也无法选到新项。
 *   本模块把「取值 → 展示文案」的解析收敛到一处,统一三级回退口径:
 *
 *   1) 字典命中(dictValue → dictLabel):优先用管理员可维护的字典文案;
 *   2) 静态回退映射(组件内既有常量):字典为空/未命中时保持原展示行为;
 *   3) 原始值:两者皆无时显示原值(不吞成空串)。
 *
 * 与 pages/asset/reconciliation/exceptions/index.tsx 的 buildDictLabelMap 同口径
 *   (该处是页面内私有实现);这里抽成共享模块,供对账组件与页面共用,避免重复造。
 */
import type { DictItem } from "@/hooks/useDict";

/**
 * 冲突类型静态回退文案(字典为空/未命中时才用)。
 * 字典 asset_reconciliation_conflict_type 为准 —— 其标签形如
 * "B类-物理有/责任人无",与此处文案不一致,故正常路径走字典。
 */
export const CONFLICT_TYPE_LABELS: Record<string, string> = {
  A: "正常",
  B: "物理有/责任人无",
  C: "物理有/责任人不一致",
  D: "物理无/责任人有",
  E: "三路均无",
  F: "AD 单独不一致",
};

/** 字典索引:dictValue → 展示文案 / 列表样式 / 自定义样式 */
export interface DictLabelIndex {
  /** dictValue → dictLabel */
  labels: Record<string, string>;
  /** dictValue → listClass(antd Tag 预设色) */
  listClass: Record<string, string | undefined>;
  /** dictValue → cssClass(自定义样式类名) */
  cssClass: Record<string, string | undefined>;
}

/** 下拉选项 */
export interface DictOption {
  label: string;
  value: string;
}

/** antd Tag 支持的预设色 */
const TAG_COLOR_PRESETS = ["success", "processing", "error", "warning", "default", "info"];

/**
 * 样式关键字 → antd Tag 预设色。
 * 字典的 listClass/cssClass 可能是 "primary"/"danger"/"el-tag--success" 等写法,
 * 这里按关键字归一,归不中再走静态回退色板。
 */
const COLOR_KEYWORD_HINTS: Array<[RegExp, string]> = [
  [/danger|error/, "error"],
  [/warn/, "warning"],
  [/success/, "success"],
  [/info|processing|primary/, "processing"],
];

/** 把字典项数组转成 dictValue 索引(空字典返回空索引) */
export function buildDictLabelIndex(items: DictItem[] | undefined): DictLabelIndex {
  const index: DictLabelIndex = { labels: {}, listClass: {}, cssClass: {} };
  for (const item of items ?? []) {
    if (!item?.dictValue) continue;
    index.labels[item.dictValue] = item.dictLabel;
    index.listClass[item.dictValue] = item.listClass;
    index.cssClass[item.dictValue] = item.cssClass;
  }
  return index;
}

/** 字典项数组 → 下拉选项(dictValue 为值,dictLabel 为文案) */
export function buildDictOptions(items: DictItem[] | undefined): DictOption[] {
  return (items ?? [])
    .filter((item) => Boolean(item?.dictValue))
    .map((item) => ({ label: item.dictLabel, value: item.dictValue }));
}

/**
 * 三级回退取文案:字典 → 静态映射 → 原始值。
 * 空值统一显示 "-"(不吞成空串)。
 */
export function resolveDictLabel(
  value: string | null | undefined,
  labels: Record<string, string>,
  fallbackLabels?: Record<string, string>
): string {
  if (!value) return "-";
  return labels[value] ?? fallbackLabels?.[value] ?? value;
}

/**
 * 三级回退取 antd Tag 颜色:字典 listClass/cssClass → 静态回退色 → default。
 */
export function resolveDictTagColor(
  value: string | null | undefined,
  index: DictLabelIndex,
  fallbackColor?: string
): string {
  if (!value) return fallbackColor ?? "default";
  const listClass = index.listClass[value];
  const cssClass = index.cssClass[value];
  // 1) listClass 直接命中 antd 预设色(primary 是字典写法,antd 叫 processing)
  if (listClass) {
    const normalized = listClass === "primary" ? "processing" : listClass;
    if (TAG_COLOR_PRESETS.includes(normalized)) return normalized;
  }
  // 2) listClass / cssClass 关键字推断
  for (const source of [listClass, cssClass]) {
    if (!source) continue;
    for (const [pattern, color] of COLOR_KEYWORD_HINTS) {
      if (pattern.test(source)) return color;
    }
  }
  // 3) 静态回退色板
  return fallbackColor ?? "default";
}
