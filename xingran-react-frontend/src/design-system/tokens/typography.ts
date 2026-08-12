/**
 * 字体系统令牌
 */

/**
 * 字体家族
 */
export const fontFamily = {
  // 默认字体（系统字体栈）
  sans: [
    "-apple-system",
    "BlinkMacSystemFont",
    '"Segoe UI"',
    "Roboto",
    '"Helvetica Neue"',
    "Arial",
    "sans-serif",
    '"Apple Color Emoji"',
    '"Segoe UI Emoji"',
    '"Segoe UI Symbol"',
  ].join(", "),

  // 等宽字体
  mono: [
    "ui-monospace",
    "SFMono-Regular",
    "Menlo",
    "Monaco",
    "Consolas",
    '"Liberation Mono"',
    '"Courier New"',
    "monospace",
  ].join(", "),

  // 衬线字体
  serif: [
    "ui-serif",
    "Georgia",
    "Cambria",
    '"Times New Roman"',
    "Times",
    "serif",
  ].join(", "),
} as const;

/**
 * 字体大小
 */
export const fontSize = {
  xs: "12px",   // 0.75rem
  sm: "14px",   // 0.875rem
  base: "16px", // 1rem
  lg: "18px",   // 1.125rem
  xl: "20px",   // 1.25rem
  "2xl": "24px", // 1.5rem
  "3xl": "30px", // 1.875rem
  "4xl": "36px", // 2.25rem
  "5xl": "48px", // 3rem
  "6xl": "60px", // 3.75rem
} as const;

/**
 * 字重
 */
export const fontWeight = {
  thin: "100",
  extralight: "200",
  light: "300",
  normal: "400",
  medium: "500",
  semibold: "600",
  bold: "700",
  extrabold: "800",
  black: "900",
} as const;

/**
 * 行高
 */
export const lineHeight = {
  none: "1",
  tight: "1.25",
  snug: "1.375",
  normal: "1.5",
  relaxed: "1.625",
  loose: "2",
} as const;

/**
 * 字母间距
 */
export const letterSpacing = {
  tighter: "-0.05em",
  tight: "-0.025em",
  normal: "0",
  wide: "0.025em",
  wider: "0.05em",
  widest: "0.1em",
} as const;

/**
 * 排版预设
 */
export const typography = {
  // 标题
  h1: {
    fontSize: fontSize["3xl"],
    fontWeight: fontWeight.bold,
    lineHeight: lineHeight.tight,
  },
  h2: {
    fontSize: fontSize["2xl"],
    fontWeight: fontWeight.semibold,
    lineHeight: lineHeight.tight,
  },
  h3: {
    fontSize: fontSize.xl,
    fontWeight: fontWeight.semibold,
    lineHeight: lineHeight.snug,
  },
  h4: {
    fontSize: fontSize.lg,
    fontWeight: fontWeight.medium,
    lineHeight: lineHeight.snug,
  },
  h5: {
    fontSize: fontSize.base,
    fontWeight: fontWeight.medium,
    lineHeight: lineHeight.normal,
  },
  h6: {
    fontSize: fontSize.sm,
    fontWeight: fontWeight.medium,
    lineHeight: lineHeight.normal,
  },

  // 正文
  body: {
    fontSize: fontSize.base,
    fontWeight: fontWeight.normal,
    lineHeight: lineHeight.normal,
  },
  bodySmall: {
    fontSize: fontSize.sm,
    fontWeight: fontWeight.normal,
    lineHeight: lineHeight.normal,
  },
  bodyLarge: {
    fontSize: fontSize.lg,
    fontWeight: fontWeight.normal,
    lineHeight: lineHeight.relaxed,
  },

  // 标签
  label: {
    fontSize: fontSize.sm,
    fontWeight: fontWeight.medium,
    lineHeight: lineHeight.normal,
  },
  labelSmall: {
    fontSize: fontSize.xs,
    fontWeight: fontWeight.medium,
    lineHeight: lineHeight.normal,
  },

  // 代码
  code: {
    fontFamily: fontFamily.mono,
    fontSize: fontSize.sm,
    fontWeight: fontWeight.normal,
  },
  codeInline: {
    fontFamily: fontFamily.mono,
    fontSize: fontSize.sm,
    fontWeight: fontWeight.normal,
    padding: "2px 6px",
  },

  // 链接
  link: {
    fontSize: fontSize.base,
    fontWeight: fontWeight.normal,
    textDecoration: "underline",
  },

  // 按钮
  button: {
    fontSize: fontSize.base,
    fontWeight: fontWeight.medium,
  },
  buttonSmall: {
    fontSize: fontSize.sm,
    fontWeight: fontWeight.medium,
  },
  buttonLarge: {
    fontSize: fontSize.lg,
    fontWeight: fontWeight.medium,
  },

  // 输入框
  input: {
    fontSize: fontSize.base,
    fontWeight: fontWeight.normal,
  },

  // 辅助文本
  caption: {
    fontSize: fontSize.xs,
    fontWeight: fontWeight.normal,
    lineHeight: lineHeight.normal,
  },
  overline: {
    fontSize: fontSize.xs,
    fontWeight: fontWeight.medium,
    letterSpacing: letterSpacing.wider,
    textTransform: "uppercase",
  },
} as const;

/**
 * 文本对齐
 */
export const textAlign = {
  left: "left",
  center: "center",
  right: "right",
  justify: "justify",
} as const;

/**
 * 文本变换
 */
export const textTransform = {
  none: "none",
  uppercase: "uppercase",
  lowercase: "lowercase",
  capitalize: "capitalize",
} as const;

/**
 * 文本装饰
 */
export const textDecoration = {
  none: "none",
  underline: "underline",
  "line-through": "line-through",
} as const;
