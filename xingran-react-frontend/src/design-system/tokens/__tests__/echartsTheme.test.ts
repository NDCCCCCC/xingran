/**
 * Phase 84 84-03b — Design-system echartsTheme 静态断言(D-12)
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

describe("design-system tokens echartsTheme (D-12)", () => {
  it("brandSeriesColors is non-empty array", () => {
    expect(Array.isArray(brandSeriesColors)).toBe(true);
    expect(brandSeriesColors.length).toBeGreaterThan(0);
    for (const c of brandSeriesColors) {
      expect(c).toMatch(/^#[0-9a-fA-F]{6}$/);
    }
  });

  it("brandHeatRamp has heatmap color stops", () => {
    expect(Array.isArray(brandHeatRamp)).toBe(true);
    expect(brandHeatRamp.length).toBeGreaterThan(0);
  });

  it("brandAxisLine is CSS color or var", () => {
    expect(brandAxisLine).toBeTruthy();
    expect(brandAxisLine.length).toBeGreaterThan(0);
  });

  it("brandSplitLine is CSS color or var", () => {
    expect(brandSplitLine).toBeTruthy();
  });

  it("brandTextStyle is defined", () => {
    expect(brandTextStyle).toBeTruthy();
  });

  it("brandAreaFade has rgba format", () => {
    expect(brandAreaFade).toMatch(/^rgba?\(/);
  });

  it("brandHeatZero is defined", () => {
    expect(brandHeatZero).toBeTruthy();
  });
});
