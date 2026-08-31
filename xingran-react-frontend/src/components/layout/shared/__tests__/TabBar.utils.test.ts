/**
 * Phase 88 Batch287 — components/layout/shared/TabBar.utils 测试
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { checkScrollState, scrollContainer, setupDelayedChecks } from "../TabBar.utils";

describe("layout/shared/TabBar.utils", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("checkScrollState null → 默认", () => {
    const s = checkScrollState(null);
    expect(s.canScrollLeft).toBe(false);
    expect(s.canScrollRight).toBe(false);
    expect(s.scrollLeft).toBe(0);
  });

  it("checkScrollState scrollLeft > 0", () => {
    const el = { scrollLeft: 100, scrollWidth: 1000, clientWidth: 500 } as HTMLElement;
    const s = checkScrollState(el);
    expect(s.canScrollLeft).toBe(true);
  });

  it("checkScrollState 可向右滚动", () => {
    const el = { scrollLeft: 0, scrollWidth: 1000, clientWidth: 500 } as HTMLElement;
    const s = checkScrollState(el);
    expect(s.canScrollRight).toBe(true);
  });

  it("checkScrollState 不可向右滚动", () => {
    const el = { scrollLeft: 500, scrollWidth: 1000, clientWidth: 500 } as HTMLElement;
    const s = checkScrollState(el);
    expect(s.canScrollRight).toBe(false);
  });

  it("scrollContainer null → 不报错", () => {
    expect(() => scrollContainer(null, "left", 100)).not.toThrow();
  });

  it("scrollContainer left", () => {
    const scrollToMock = vi.fn();
    const el = { scrollLeft: 200, scrollTo: scrollToMock } as any;
    scrollContainer(el, "left", 50);
    expect(scrollToMock).toHaveBeenCalledWith({ left: 150, behavior: "smooth" });
  });

  it("scrollContainer right", () => {
    const scrollToMock = vi.fn();
    const el = { scrollLeft: 200, scrollTo: scrollToMock } as any;
    scrollContainer(el, "right", 50);
    expect(scrollToMock).toHaveBeenCalledWith({ left: 250, behavior: "smooth" });
  });

  it("setupDelayedChecks 多次调用 callback", () => {
    const cb = vi.fn();
    const timers = setupDelayedChecks(cb, [0, 100, 300]);
    expect(timers.length).toBe(3);
    vi.advanceTimersByTime(50);
    expect(cb).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(60);
    expect(cb).toHaveBeenCalledTimes(2);
    vi.advanceTimersByTime(200);
    expect(cb).toHaveBeenCalledTimes(3);
  });
});
