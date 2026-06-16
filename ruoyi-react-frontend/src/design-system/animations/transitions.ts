/**
 * 过渡动画配置
 */

/**
 * 缓动函数
 */
export const easings = {
  // 线性
  linear: "linear",

  // 标准缓动
  ease: "ease",

  // 优雅缓动（推荐）
  easeInOut: "cubic-bezier(0.4, 0, 0.2, 1)",
  easeOut: "cubic-bezier(0, 0, 0.2, 1)",
  easeIn: "cubic-bezier(0.4, 0, 1, 1)",

  // 弹性缓动
  bouncy: "cubic-bezier(0.34, 1.56, 0.64, 1)",
  bouncyIn: "cubic-bezier(0.34, 1.56, 0.64, 1)",
  bouncyOut: "cubic-bezier(0.34, 1.56, 0.64, 1)",

  // 平滑缓动
  smooth: "cubic-bezier(0.25, 0.1, 0.25, 1)",

  // 快速响应
  snappy: "cubic-bezier(0.19, 1, 0.22, 1)",
} as const;

/**
 * 过渡时长
 */
export const durations = {
  instant: "0ms",
  fast: "150ms",
  base: "200ms",
  normal: "300ms",
  slow: "500ms",
  slower: "700ms",
} as const;

/**
 * 过渡延迟
 */
export const delays = {
  none: "0ms",
  short: "100ms",
  medium: "200ms",
  long: "300ms",
} as const;

/**
 * 常用过渡预设
 */
export const transitions = {
  // 快速过渡（用于hover等即时反馈）
  fast: `${durations.fast} ${easings.easeOut}`,

  // 基础过渡（用于一般状态变化）
  base: `${durations.base} ${easings.easeInOut}`,

  // 标准过渡（用于页面元素进入/退出）
  normal: `${durations.normal} ${easings.easeInOut}`,

  // 慢速过渡（用于大幅动画）
  slow: `${durations.slow} ${easings.easeInOut}`,

  // 弹性过渡（用于强调动画）
  bouncy: `${durations.normal} ${easings.bouncy}`,

  // 主题切换过渡
  theme: `${durations.slow} ${easings.easeInOut}`,

  // 布局切换过渡
  layout: `${durations.normal} ${easings.easeInOut}`,

  // 侧边栏过渡
  sidebar: `${durations.normal} ${easings.easeInOut}`,

  // 模态框过渡
  modal: `${durations.normal} ${easings.bouncy}`,

  // 下拉菜单过渡
  dropdown: `${durations.fast} ${easings.easeOut}`,

  // 标签页过渡
  tab: `${durations.base} ${easings.easeInOut}`,
} as const;

/**
 * 属性特定过渡
 */
export const propertyTransitions = {
  // 颜色过渡
  colors: `color ${durations.base} ${easings.easeOut}, background-color ${durations.base} ${easings.easeOut}, border-color ${durations.base} ${easings.easeOut}`,

  // 阴影过渡
  shadow: `box-shadow ${durations.base} ${easings.easeOut}`,

  // 变换过渡
  transform: `transform ${durations.normal} ${easings.easeInOut}`,

  // 透明度过渡
  opacity: `opacity ${durations.base} ${easings.easeOut}`,

  // 全过渡（所有属性）
  all: `all ${durations.base} ${easings.easeInOut}`,

  // 组合过渡（常用组合）
  common: `color ${durations.base} ${easings.easeOut}, background-color ${durations.base} ${easings.easeOut}, border-color ${durations.base} ${easings.easeOut}, box-shadow ${durations.base} ${easings.easeOut}`,
} as const;

/**
 * Stagger延迟配置（用于序列动画）
 */
export const staggerDelays = {
  none: "0ms",
  fast: "50ms",
  base: "100ms",
  slow: "150ms",
} as const;

/**
 * 获取stagger延迟
 * @param index 元素索引
 * @param delay 延迟间隔
 */
export function getStaggerDelay(index: number, delay: keyof typeof staggerDelays = "base"): number {
  const delayMs = parseInt(staggerDelays[delay]);
  return index * delayMs;
}

/**
 * 过渡工具函数
 */
export const transitionUtils = {
  /**
   * 生成自定义过渡
   */
  create: (
    properties: string | string[],
    duration: keyof typeof durations = "base",
    easing: keyof typeof easings = "easeInOut"
  ): string => {
    const props = Array.isArray(properties) ? properties.join(", ") : properties;
    return `${props} ${durations[duration]} ${easings[easing]}`;
  },

  /**
   * 生成stagger过渡
   */
  stagger: (
    property: string,
    index: number,
    delay: keyof typeof staggerDelays = "base",
    duration: keyof typeof durations = "base",
    easing: keyof typeof easings = "easeInOut"
  ): string => {
    const delayMs = getStaggerDelay(index, delay);
    return `${property} ${durations[duration]} ${easing} ${delayMs}ms`;
  },
} as const;
