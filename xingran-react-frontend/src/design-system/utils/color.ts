/**
 * 统一的颜色工具库
 * Unified Color Utilities
 *
 * 整合了颜色处理相关的所有函数，避免代码重复
 */

/**
 * 颜色变体接口
 */
export interface ColorVariants {
  primary: string;
  primaryHover: string;
  primaryLight: string;
  primaryLighter: string;
}

/**
 * 计算颜色的亮度（用于判断是深色还是浅色）
 * 基于 WCAG 2.0 规范的相对亮度计算
 *
 * @param hexColor HEX格式的颜色值（如 #1E293B）
 * @returns 亮度值 (0-1)，<0.5 为深色，>=0.5 为浅色
 */
export function getLuminance(hexColor: string | undefined): number {
  if (!hexColor || typeof hexColor !== "string") {
    return 0; // 默认返回深色
  }
  const hex = hexColor.replace("#", "");
  const r = parseInt(hex.substring(0, 2), 16) / 255;
  const g = parseInt(hex.substring(2, 4), 16) / 255;
  const b = parseInt(hex.substring(4, 6), 16) / 255;

  const toLinear = (c: number) => {
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  };

  const R = toLinear(r);
  const G = toLinear(g);
  const B = toLinear(b);

  return 0.2126 * R + 0.7152 * G + 0.0722 * B;
}

/**
 * 根据背景色自动选择合适的文字颜色
 *
 * @param backgroundColor 背景颜色（HEX格式）
 * @returns 对比色文字颜色（深色背景返回浅色，浅色背景返回深色）
 */
export function getContrastTextColor(backgroundColor: string): string {
  const luminance = getLuminance(backgroundColor);

  if (luminance < 0.5) {
    return "#F8FAFC"; // 深色背景配浅色文字
  } else {
    return "#1E293B"; // 浅色背景配深色文字
  }
}

/**
 * 生成悬停背景色
 * 通过调整 RGB 值来生成稍微亮一点或暗一点的颜色
 *
 * @param backgroundColor 原始背景色（HEX格式）
 * @returns 悬停背景色
 */
export function getHoverBackgroundColor(backgroundColor: string | undefined): string {
  if (!backgroundColor || typeof backgroundColor !== "string") {
    return "#1E293B"; // 默认返回深灰
  }
  const hex = backgroundColor.replace("#", "");
  let r = parseInt(hex.substring(0, 2), 16);
  let g = parseInt(hex.substring(2, 4), 16);
  let b = parseInt(hex.substring(4, 6), 16);

  const luminance = getLuminance(backgroundColor);

  if (luminance < 0.5) {
    // 深色背景：变亮
    r = Math.min(255, r + 20);
    g = Math.min(255, g + 20);
    b = Math.min(255, b + 20);
  } else {
    // 浅色背景：变暗
    r = Math.max(0, r - 20);
    g = Math.max(0, g - 20);
    b = Math.max(0, b - 20);
  }

  return `#${r.toString(16).padStart(2, "0")}${g.toString(16).padStart(2, "0")}${b.toString(16).padStart(2, "0")}`;
}

/**
 * 生成主色调的浅色变体
 * 将 HEX 转换为 HSL，然后生成不同的亮度变体
 *
 * @param hexColor 主色调（HEX格式）
 * @returns 颜色变体对象
 */
export function generateColorVariants(hexColor: string): ColorVariants {
  // 将 HEX 转换为 HSL
  const hex = hexColor.replace("#", "");
  const r = parseInt(hex.substring(0, 2), 16) / 255;
  const g = parseInt(hex.substring(2, 4), 16) / 255;
  const b = parseInt(hex.substring(4, 6), 16) / 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h = 0;
  let s = 0;
  const l = (max + min) / 2;

  if (max !== min) {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);

    switch (max) {
      case r:
        h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
        break;
      case g:
        h = ((b - r) / d + 2) / 6;
        break;
      case b:
        h = ((r - g) / d + 4) / 6;
        break;
    }
  }

  // HSL 转 HEX 辅助函数
  const hexFromHSL = (h: number, s: number, l: number) => {
    let r = 0;
    let g = 0;
    let b = 0;

    if (s === 0) {
      r = g = b = l;
    } else {
      const hue2rgb = (p: number, q: number, t: number) => {
        if (t < 0) t += 1;
        if (t > 1) t -= 1;
        if (t < 1 / 6) return p + (q - p) * 6 * t;
        if (t < 1 / 2) return q;
        if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
        return p;
      };

      const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
      const p = 2 * l - q;
      r = hue2rgb(p, q, h + 1 / 3);
      g = hue2rgb(p, q, h);
      b = hue2rgb(p, q, h - 1 / 3);
    }

    const toHex = (x: number) => {
      const hex = Math.round(x * 255).toString(16);
      return hex.length === 1 ? "0" + hex : hex;
    };

    return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
  };

  // hover 色：稍微降低亮度
  const hoverL = Math.max(0.1, l - 0.1);
  const primaryHover = hexFromHSL(h, s, hoverL);

  // light 色：大幅增加亮度
  const primaryLight = hexFromHSL(h, s * 0.3, 0.97);

  // lighter 色：中等亮度
  const primaryLighter = hexFromHSL(h, s * 0.5, 0.92);

  return {
    primary: hexColor,
    primaryHover,
    primaryLight,
    primaryLighter,
  };
}
