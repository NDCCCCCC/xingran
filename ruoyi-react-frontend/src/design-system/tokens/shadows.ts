/**
 * 阴影系统令牌
 */

/**
 * 基础阴影
 */
export const shadows = {
  // 无阴影
  none: "none",

  // 超小阴影
  xs: "0 1px 2px 0 rgba(0, 0, 0, 0.05)",

  // 小阴影
  sm: "0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1)",

  // 中阴影
  md: "0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1)",

  // 大阴影
  lg: "0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1)",

  // 超大阴影
  xl: "0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1)",

  // 2xl阴影
  "2xl": "0 25px 50px -12px rgba(0, 0, 0, 0.25)",

  // 内阴影
  inner: "inset 0 2px 4px 0 rgba(0, 0, 0, 0.05)",
} as const;

/**
 * 新拟态阴影（Neumorphism）
 */
export const neumorphicShadows = {
  // 凸起效果（默认）
  convex: {
    sm: "6px 6px 12px rgba(163, 177, 198, 0.6), -6px -6px 12px rgba(255, 255, 255, 0.8)",
    md: "8px 8px 16px rgba(163, 177, 198, 0.6), -8px -8px 16px rgba(255, 255, 255, 0.8)",
    lg: "12px 12px 24px rgba(163, 177, 198, 0.6), -12px -12px 24px rgba(255, 255, 255, 0.8)",
  },

  // 凹陷效果（按压）
  concave: {
    sm: "inset 4px 4px 8px rgba(163, 177, 198, 0.6), inset -4px -4px 8px rgba(255, 255, 255, 0.8)",
    md: "inset 6px 6px 12px rgba(163, 177, 198, 0.6), inset -6px -6px 12px rgba(255, 255, 255, 0.8)",
    lg: "inset 8px 8px 16px rgba(163, 177, 198, 0.6), inset -8px -8px 16px rgba(255, 255, 255, 0.8)",
  },

  // 扁平效果
  flat: {
    sm: "2px 2px 4px rgba(163, 177, 198, 0.4), -2px -2px 4px rgba(255, 255, 255, 0.9)",
    md: "4px 4px 8px rgba(163, 177, 198, 0.4), -4px -4px 8px rgba(255, 255, 255, 0.9)",
    lg: "6px 6px 12px rgba(163, 177, 198, 0.4), -6px -6px 12px rgba(255, 255, 255, 0.9)",
  },
} as const;

/**
 * 玻璃拟态阴影（Glassmorphism）
 */
export const glassShadows = {
  // 柔和阴影
  soft: "0 8px 32px rgba(31, 38, 135, 0.15)",

  // 光晕效果
  glow: {
    primary: "0 0 20px rgba(99, 102, 241, 0.4)",
    purple: "0 0 20px rgba(168, 85, 247, 0.4)",
    blue: "0 0 20px rgba(59, 130, 246, 0.4)",
  },

  // 浮动阴影
  floating: "0 12px 40px rgba(0, 0, 0, 0.12)",
} as const;

/**
 * 彩色阴影
 */
export const coloredShadows = {
  primary: "0 4px 14px rgba(59, 130, 246, 0.4)",
  purple: "0 4px 14px rgba(168, 85, 247, 0.4)",
  success: "0 4px 14px rgba(34, 197, 94, 0.4)",
  warning: "0 4px 14px rgba(245, 158, 11, 0.4)",
  error: "0 4px 14px rgba(239, 68, 68, 0.4)",
} as const;

/**
 * 方向性阴影
 */
export const directionalShadows = {
  top: "0 -4px 6px -1px rgba(0, 0, 0, 0.1)",
  bottom: "0 4px 6px -1px rgba(0, 0, 0, 0.1)",
  left: "-4px 0 6px -1px rgba(0, 0, 0, 0.1)",
  right: "4px 0 6px -1px rgba(0, 0, 0, 0.1)",
} as const;

/**
 * 圆角令牌
 */
export const radius = {
  none: "0",
  sm: "4px",
  md: "8px",
  lg: "12px",
  xl: "16px",
  "2xl": "24px",
  "3xl": "32px",
  full: "9999px",
} as const;

/**
 * 模糊效果
 */
export const blur = {
  none: "0",
  sm: "4px",
  md: "8px",
  lg: "16px",
  xl: "24px",
  "2xl": "40px",
  "3xl": "64px",
} as const;
