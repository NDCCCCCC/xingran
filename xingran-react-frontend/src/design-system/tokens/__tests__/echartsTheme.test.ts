/**
 * Phase 88 Batch315 — design-system/tokens/echartsTheme 测试
 */
import { describe, it, expect } from "vitest";
import {
  brandSeriesColors,
  brandHeatRamp,
  brandAxisLine,
  brandSplitLine,
  brandTextStyle,
  brandAreaFade,
  brandHeatZero,
} from "../echartsTheme";

describe("design-system/tokens/echartsTheme", () => {
  it("brandSeriesColors 5 项", () => {
    expect(brandSeriesColors.length).toBe(5);
  });

  it("brandSeriesColors 第一项 brand primary green", () => {
    expect(brandSeriesColors[0]).toBe("#156031");
  });

  it("brandSeriesColors 含铜金", () => {
    expect(brandSeriesColors).toContain("#C09058");
  });

  it("brandHeatRamp 5 项", () => {
    expect(brandHeatRamp.length).toBe(5);
  });

  it("brandHeatRamp 低 → 高", () => {
    expect(brandHeatRamp[0]).toBe("#E9EFEB");
    expect(brandHeatRamp[2]).toBe("#156031");
    expect(brandHeatRamp[4]).toBe("#B88850");
  });

  it("brandAxisLine = cream.border", () => {
    expect(brandAxisLine).toBe("#DBD7CE");
  });

  it("brandSplitLine = green[50]", () => {
    expect(brandSplitLine).toBe("#E9EFEB");
  });

  it("brandTextStyle cream.mutedStrong", () => {
    expect(brandTextStyle).toBe("#64645C");
  });

  it("brandAreaFade rgba", () => {
    expect(brandAreaFade).toMatch(/^rgba/);
    expect(brandAreaFade).toContain("0.1");
  });

  it("brandHeatZero = cream.borderStrong", () => {
    expect(brandHeatZero).toBe("#C2BDB2");
  });

  it("导出项都是 readonly string", () => {
    for (const c of brandSeriesColors) expect(typeof c).toBe("string");
    for (const c of brandHeatRamp) expect(typeof c).toBe("string");
  });
});
