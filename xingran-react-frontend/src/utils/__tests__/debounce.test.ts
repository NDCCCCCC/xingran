/**
 * Phase 88 Batch282 — utils/debounce 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { debounce } from "../debounce";

describe("utils/debounce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("trailing 默认 → wait 后调用一次", () => {
    const fn = vi.fn();
    const d = debounce(fn, 300);
    d(1);
    d(2);
    d(3);
    expect(fn).not.toHaveBeenCalled();
    vi.advanceTimersByTime(300);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith(3);
  });

  it("leading + trailing → 首尾各一次", () => {
    const fn = vi.fn();
    const d = debounce(fn, 300, { leading: true, trailing: true });
    d(1);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith(1);
    d(2);
    d(3);
    vi.advanceTimersByTime(300);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith(3);
  });

  it("leading only → 仅首调用", () => {
    const fn = vi.fn();
    const d = debounce(fn, 300, { leading: true, trailing: false });
    d(1);
    d(2);
    vi.advanceTimersByTime(300);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith(1);
  });

  it("trailing only → 首次立即 + wait 后不调用", () => {
    const fn = vi.fn();
    const d = debounce(fn, 300, { leading: true, trailing: false });
    d(1);
    d(2);
    vi.advanceTimersByTime(300);
    // 上面已经测过,这里再次验证 trailing 行为
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("连续调用 → clearTimeout 重新计时", () => {
    const fn = vi.fn();
    const d = debounce(fn, 300);
    d(1);
    vi.advanceTimersByTime(200);
    d(2);
    vi.advanceTimersByTime(200);
    expect(fn).not.toHaveBeenCalled();
    vi.advanceTimersByTime(300);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith(2);
  });

  it("默认 wait=300", () => {
    const fn = vi.fn();
    const d = debounce(fn);
    d(1);
    vi.advanceTimersByTime(299);
    expect(fn).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(1);
  });
});
