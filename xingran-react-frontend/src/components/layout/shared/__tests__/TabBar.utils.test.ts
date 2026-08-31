/**
 * Phase 88 Batch361 — components/layout/shared/TabBar.utils 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { checkScrollState, scrollContainer, setupDelayedChecks } from "../TabBar.utils";

describe("components/layout/shared/TabBar.utils", () => {
  describe("checkScrollState", () => {
    it("null container → 默认值", () => {
      const result = checkScrollState(null);
      expect(result.canScrollLeft).toBe(false);
      expect(result.canScrollRight).toBe(false);
      expect(result.scrollLeft).toBe(0);
    });

    it("scrollLeft=0 → canScrollLeft=false", () => {
      const container = {
        scrollLeft: 0,
        scrollWidth: 1000,
        clientWidth: 500,
      } as HTMLElement;
      const result = checkScrollState(container);
      expect(result.canScrollLeft).toBe(false);
      expect(result.canScrollRight).toBe(true);
    });

    it("scrollLeft>0 → canScrollLeft=true", () => {
      const container = {
        scrollLeft: 100,
        scrollWidth: 1000,
        clientWidth: 500,
      } as HTMLElement;
      expect(checkScrollState(container).canScrollLeft).toBe(true);
    });

    it("完全滚动到底 → canScrollRight=false", () => {
      const container = {
        scrollLeft: 500,
        scrollWidth: 1000,
        clientWidth: 500,
      } as HTMLElement;
      expect(checkScrollState(container).canScrollRight).toBe(false);
    });

    it("scrollWidth <= clientWidth → canScrollRight=false", () => {
      const container = {
        scrollLeft: 0,
        scrollWidth: 500,
        clientWidth: 500,
      } as HTMLElement;
      expect(checkScrollState(container).canScrollRight).toBe(false);
    });
  });

  describe("scrollContainer", () => {
    it("null container → 不调用 scrollTo", () => {
      expect(() => scrollContainer(null, "left", 100)).not.toThrow();
    });

    it("left 方向 → 减少 scrollLeft", () => {
      const container = {
        scrollLeft: 200,
        scrollTo: vi.fn(),
      } as unknown as HTMLElement;
      scrollContainer(container, "left", 100);
      expect(container.scrollTo).toHaveBeenCalledWith({
        left: 100,
        behavior: "smooth",
      });
    });

    it("right 方向 → 增加 scrollLeft", () => {
      const container = {
        scrollLeft: 200,
        scrollTo: vi.fn(),
      } as unknown as HTMLElement;
      scrollContainer(container, "right", 50);
      expect(container.scrollTo).toHaveBeenCalledWith({
        left: 250,
        behavior: "smooth",
      });
    });
  });

  describe("setupDelayedChecks", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });
    afterEach(() => {
      vi.useRealTimers();
    });

    it("返回 N 个 timer", () => {
      const callback = vi.fn();
      const timers = setupDelayedChecks(callback, [100, 200, 300]);
      expect(timers.length).toBe(3);
    });

    it("callback 在每个 delay 都触发", () => {
      const callback = vi.fn();
      setupDelayedChecks(callback, [100, 200]);
      vi.advanceTimersByTime(250);
      expect(callback).toHaveBeenCalledTimes(2);
    });

    it("empty delays → empty array", () => {
      const timers = setupDelayedChecks(() => {}, []);
      expect(timers.length).toBe(0);
    });
  });
});
