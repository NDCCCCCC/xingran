/**
 * 阴影系统令牌
 *
 * v1.22 品牌化（Phase 64 · TOKEN-04）
 * 调性：奶油底衬白卡双层纸感 —— 阴影色相由中性黑转深绿低饱和
 * 强度不变仅换色（rgba(0,0,0,*) → rgba(15,46,27,*)），保持阴影层次清晰。
 *
 * 旧主题组件（neumorphicShadows / glassShadows）将于 Phase 65 移除。
 */

/**
 * 基础阴影（深绿低饱和色相）
 */
export const shadows = {
  // 无阴影
  none: "none",

  // 超小阴影
  xs: "0 1px 2px 0 rgba(15, 46, 27, 0.05)",

  // 小阴影
  sm: "0 1px 3px 0 rgba(15, 46, 27, 0.1), 0 1px 2px -1px rgba(15, 46, 27, 0.1)",

  // 中阴影
  md: "0 4px 6px -1px rgba(15, 46, 27, 0.1), 0 2px 4px -2px rgba(15, 46, 27, 0.1)",

  // 大阴影
  lg: "0 10px 15px -3px rgba(15, 46, 27, 0.1), 0 4px 6px -4px rgba(15, 46, 27, 0.1)",

  // 超大阴影
  xl: "0 20px 25px -5px rgba(15, 46, 27, 0.1), 0 8px 10px -6px rgba(15, 46, 27, 0.1)",

  // 2xl阴影
  "2xl": "0 25px 50px -12px rgba(15, 46, 27, 0.25)",

  // 内阴影
  inner: "inset 0 2px 4px 0 rgba(15, 46, 27, 0.05)",
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
 * Phase 65 收敛 —— purple/blue 项与多主题共存时保留
 */
export const glassShadows = {
  // 柔和阴影
  soft: "0 8px 32px rgba(15, 46, 27, 0.15)",

  // 光晕效果
  glow: {
    /** 品牌绿（v1.22 品牌化） */
    primary: "0 0 20px rgba(21, 96, 49, 0.4)",
    purple: "0 0 20px rgba(168, 85, 247, 0.4)",
    blue: "0 0 20px rgba(59, 130, 246, 0.4)",
  },

  // 浮动阴影
  floating: "0 12px 40px rgba(15, 46, 27, 0.12)",
} as const;

/**
 * 彩色阴影（v1.22 品牌化）
 * primary 替换为品牌绿 rgba(21,96,49)；
 * success/warning/error 语义对保留，仅在阴影使用场景切换色相到品牌色域
 */
export const coloredShadows = {
  /** 品牌绿（v1.22 替换原 indigo） */
  primary: "0 4px 14px rgba(21, 96, 49, 0.4)",
  /** 铜金点缀（克制点缀，新增） */
  copper: "0 4px 14px rgba(192, 144, 88, 0.4)",
  success: "0 4px 14px rgba(45, 137, 73, 0.4)",
  warning: "0 4px 14px rgba(176, 122, 32, 0.4)",
  error: "0 4px 14px rgba(186, 54, 48, 0.4)",
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
 * 圆角令牌（v1.22 品牌化 · TOKEN-04）
 * 控件 8px 一档落实；新增 control 别名等同 md；大面板 12-16px 一档；不混用混合圆角体系。
 */
export const radius = {
  none: "0",
  sm: "4px",
  /** 控件一档（按钮/输入框/Tag）— 与 brand-spec「控件 8px 一档」对齐 */
  control: "8px",
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
