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

/**
 * v1.22 品牌化（Phase 64 · TOKEN-02）
 * xingranBrand 为 TS 侧唯一色值真相源。
 *
 * 色值来源：`655aa291-9bfe-4e94-ad5d-b3c8b2d24984/brand-spec.md`
 * - 标注「实测」直接采用
 * - 标注「推导」微调后须通过 QA-01 对比度校验
 *
 * 用途：AntdThemeBridge、组件样式、未来业务组件（Phase 66）仅从此常量读取，
 * 不再硬编码 hex 值。
 */
export const xingranBrand = {
  /**
   * 绿色梯度（6 阶，实测）
   * 从深到浅：渐变深端 / 品牌主色 / 渐变亮端 / 次级绿 / 深底图标绿 / 绿灰淡彩
   */
  green: {
    /** 渐变深端 / 侧边栏底 — oklch(0.393 0.090 152.5) */
    900: "#14532D",
    /** 品牌主色 — oklch(0.432 0.104 151.2) */
    400: "#156031",
    /** 渐变亮端 — oklch(0.459 0.106 152.4) */
    300: "#1A6839",
    /** 次级绿 / 图表系列 — oklch(0.519 0.094 151.0) */
    200: "#3B784C",
    /** 深底上的图标绿 — oklch(0.596 0.092 146.5) */
    100: "#598E5E",
    /** 绿灰淡彩（表头/斑马纹/分区底）— oklch(0.946 0.008 157.1) */
    50: "#E9EFEB",
  },

  /**
   * 品牌主色 4 变体（实测 + 推导）
   * 对比度：white on greenPrimary 7.64:1 ✓ AA | white on greenPrimaryHover 5.68:1 ✓ AA
   */
  greenPrimary: "#156031", // 主按钮 bg — 7.64:1
  greenPrimaryHover: "#2E7444", // 主按钮 hover — 5.68:1
  greenPrimaryActive: "#14542E", // 主按钮 active — 实测渐变深端
  greenPrimaryLight: "#1A6839", // 主按钮 light / 渐变亮端

  /**
   * 铜金梯度（4 阶，实测）
   * 用途：登录按钮（≥16px 半粗体）、点缀 / 描边 / 图标 / 插画亮部
   * 纪律：铜金 #C09058 不做实心主按钮（白字仅 2.85:1 不达标）
   */
  copper: {
    /** 铜金 hover（推导，只加深）— oklch(0.617 0.094 69.3) */
    400: "#AA7B42",
    /** 铜金深（大字实心按钮可用，白字 3.15:1）— oklch(0.662 0.094 69.3) */
    500: "#B88850",
    /** 铜金主（点缀/描边/图标）— oklch(0.687 0.093 69.5) */
    600: "#C09058",
    /** 铜金浅（插画亮部）— oklch(0.714 0.086 65.7) */
    700: "#C89868",
  },

  /** 铜金主 — 同 copper[600]，单独别名方便阅读 */
  copperAccent: "#C09058",

  /**
   * 奶油中性阶（实测 + 推导）
   * 用途：双层纸感 —— 奶油底 #F0ECE3 衬白卡 #FFFFFF，靠 1px 暖灰描边 #DBD7CE 分层
   */
  cream: {
    /** 画布底色（实测）— oklch(0.944 0.013 86.8) */
    canvas: "#F0ECE3",
    /** 白卡 surface（实测）— oklch(1 0 0) */
    surface: "#FFFFFF",
    /** 主文字 / 标题（实测）— oklch(0.173 0 0) */
    fg: "#101010",
    /** 次级文字（仅用于白卡上，4.99:1 ✓ AA）— oklch(0.543 0.012 106.8) */
    muted: "#707068",
    /** muted-strong（奶油底 5.06:1 ✓ AA） */
    mutedStrong: "#64645C",
    /** 描边 / 分割线（推导自 --bg 降明度）— oklch(0.880 0.013 86.8) */
    border: "#DBD7CE",
    /** 输入框描边 / 强分割 */
    borderStrong: "#C2BDB2",
    /** 表头底 / 选中行 / 斑马纹 */
    headerBg: "#E9EFEB",
    /** 斑马纹 / 浅交互底（推导自 canvas + 5% 白） */
    zebraBg: "#F7F5EE",
  },

  /**
   * 深绿底上的文字（实测）
   * 用途：侧边栏文字、强调标题点缀
   */
  onDark: {
    /** 主文字 — on #156031 对比度 7.64:1 ✓ */
    white: "#FFFFFF",
    /** 强调浅黄（侧边栏 active / 标题点缀）— on #156031 对比度 5.62:1 ✓ */
    lightYellow: "#E0E0B0",
    /** 淡黄标签底（SM2/SM3/SM4 同款） */
    paleYellow: "#FEF3C7",
  },

  /**
   * 功能色（推导 + WCAG 验证）
   * 用途：错误 / 警告 / 成功 / 信息按钮、Tag、徽章
   */
  functional: {
    /** 成功图标/大字 4.39:1 */
    success: "#2D8949",
    /** 成功实心按钮 4.89:1 */
    successSolid: "#238142",
    /** 警告文字/图标 */
    warning: "#B07A20",
    /** 警告文字（白底 5.60:1 / 淡黄底 5.03:1） */
    warningText: "#905D00",
    /** 危险/错误 — 白字 5.74:1 ✓ */
    danger: "#BA3630",
    /** 信息 — 中性蓝 */
    info: "#337AB0",
  },

  /**
   * 渐变（实测，来源 brand-spec 附录）
   */
  gradient: {
    /** 品牌面板渐变（侧边栏 / 登录页左侧） */
    brandPanel: "linear-gradient(135deg, #14532D 0%, #156031 60%, #1E6B3F 100%)",
    /** 画布渐变（登录页背景） */
    canvas: "linear-gradient(135deg, #FAF7F2 0%, #F0EBE0 100%)",
  },
} as const;
