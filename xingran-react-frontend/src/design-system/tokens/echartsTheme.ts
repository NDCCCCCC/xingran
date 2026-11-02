/**
 * ECharts 品牌主题令牌（v1.22 · Phase 66 · COMP-04）
 *
 * Canvas 不解析 CSS var —— 此模块从 xingranBrand 组装 canvas 安全的
 * 字面量色值，供 ECharts 系列 / 热力 / 轴线使用。
 *
 * 色值来源（D-02 单一来源）：
 * - xingranBrand.ts（brand-spec.md 像素实测 + OKLch 派生 + WCAG 验证）
 */

import { xingranBrand } from "./colors";

/**
 * 系列色（绿金梯度，ROADMAP SC#4 顺序）
 * 0=品牌主色 1=次级绿 2=铜金主 3=深底图标绿 4=铜金浅
 */
export const brandSeriesColors: readonly string[] = [
  xingranBrand.greenPrimary, // #156031
  xingranBrand.green[200], // #3B784C
  xingranBrand.copperAccent, // #C09058
  xingranBrand.green[100], // #598E5E
  xingranBrand.copper[700], // #C89868
] as const;

/**
 * 热力梯度（低 → 高）
 * 0=#E9EFEB 绿灰淡彩, 1=#598E5E 深底绿, 2=#156031 品牌主色,
 * 3=#C09058 铜金主, 4=#B88850 铜金深
 */
export const brandHeatRamp: readonly string[] = [
  xingranBrand.green[50], // #E9EFEB
  xingranBrand.green[100], // #598E5E
  xingranBrand.greenPrimary, // #156031
  xingranBrand.copperAccent, // #C09058
  xingranBrand.copper[500], // #B88850
] as const;

/** 轴线色（暖灰描边） */
export const brandAxisLine: string = xingranBrand.cream.border; // #DBD7CE

/** 分割线色（绿灰淡彩） */
export const brandSplitLine: string = xingranBrand.green[50]; // #E9EFEB

/** 文本默认色（次级文字） */
export const brandTextStyle: string = xingranBrand.cream.mutedStrong; // #64645C

/** 面积渐变端色（品牌绿 10% 透明度） */
export const brandAreaFade: string = "rgba(21, 96, 49, 0.1)";

/**
 * 热力图零态占位色（max<=0 时单色）
 * 使用 cream.borderStrong (#C2BDB2) —— 暖灰中性与 bg-elevated #1f3524 对比 ≈5.5:1
 */
export const brandHeatZero: string = xingranBrand.cream.borderStrong; // #C2BDB2
