import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { debounce } from "./debounce";

describe("debounce（leading/trailing）", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("默认 trailing：等待结束后以最后一次参数执行一次", () => {
    const fn = vi.fn();
    const debounced = debounce(fn);

    debounced("a");
    debounced("b");
    debounced("c");
    expect(fn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(300);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith("c");
  });

  it("默认 wait 300ms：299ms 未执行，300ms 执行", () => {
    const fn = vi.fn();
    const debounced = debounce(fn);

    debounced("x");
    vi.advanceTimersByTime(299);
    expect(fn).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("等待窗口过后新调用开启新周期", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100);

    debounced(1);
    vi.advanceTimersByTime(100);
    debounced(2);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(1, 1);
    expect(fn).toHaveBeenNthCalledWith(2, 2);
  });

  it("leading：首次调用立即执行", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 300, { leading: true });

    debounced("now");
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("now");
  });

  it("仅 leading（未开 trailing）：窗口内后续调用不执行", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 300, { leading: true });

    debounced("first");
    debounced("second");
    debounced("third");
    vi.advanceTimersByTime(500);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith("first");
  });

  it("leading + trailing：首尾各执行一次", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 300, { leading: true, trailing: true });

    debounced("a");
    debounced("b");
    expect(fn).toHaveBeenCalledTimes(1); // leading

    vi.advanceTimersByTime(300);
    expect(fn).toHaveBeenCalledTimes(2); // trailing
    expect(fn).toHaveBeenLastCalledWith("b");
  });

  it("trailing: false：不执行尾部调用", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 300, { trailing: false });

    debounced("a");
    vi.advanceTimersByTime(500);
    expect(fn).not.toHaveBeenCalled();
  });
});
