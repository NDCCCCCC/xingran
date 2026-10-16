/**
 * Phase 88 Batch359 — design-system/animations/transitions 测试
 */
import { describe, it, expect } from "vitest";
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
  describe("easings", () => {
    it("含 linear/ease/easeInOut", () => {
      expect(easings.linear).toBe("linear");
      expect(easings.ease).toBe("ease");
      expect(easings.easeInOut).toContain("cubic-bezier");
    });

    it("含弹性缓动 bouncy", () => {
      expect(easings.bouncy).toContain("cubic-bezier");
    });
  });

  describe("durations", () => {
    it("含 instant/fast/base/normal/slow/slower", () => {
      expect(durations.instant).toBe("0ms");
      expect(durations.fast).toBe("150ms");
      expect(durations.base).toBe("200ms");
    });

    it("duration 单调递增", () => {
      const order = ["instant", "fast", "base", "normal", "slow", "slower"];
      for (let i = 1; i < order.length; i++) {
        const prev = parseInt(durations[order[i - 1] as keyof typeof durations]);
        const curr = parseInt(durations[order[i] as keyof typeof durations]);
        expect(curr).toBeGreaterThan(prev);
      }
    });
  });

  describe("delays", () => {
    it("含 none/short/medium/long", () => {
      expect(delays.none).toBe("0ms");
      expect(delays.short).toBe("100ms");
      expect(delays.medium).toBe("200ms");
      expect(delays.long).toBe("300ms");
    });
  });

  describe("transitions", () => {
    it("fast/base/slow 等预设", () => {
      expect(transitions.fast).toBeDefined();
      expect(transitions.base).toBeDefined();
      expect(transitions.slow).toBeDefined();
    });

    it("transitions 含 duration + easing", () => {
      expect(transitions.fast).toContain("ms");
      expect(transitions.fast).toMatch(/cubic-bezier|ease|linear/);
    });
  });

  describe("propertyTransitions", () => {
    it("含 colors/shadow/transform/opacity/all/common", () => {
      expect(propertyTransitions.colors).toBeDefined();
      expect(propertyTransitions.shadow).toBeDefined();
      expect(propertyTransitions.transform).toBeDefined();
      expect(propertyTransitions.opacity).toBeDefined();
      expect(propertyTransitions.all).toBeDefined();
      expect(propertyTransitions.common).toBeDefined();
    });
  });

  describe("staggerDelays", () => {
    it("含 fast/base/slow", () => {
      expect(staggerDelays.fast).toBeDefined();
      expect(staggerDelays.base).toBeDefined();
      expect(staggerDelays.slow).toBeDefined();
    });
  });

  describe("getStaggerDelay", () => {
    it("index 0 → 0", () => {
      expect(getStaggerDelay(0)).toBe(0);
    });

    it("index 5 + base delay → 5 * base", () => {
      const r = getStaggerDelay(5);
      expect(r).toBe(parseInt(staggerDelays.base) * 5);
    });

    it("自定义 delay (fast)", () => {
      const r = getStaggerDelay(3, "fast");
      expect(r).toBe(parseInt(staggerDelays.fast) * 3);
    });
  });

  describe("transitionUtils.create", () => {
    it("单 property", () => {
      const result = transitionUtils.create("opacity", "fast", "easeOut");
      expect(result).toBe("opacity 150ms cubic-bezier(0, 0, 0.2, 1)");
    });

    it("多 properties", () => {
      const result = transitionUtils.create(["opacity", "color"], "base", "easeInOut");
      expect(result).toBe("opacity, color 200ms cubic-bezier(0.4, 0, 0.2, 1)");
    });

    it("默认 duration/easing", () => {
      const result = transitionUtils.create("opacity");
      expect(result).toContain("200ms"); // base
      expect(result).toContain("cubic-bezier"); // easeInOut
    });
  });

  describe("transitionUtils.stagger", () => {
    it("单 property + index", () => {
      const result = transitionUtils.stagger("opacity", 2, "base", "fast", "easeOut");
      expect(result).toContain("opacity");
      expect(result).toContain("150ms");
      // delay should be 2 * 100 = 200ms
      expect(result).toContain("200ms");
    });

    it("默认参数", () => {
      const result = transitionUtils.stagger("color", 0);
      expect(result).toContain("color");
      expect(result).toContain("0ms");
    });
  });
});
