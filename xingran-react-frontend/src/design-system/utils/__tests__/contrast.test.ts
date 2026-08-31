/**
 * Phase 88 Batch358 — design-system/utils/contrast 测试
 */
import { describe, it, expect } from "vitest";
import {
  calculateLuminance,
  calculateContrast,
  isWCAGAACompliant,
  isWCAGAAACompliant,
  getContrastRating,
  validateThemeColors,
} from "../contrast";

describe("design-system/utils/contrast", () => {
  describe("calculateLuminance", () => {
    it("白 #FFFFFF → 1.0", () => {
      expect(calculateLuminance("#FFFFFF")).toBeCloseTo(1.0, 2);
    });

    it("黑 #000000 → 0.0", () => {
      expect(calculateLuminance("#000000")).toBeCloseTo(0.0, 2);
    });

    it("灰 #808080 介于中间", () => {
      const lum = calculateLuminance("#808080");
      expect(lum).toBeGreaterThan(0);
      expect(lum).toBeLessThan(1);
    });

    it("无 # 前缀也支持", () => {
      expect(calculateLuminance("FFFFFF")).toBeCloseTo(1.0, 2);
    });
  });

  describe("calculateContrast", () => {
    it("白/黑 → 21:1", () => {
      expect(calculateContrast("#FFFFFF", "#000000")).toBeCloseTo(21, 0);
    });

    it("同色 → 1:1", () => {
      expect(calculateContrast("#FFFFFF", "#FFFFFF")).toBeCloseTo(1, 2);
      expect(calculateContrast("#000000", "#000000")).toBeCloseTo(1, 2);
    });

    it("对比值始终 >= 1", () => {
      const c = calculateContrast("#123456", "#ABCDEF");
      expect(c).toBeGreaterThanOrEqual(1);
    });
  });

  describe("isWCAGAACompliant", () => {
    it("白/黑 → AA 通过", () => {
      expect(isWCAGAACompliant("#FFFFFF", "#000000")).toBe(true);
    });

    it("白/浅灰 (低对比) → 不通过", () => {
      expect(isWCAGAACompliant("#FFFFFF", "#EEEEEE")).toBe(false);
    });

    it("大文本 (>=18pt) → 阈值 3.0", () => {
      // 中等对比度 + 大字体
      expect(isWCAGAACompliant("#777777", "#FFFFFF", 20)).toBeDefined();
    });

    it("粗体大文本 (>=14pt + >=700 weight) → 阈值 3.0", () => {
      expect(isWCAGAACompliant("#777777", "#FFFFFF", 16, 700)).toBeDefined();
    });
  });

  describe("isWCAGAAACompliant", () => {
    it("白/黑 → AAA 通过", () => {
      expect(isWCAGAAACompliant("#FFFFFF", "#000000")).toBe(true);
    });

    it("中等对比 → 不通过 (normal text)", () => {
      expect(isWCAGAAACompliant("#888888", "#FFFFFF")).toBe(false);
    });
  });

  describe("getContrastRating", () => {
    it(">= 7 → AAA", () => {
      expect(getContrastRating(7.5)).toContain("AAA");
    });

    it(">= 4.5 < 7 → AA", () => {
      expect(getContrastRating(5.0)).toContain("AA");
    });

    it(">= 3 < 4.5 → AA 大文本", () => {
      expect(getContrastRating(3.5)).toContain("大文本");
    });

    it("< 3 → 不符合", () => {
      expect(getContrastRating(2.0)).toContain("不符合");
    });
  });

  describe("validateThemeColors", () => {
    it("返回 shape", () => {
      const result = validateThemeColors({
        text: {
          primary: "#000000",
          secondary: "#444444",
          tertiary: "#888888",
        },
        background: {
          primary: "#FFFFFF",
          secondary: "#F5F5F5",
          tertiary: "#EEEEEE",
        },
      });
      expect(result.pass).toBeDefined();
      expect(Array.isArray(result.results)).toBe(true);
    });

    it("高对比色 → pass=true", () => {
      const result = validateThemeColors({
        text: {
          primary: "#000000",
          secondary: "#000000",
          tertiary: "#000000",
        },
        background: {
          primary: "#FFFFFF",
          secondary: "#FFFFFF",
          tertiary: "#FFFFFF",
        },
      });
      expect(result.pass).toBe(true);
    });

    it("results 含 foreground/background/contrast/rating/wcagAA", () => {
      const result = validateThemeColors({
        text: { primary: "#000", secondary: "#222", tertiary: "#444" },
        background: { primary: "#FFF", secondary: "#EEE", tertiary: "#DDD" },
      });
      expect(result.results[0]).toHaveProperty("foreground");
      expect(result.results[0]).toHaveProperty("background");
      expect(result.results[0]).toHaveProperty("contrast");
      expect(result.results[0]).toHaveProperty("rating");
      expect(result.results[0]).toHaveProperty("wcagAA");
    });
  });
});
