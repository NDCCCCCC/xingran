/**
 * 关键帧动画配置
 */

/**
 * 淡入淡出动画
 */
export const fadeIn = {
  from: { opacity: 0 },
  to: { opacity: 1 },
};

export const fadeOut = {
  from: { opacity: 1 },
  to: { opacity: 0 },
};

export const fadeInUp = {
  from: {
    opacity: 0,
    transform: "translateY(20px)",
  },
  to: {
    opacity: 1,
    transform: "translateY(0)",
  },
};

export const fadeInDown = {
  from: {
    opacity: 0,
    transform: "translateY(-20px)",
  },
  to: {
    opacity: 1,
    transform: "translateY(0)",
  },
};

export const fadeInLeft = {
  from: {
    opacity: 0,
    transform: "translateX(-20px)",
  },
  to: {
    opacity: 1,
    transform: "translateX(0)",
  },
};

export const fadeInRight = {
  from: {
    opacity: 0,
    transform: "translateX(20px)",
  },
  to: {
    opacity: 1,
    transform: "translateX(0)",
  },
};

/**
 * 缩放动画
 */
export const scaleIn = {
  from: {
    opacity: 0,
    transform: "scale(0.9)",
  },
  to: {
    opacity: 1,
    transform: "scale(1)",
  },
};

export const scaleOut = {
  from: {
    opacity: 1,
    transform: "scale(1)",
  },
  to: {
    opacity: 0,
    transform: "scale(0.9)",
  },
};

export const scaleUp = {
  from: {
    transform: "scale(1)",
  },
  to: {
    transform: "scale(1.05)",
  },
};

export const scaleDown = {
  from: {
    transform: "scale(1)",
  },
  to: {
    transform: "scale(0.95)",
  },
};

/**
 * 滑动动画
 */
export const slideInUp = {
  from: {
    transform: "translateY(100%)",
  },
  to: {
    transform: "translateY(0)",
  },
};

export const slideInDown = {
  from: {
    transform: "translateY(-100%)",
  },
  to: {
    transform: "translateY(0)",
  },
};

export const slideInLeft = {
  from: {
    transform: "translateX(-100%)",
  },
  to: {
    transform: "translateX(0)",
  },
};

export const slideInRight = {
  from: {
    transform: "translateX(100%)",
  },
  to: {
    transform: "translateX(0)",
  },
};

/**
 * 旋转动画
 */
export const rotate = {
  from: {
    transform: "rotate(0deg)",
  },
  to: {
    transform: "rotate(360deg)",
  },
};

export const rotatePulse = {
  "0%, 100%": {
    transform: "rotate(0deg)",
  },
  "50%": {
    transform: "rotate(180deg)",
  },
};

/**
 * 弹跳动画
 */
export const bounce = {
  "0%, 100%": {
    transform: "translateY(0)",
  },
  "50%": {
    transform: "translateY(-25%)",
  },
};

export const bounceIn = {
  "0%": {
    opacity: 0,
    transform: "scale(0.3)",
  },
  "50%": {
    transform: "scale(1.05)",
  },
  "70%": {
    transform: "scale(0.9)",
  },
  "100%": {
    opacity: 1,
    transform: "scale(1)",
  },
};

/**
 * 脉冲动画
 */
export const pulse = {
  "0%, 100%": {
    opacity: 1,
  },
  "50%": {
    opacity: 0.5,
  },
};

export const pulseScale = {
  "0%, 100%": {
    transform: "scale(1)",
  },
  "50%": {
    transform: "scale(1.05)",
  },
};

/**
 * 摇晃动画
 */
export const shake = {
  "0%, 100%": {
    transform: "translateX(0)",
  },
  "10%, 30%, 50%, 70%, 90%": {
    transform: "translateX(-10px)",
  },
  "20%, 40%, 60%, 80%": {
    transform: "translateX(10px)",
  },
};

/**
 * 闪烁动画
 */
export const shimmer = {
  from: {
    backgroundPosition: "0% 50%",
  },
  to: {
    backgroundPosition: "100% 50%",
  },
};

/**
 * 主题切换动画
 */
export const themeTransition = {
  "0%": { opacity: 1 },
  "50%": { opacity: 0.8 },
  "100%": { opacity: 1 },
};

/**
 * 新拟态按压动画
 */
export const neumorphicPress = {
  "0%": {
    boxShadow: "8px 8px 16px rgba(163, 177, 198, 0.6), -8px -8px 16px rgba(255, 255, 255, 0.8)",
  },
  "50%": {
    boxShadow:
      "inset 6px 6px 12px rgba(163, 177, 198, 0.6), inset -6px -6px 12px rgba(255, 255, 255, 0.8)",
  },
  "100%": {
    boxShadow: "8px 8px 16px rgba(163, 177, 198, 0.6), -8px -8px 16px rgba(255, 255, 255, 0.8)",
  },
};

/**
 * 玻璃拟态光效动画
 */
export const glassGlow = {
  "0%, 100%": {
    backgroundPosition: "0% 50%",
  },
  "50%": {
    backgroundPosition: "100% 50%",
  },
};

/**
 * 加载动画
 */
export const spin = {
  from: {
    transform: "rotate(0deg)",
  },
  to: {
    transform: "rotate(360deg)",
  },
};

export const dots = {
  "0%, 20%": {
    opacity: 0,
  },
  "50%": {
    opacity: 1,
  },
  "100%": {
    opacity: 0,
  },
};

/**
 * 关键帧集合
 */
export const keyframes = {
  // 淡入淡出
  fadeIn,
  fadeOut,
  fadeInUp,
  fadeInDown,
  fadeInLeft,
  fadeInRight,

  // 缩放
  scaleIn,
  scaleOut,
  scaleUp,
  scaleDown,

  // 滑动
  slideInUp,
  slideInDown,
  slideInLeft,
  slideInRight,

  // 旋转
  rotate,
  rotatePulse,

  // 弹跳
  bounce,
  bounceIn,

  // 脉冲
  pulse,
  pulseScale,

  // 摇晃
  shake,

  // 特效
  shimmer,
  themeTransition,
  neumorphicPress,
  glassGlow,

  // 加载
  spin,
  dots,
} as const;

/**
 * 动画时长预设
 */
export const animationDurations = {
  fast: "150ms",
  base: "300ms",
  normal: "500ms",
  slow: "700ms",
} as const;
