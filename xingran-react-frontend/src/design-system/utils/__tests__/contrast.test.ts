/**
 * Phase 84 84-03b — Design-system utils contrast 测试
 */
import { describe, it, expect } from "vitest";
import {
  calculateLuminance,
  calculateContrast,
  isWCAGAACompliant,
  isWCAGAAACompliant,
  getContrastRating,
} from "../contrast";

describe("contrast utils", () => {
  it("calculateLuminance returns 0 for black", () => {
    expect(calculateLuminance("#000000")).toBe(0);
  });

  it("calculateLuminance returns 1 for white", () => {
    expect(calculateLuminance("#ffffff")).toBe(1);
  });

  it("calculateContrast black vs white approximately 21", () => {
    expect(calculateContrast("#000000", "#ffffff")).toBeGreaterThan(20);
  });

  it("isWCAGAACompliant returns boolean for black/white", () => {
    expect(typeof isWCAGAACompliant("#000000", "#ffffff")).toBe("boolean");
  });

  it("isWCAGAAACompliant returns boolean", () => {
    expect(typeof isWCAGAAACompliant("#000000", "#ffffff")).toBe("boolean");
  });

  it("getContrastRating returns rating string for high contrast", () => {
    expect(typeof getContrastRating(7)).toBe("string");
  });
});
