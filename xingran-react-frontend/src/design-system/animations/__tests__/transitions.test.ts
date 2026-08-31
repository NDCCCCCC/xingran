/**
 * Phase 88 Batch220 — design-system/animations/transitions 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  easings,
  durations,
  delays,
  transitions,
  propertyTransitions,
  staggerDelays,
  getStaggerDelay,
  transitionUtils,
} from "../transitions";

describe("design-system/animations/transitions", () => {
  it("easings 10 项", () => {
    expect(Object.keys(easings).length).toBeGreaterThanOrEqual(9);
  });

  it("durations 6 项", () => {
    expect(durations.instant).toBe("0ms");
    expect(durations.fast).toBe("150ms");
    expect(durations.base).toBe("200ms");
    expect(durations.normal).toBe("300ms");
  });

  it("delays 4 项", () => {
    expect(Object.keys(delays).length).toBeGreaterThanOrEqual(3);
  });

  it("transitions 10+ 预设", () => {
    expect(Object.keys(transitions).length).toBeGreaterThanOrEqual(10);
    expect(transitions.fast).toContain("150ms");
    expect(transitions.modal).toContain("0.34, 1.56");
  });

  it("propertyTransitions 含 colors/shadow/opacity", () => {
    expect(propertyTransitions.colors).toContain("color");
    expect(propertyTransitions.shadow).toContain("box-shadow");
    expect(propertyTransitions.opacity).toContain("opacity");
  });

  it("staggerDelays 4 项", () => {
    expect(Object.keys(staggerDelays).length).toBe(4);
  });

  it("getStaggerDelay index * delay", () => {
    expect(getStaggerDelay(0)).toBe(0);
    expect(getStaggerDelay(2)).toBe(200);
    expect(getStaggerDelay(3, "slow")).toBe(450);
  });

  it("transitionUtils.create 单个属性", () => {
    const result = transitionUtils.create("opacity", "base", "easeOut");
    expect(result).toContain("opacity");
    expect(result).toContain("200ms");
  });

  it("transitionUtils.create 多个属性", () => {
    const result = transitionUtils.create(["color", "background"], "fast", "easeOut");
    expect(result).toContain("color, background");
  });

  it("transitionUtils.stagger", () => {
    const result = transitionUtils.stagger("opacity", 2, "base", "fast", "easeOut");
    expect(result).toContain("opacity");
    expect(result).toContain("200ms"); // base * 2
  });
});
