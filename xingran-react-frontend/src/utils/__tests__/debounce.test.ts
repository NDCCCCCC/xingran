/**
 * Phase 88 Batch170 — utils/debounce 测试
 */
import { describe, it, expect, vi } from "vitest";
import { debounce } from "../debounce";

describe("debounce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("默认 trailing → wait 之后调用最后一次", () => {
    const fn = vi.fn();
    const d = debounce(fn, 200);
    d("a");
    d("b");
    d("c");
    expect(fn).not.toHaveBeenCalled();
    vi.advanceTimersByTime(200);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("c");
  });

  it("trailing=false → wait 之后不调用", () => {
    const fn = vi.fn();
    const d = debounce(fn, 200, { trailing: false });
    d("a");
    vi.advanceTimersByTime(200);
    expect(fn).not.toHaveBeenCalled();
  });

  it("leading=true → 首次立即调用", () => {
    const fn = vi.fn();
    const d = debounce(fn, 200, { leading: true });
    d("a");
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("a");
  });

  it("leading=true + trailing=true → 首次立即 + 最后一次", () => {
    const fn = vi.fn();
    const d = debounce(fn, 200, { leading: true, trailing: true });
    d("a");
    d("b");
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("a");
    vi.advanceTimersByTime(200);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenCalledWith("b");
  });

  it("持续调用 → 持续重置 timer", () => {
    const fn = vi.fn();
    const d = debounce(fn, 200);
    d("a");
    vi.advanceTimersByTime(100);
    d("b");
    vi.advanceTimersByTime(100);
    expect(fn).not.toHaveBeenCalled();
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("传多参数", () => {
    const fn = vi.fn();
    const d = debounce(fn, 100);
    d("a", 1, true);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledWith("a", 1, true);
  });
});
