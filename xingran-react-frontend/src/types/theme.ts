/**
 * 主题系统类型定义
 */

export type ThemeType =
  "minimal" | "glassmorphism" | "neumorphism" | "flat2.0" | "luxury-quiet" | "ink-amber";

/**
 * 颜色模式
 */
export type ColorMode = "light" | "dark";

/**
 * 颜色令牌
 */
export interface ColorTokens {
  primary: string[];
  secondary: string[];
  accent: string[];
  neutral: string[];
  success: string[];
  warning: string[];
  error: string[];
  info: string[];
  processing: string[];

  background: {
    primary: string;
    secondary: string;
    tertiary: string;
    surface: string;
    elevated: string;
  };

  text: {
    primary: string;
    secondary: string;
    tertiary: string;
    disabled: string;
    inverse: string;
  };

  border: {
    primary: string;
    secondary: string;
    divider: string;
  };
}

/**
 * 间距令牌
 */
export interface SpacingTokens {
  xs: string;
  sm: string;
  md: string;
  lg: string;
  xl: string;
  "2xl": string;
  "3xl": string;
}

/**
 * 字体令牌
 */
export interface TypographyTokens {
  fontFamily: string;
  fontSize: {
    xs: string;
    sm: string;
    base: string;
    lg: string;
    xl: string;
    "2xl": string;
    "3xl": string;
  };
  fontWeight: {
    normal: string;
    medium: string;
    semibold: string;
    bold: string;
  };
  lineHeight: {
    tight: string;
    normal: string;
    relaxed: string;
  };
}

/**
 * 阴影令牌
 */
export interface ShadowTokens {
  xs: string;
  sm: string;
  md: string;
  lg: string;
  xl: string;
  "2xl": string;
  inner: string;
}

/**
 * 圆角令牌
 */
export interface RadiusTokens {
  sm: string;
  md: string;
  lg: string;
  xl: string;
  "2xl": string;
  full: string;
}

/**
 * 效果令牌
 */
export interface EffectTokens {
  // 玻璃拟态效果
  glass?: {
    blur: string;
    opacity: string;
    border: string;
    saturation?: string;
  };

  // 新拟态效果
  neumorphic?: {
    light: string;
    dark: string;
    radius: string;
    distance: string;
  };

  // 极简效果
  minimal?: {
    borderWidth: string;
    borderColor: string;
  };

  // 扁平化2.0效果
  flat2?: {
    gradient: string;
    hoverLift: string;
  };

  // 过渡动画
  transition?: {
    fast: string;
    base: string;
    slow: string;
  };
}

/**
 * 主题配置
 */
export interface ThemeConfig {
  id: ThemeType;
  name: string;
  description: string;
  colors: ColorTokens;
  spacing: SpacingTokens;
  typography: TypographyTokens;
  shadows: ShadowTokens;
  radius: RadiusTokens;
  effects: EffectTokens;
}

/**
 * 主题预设
 */
export interface ThemePreset {
  id: ThemeType;
  name: string;
  icon: string;
  description: string;
  preview: string;
}

/**
 * 主题切换动画配置
 */
export interface ThemeTransition {
  duration: number;
  easing: string;
  keyframes: Keyframe[];
}
