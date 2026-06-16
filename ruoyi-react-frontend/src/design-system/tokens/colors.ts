/**
 * 颜色系统令牌
 * 提供统一的颜色定义，支持不同主题使用
 */

/**
 * 基础色阶
 */
export const baseColors = {
  // 主色阶
  primary: {
    50: "#eff6ff",
    100: "#dbeafe",
    200: "#bfdbfe",
    300: "#93c5fd",
    400: "#60a5fa",
    500: "#3b82f6",
    600: "#2563eb",
    700: "#1d4ed8",
    800: "#1e40af",
    900: "#1e3a8a",
    950: "#172554",
  },

  // 紫色系（用于玻璃拟态）
  purple: {
    50: "#faf5ff",
    100: "#f3e8ff",
    200: "#e9d5ff",
    300: "#d8b4fe",
    400: "#c084fc",
    500: "#a855f7",
    600: "#9333ea",
    700: "#7e22ce",
    800: "#6b21a8",
    900: "#581c87",
  },

  // 靛青色系（用于新拟态）
  indigo: {
    50: "#eef2ff",
    100: "#e0e7ff",
    200: "#c7d2fe",
    300: "#a5b4fc",
    400: "#818cf8",
    500: "#6366f1",
    600: "#4f46e5",
    700: "#4338ca",
    800: "#3730a3",
    900: "#312e81",
  },

  // 灰色系
  gray: {
    50: "#f9fafb",
    100: "#f3f4f6",
    200: "#e5e7eb",
    300: "#d1d5db",
    400: "#9ca3af",
    500: "#6b7280",
    600: "#4b5563",
    700: "#374151",
    800: "#1f2937",
    900: "#111827",
    950: "#030712",
  },

  // 中性色（用于新拟态背景）
  neutral: {
    50: "#fafafa",
    100: "#f4f4f5",
    200: "#e4e4e7",
    300: "#d4d4d8",
    400: "#a1a1aa",
    500: "#71717a",
    600: "#52525b",
    700: "#3f3f46",
    800: "#27272a",
    900: "#18181b",
  },

  // 新拟态专用色
  neumorphic: {
    bg: "#e0e5ec",
    light: "rgba(255, 255, 255, 0.8)",
    dark: "rgba(163, 177, 198, 0.6)",
    text: "#4a5568",
  },

  // 功能色
  success: {
    light: "#86efac",
    DEFAULT: "#22c55e",
    dark: "#16a34a",
  },

  warning: {
    light: "#fcd34d",
    DEFAULT: "#f59e0b",
    dark: "#d97706",
  },

  error: {
    light: "#fca5a5",
    DEFAULT: "#ef4444",
    dark: "#dc2626",
  },

  info: {
    light: "#93c5fd",
    DEFAULT: "#3b82f6",
    dark: "#2563eb",
  },
};

/**
 * 语义色映射
 */
export const semanticColors = {
  // 背景色
  background: {
    ...baseColors.gray,
    canvas: "#ffffff",
    overlay: "rgba(0, 0, 0, 0.5)",
  },

  // 文字色
  text: {
    primary: baseColors.gray[900],
    secondary: baseColors.gray[600],
    tertiary: baseColors.gray[500],
    disabled: baseColors.gray[400],
    inverse: "#ffffff",
    link: baseColors.primary[600],
  },

  // 边框色
  border: {
    primary: baseColors.gray[200],
    secondary: baseColors.gray[300],
    divider: baseColors.gray[200],
    focus: baseColors.primary[500],
    error: baseColors.error.DEFAULT,
  },

  // 阴影色
  shadow: {
    sm: "rgba(0, 0, 0, 0.05)",
    md: "rgba(0, 0, 0, 0.1)",
    lg: "rgba(0, 0, 0, 0.15)",
    xl: "rgba(0, 0, 0, 0.2)",
  },
} as const;

/**
 * 渐变色预设
 */
export const gradients = {
  // 蓝色渐变
  blue: {
    DEFAULT: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
    light: "linear-gradient(135deg, #a5b4fc 0%, #c4b5fd 100%)",
    dark: "linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%)",
  },

  // 紫色渐变
  purple: {
    DEFAULT: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
    light: "linear-gradient(135deg, #c4b5fd 0%, #e9d5ff 100%)",
    dark: "linear-gradient(135deg, #7c3aed 0%, #9333ea 100%)",
  },

  // 极光渐变
  aurora: {
    DEFAULT: "linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%)",
  },

  // 日落渐变
  sunset: {
    DEFAULT: "linear-gradient(135deg, #fa709a 0%, #fee140 100%)",
  },

  // 海洋渐变
  ocean: {
    DEFAULT: "linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)",
  },
} as const;

/**
 * 品牌色（可自定义）
 */
export const brandColors = {
  primary: baseColors.primary[600],
  secondary: baseColors.purple[500],
  accent: baseColors.indigo[500],
  success: baseColors.success.DEFAULT,
  warning: baseColors.warning.DEFAULT,
  error: baseColors.error.DEFAULT,
  info: baseColors.info.DEFAULT,
} as const;
