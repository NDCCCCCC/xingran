/**
 * Phase 88 Batch357 — design-system/tokens/typography 测试
 */
import { describe, it, expect } from "vitest";
import {
  fontFamily,
  fontSize,
  fontWeight,
  lineHeight,
  letterSpacing,
  textAlign,
  textTransform,
  textDecoration,
} from "../typography";

describe("design-system/tokens/typography", () => {
  describe("fontFamily", () => {
    it("含 sans/mono/serif", () => {
      expect(fontFamily.sans).toContain("PingFang SC");
      expect(fontFamily.mono).toContain("JetBrains Mono");
      expect(fontFamily.serif).toContain("Georgia");
    });

    it("sans 包含系统字体", () => {
      expect(fontFamily.sans).toContain("-apple-system");
      expect(fontFamily.sans).toContain("sans-serif");
    });
  });

  describe("fontSize", () => {
    it("含 xs/sm/base/lg/xl/2xl", () => {
      expect(fontSize.xs).toBeDefined();
      expect(fontSize.sm).toBeDefined();
      expect(fontSize.base).toBeDefined();
      expect(fontSize.lg).toBeDefined();
      expect(fontSize.xl).toBeDefined();
      expect(fontSize["2xl"]).toBeDefined();
    });

    it("size 单调递增 (xs < sm < base < lg < xl)", () => {
      expect(parseFloat(fontSize.xs)).toBeLessThan(parseFloat(fontSize.sm));
      expect(parseFloat(fontSize.sm)).toBeLessThan(parseFloat(fontSize.base));
      expect(parseFloat(fontSize.base)).toBeLessThan(parseFloat(fontSize.lg));
      expect(parseFloat(fontSize.lg)).toBeLessThan(parseFloat(fontSize.xl));
      expect(parseFloat(fontSize.xl)).toBeLessThan(parseFloat(fontSize["2xl"]));
    });
  });

  describe("fontWeight", () => {
    it("含 normal/medium/semibold/bold", () => {
      expect(fontWeight.normal).toBeDefined();
      expect(fontWeight.medium).toBeDefined();
      expect(fontWeight.semibold).toBeDefined();
      expect(fontWeight.bold).toBeDefined();
    });
  });

  describe("lineHeight", () => {
    it("含 tight/normal/relaxed/loose", () => {
      expect(lineHeight.tight).toBeDefined();
      expect(lineHeight.normal).toBeDefined();
      expect(lineHeight.relaxed).toBeDefined();
      expect(lineHeight.loose).toBeDefined();
    });
  });

  describe("letterSpacing", () => {
    it("含 tight/normal/wide/wider", () => {
      expect(letterSpacing.tight).toBeDefined();
      expect(letterSpacing.normal).toBeDefined();
      expect(letterSpacing.wide).toBeDefined();
    });
  });

  describe("textAlign", () => {
    it("含 left/center/right/justify", () => {
      expect(textAlign.left).toBe("left");
      expect(textAlign.center).toBe("center");
      expect(textAlign.right).toBe("right");
      expect(textAlign.justify).toBe("justify");
    });
  });

  describe("textTransform", () => {
    it("含 none/uppercase/lowercase/capitalize", () => {
      expect(textTransform.none).toBe("none");
      expect(textTransform.uppercase).toBe("uppercase");
      expect(textTransform.lowercase).toBe("lowercase");
      expect(textTransform.capitalize).toBe("capitalize");
    });
  });

  describe("textDecoration", () => {
    it("含 none/underline/line-through", () => {
      expect(textDecoration.none).toBe("none");
      expect(textDecoration.underline).toBe("underline");
      expect(textDecoration["line-through"]).toBe("line-through");
    });
  });
});
