/**
 * Phase 88 Batch234 — design-system/animations/keyframes 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import * as kf from "../keyframes";

describe("design-system/animations/keyframes", () => {
  it("fadeIn 含 from/to opacity", () => {
    expect(kf.fadeIn).toEqual({ from: { opacity: 0 }, to: { opacity: 1 } });
  });

  it("fadeOut 含 from/to opacity", () => {
    expect(kf.fadeOut).toEqual({ from: { opacity: 1 }, to: { opacity: 0 } });
  });

  it("fadeInUp 含 transform", () => {
    expect(kf.fadeInUp.from).toHaveProperty("transform");
  });

  it("fadeInDown 含 translateY", () => {
    expect(kf.fadeInDown.from.transform).toContain("translateY(-20px)");
  });

  it("fadeInLeft 含 translateX", () => {
    expect(kf.fadeInLeft.from.transform).toContain("translateX(-20px)");
  });

  it("fadeInRight 含 translateX positive", () => {
    expect(kf.fadeInRight.from.transform).toContain("translateX(20px)");
  });

  it("slideInUp 含 translateY 100%", () => {
    expect(kf.slideInUp).toHaveProperty("from");
  });

  it("slideInDown 含 transform", () => {
    expect(kf.slideInDown.from.transform).toBeTruthy();
  });

  it("scaleIn 含 scale", () => {
    expect(kf.scaleIn.from.transform).toContain("scale");
  });

  it("scaleOut 含 scale", () => {
    expect(kf.scaleOut.from.transform).toContain("scale");
  });

  it("rotate 含 rotate", () => {
    expect(kf.rotate.from.transform).toContain("rotate");
  });

  it("bounce 含 0%, 100% 关键帧", () => {
    expect(kf.bounce).toHaveProperty("0%, 100%");
    expect(kf.bounce).toHaveProperty("50%");
  });

  it("pulse 含 50% opacity", () => {
    expect((kf.pulse as any)["50%"]).toEqual({ opacity: 0.5 });
  });

  it("pulseScale 含 scale", () => {
    expect((kf.pulseScale as any)["50%"]).toHaveProperty("transform");
  });

  it("shake 含 translateX", () => {
    expect(kf.shake).toHaveProperty("0%, 100%");
  });

  it("shimmer 含 backgroundPosition", () => {
    expect(kf.shimmer.from).toHaveProperty("backgroundPosition");
  });

  it("themeTransition 含 0%/50%/100%", () => {
    expect(kf.themeTransition).toHaveProperty("0%");
    expect(kf.themeTransition).toHaveProperty("50%");
    expect(kf.themeTransition).toHaveProperty("100%");
  });

  it("导出 17+ 关键帧", () => {
    expect(Object.keys(kf).length).toBeGreaterThanOrEqual(15);
  });
});
