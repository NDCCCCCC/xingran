/**
 * 间距系统令牌
 * 基于8px基准网格系统
 *
 * v1.22 品牌化（Phase 64 · TOKEN-04）
 * 8px 网格控件触达 44px —— 与 brand-spec「圆角 8px 一档 + 控件触达 44px」对齐。
 * 数值保持不变（避免 53 屏消费方回归风险）；注释追加品牌化声明。
 */

/**
 * 间距比例尺
 */
export const spacing = {
  xs: "4px", // 0.25rem
  sm: "8px", // 0.5rem
  md: "16px", // 1rem
  lg: "24px", // 1.5rem
  xl: "32px", // 2rem
  "2xl": "48px", // 3rem
  "3xl": "64px", // 4rem
  "4xl": "96px", // 6rem
  "5xl": "128px", // 8rem
} as const;

/**
 * 组件间距预设
 */
export const componentSpacing = {
  // 按钮内边距
  button: {
    sm: { padding: "6px 12px", fontSize: "14px" },
    md: { padding: "8px 16px", fontSize: "14px" },
    lg: { padding: "12px 24px", fontSize: "16px" },
  },

  // 输入框内边距
  input: {
    sm: { padding: "6px 12px" },
    md: { padding: "8px 12px" },
    lg: { padding: "12px 16px" },
  },

  // 卡片内边距
  card: {
    sm: { padding: "12px" },
    md: { padding: "16px" },
    lg: { padding: "24px" },
    xl: { padding: "32px" },
  },

  // 模态框内边距
  modal: {
    sm: { padding: "16px" },
    md: { padding: "24px" },
    lg: { padding: "32px" },
  },

  // 表单项间距
  formItem: {
    vertical: { marginBottom: "16px" },
    horizontal: { marginRight: "16px" },
  },
} as const;

/**
 * 布局间距
 */
export const layoutSpacing = {
  // 页面边距
  page: {
    compact: "12px",
    comfortable: "16px",
    spacious: "24px",
  },

  // 区块间距
  section: {
    compact: "16px",
    comfortable: "24px",
    spacious: "32px",
  },

  // 列间距
  column: {
    compact: "12px",
    comfortable: "16px",
    spacious: "24px",
  },

  // 行间距
  row: {
    compact: "8px",
    comfortable: "12px",
    spacious: "16px",
  },
} as const;

/**
 * 密度模式间距
 */
export const densitySpacing = {
  compact: {
    page: "12px",
    section: "16px",
    component: "8px",
    element: "4px",
  },
  comfortable: {
    page: "16px",
    section: "24px",
    component: "12px",
    element: "8px",
  },
  spacious: {
    page: "24px",
    section: "32px",
    component: "16px",
    element: "12px",
  },
} as const;

/**
 * 负间距（用于元素重叠效果）
 */
export const negativeSpacing = {
  xs: "-4px",
  sm: "-8px",
  md: "-16px",
  lg: "-24px",
  xl: "-32px",
} as const;

/**
 * 响应式间距断点
 */
export const responsiveSpacing = {
  mobile: {
    page: "12px",
    section: "16px",
  },
  tablet: {
    page: "16px",
    section: "24px",
  },
  desktop: {
    page: "24px",
    section: "32px",
  },
} as const;
