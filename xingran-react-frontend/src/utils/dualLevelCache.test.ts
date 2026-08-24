import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DualLevelCache, getDualLevelCache, clearDualLevelCache } from "./dualLevelCache";

const PREFIX = "test_dlc_";

function createCache() {
  return new DualLevelCache<string>({
    memoryTTL: 1000,
    storageTTL: 10_000,
    storagePrefix: PREFIX,
  });
}

describe("DualLevelCache", () => {
  let cache: DualLevelCache<string>;

  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.spyOn(console, "table").mockImplementation(() => {});
    cache = createCache();
  });

  afterEach(() => {
    cache.destroy();
    vi.useRealTimers();
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("L1 内存命中：set 后 get 直接返回", () => {
    cache.set("k1", "v1");
    expect(cache.get("k1")).toBe("v1");
    const stats = cache.getStats();
    expect(stats.memoryHits).toBe(1);
    expect(stats.storageHits).toBe(0);
  });

  it("L2 localStorage 命中并回填内存（跨实例）", () => {
    cache.set("k1", "stored");
    // 第二个实例共享同前缀 localStorage，但内存为空 → 只能 L2 命中
    const cache2 = createCache();
    expect(cache2.get("k1")).toBe("stored");
    expect(cache2.getStats().storageHits).toBe(1);
    // 回填后再次读取为内存命中
    expect(cache2.get("k1")).toBe("stored");
    expect(cache2.getStats().memoryHits).toBe(1);
    cache2.destroy();
  });

  it("两级都未命中返回 null 并计入 miss", () => {
    expect(cache.get("nope")).toBeNull();
    expect(cache.getStats().misses).toBe(1);
    // totalRequests>0 时 hitRate 走百分比格式
    expect(cache.getStats().hitRate).toBe("0.00%");
  });

  it("内存 TTL 过期后回落 localStorage（L2 命中）", () => {
    cache.set("k1", "v1");
    vi.advanceTimersByTime(2000); // > memoryTTL(1000) 且 < storageTTL(10000)
    expect(cache.get("k1")).toBe("v1");
    expect(cache.getStats().storageHits).toBe(1);
  });

  it("localStorage TTL 过期后返回 null 并删除条目", () => {
    cache.set("k1", "v1");
    vi.advanceTimersByTime(11_000); // > storageTTL
    expect(cache.get("k1")).toBeNull();
    expect(localStorage.getItem(PREFIX + "k1")).toBeNull();
  });

  it("persistToStorage=false 只写内存不落 localStorage", () => {
    cache.set("k1", "memory-only", false);
    expect(localStorage.getItem(PREFIX + "k1")).toBeNull();
    expect(cache.get("k1")).toBe("memory-only");

    const cache2 = createCache();
    expect(cache2.get("k1")).toBeNull();
    cache2.destroy();
  });

  it("delete 同时清除两级缓存", () => {
    cache.set("k1", "v1");
    cache.delete("k1");
    expect(cache.has("k1")).toBe(false);
    expect(localStorage.getItem(PREFIX + "k1")).toBeNull();
  });

  it("clear 只清本前缀条目（不误伤其他 key）", () => {
    cache.set("k1", "v1");
    localStorage.setItem("other_app_key", "keep-me");
    cache.clear();
    expect(cache.has("k1")).toBe(false);
    expect(localStorage.getItem("other_app_key")).toBe("keep-me");
  });

  it("generateKey 参数排序无关且确定", () => {
    const a = cache.generateKey({ a: 1, b: "x" });
    const b = cache.generateKey({ b: "x", a: 1 });
    expect(a).toBe(b);
    expect(cache.generateKey({ a: 2, b: "x" })).not.toBe(a);
  });

  it("损坏的 localStorage JSON 返回 null（序列化异常回退）", () => {
    localStorage.setItem(PREFIX + "bad", "{broken-json");
    expect(cache.get("bad")).toBeNull();
    expect(cache.getStats().misses).toBe(1);
  });

  it("cleanup 清理过期的 localStorage 条目", () => {
    cache.set("k1", "v1");
    vi.advanceTimersByTime(11_000);
    expect(localStorage.getItem(PREFIX + "k1")).not.toBeNull(); // 惰性未删
    cache.cleanup();
    expect(localStorage.getItem(PREFIX + "k1")).toBeNull();
  });

  it("统计与重置：hitRate 百分比 + resetStats 归零", () => {
    const tableSpy = vi.spyOn(console, "table").mockImplementation(() => {});
    cache.set("k1", "v1");
    cache.get("k1"); // hit
    cache.get("miss"); // miss
    const stats = cache.getStats();
    expect(stats.totalRequests).toBe(2);
    expect(stats.totalHits).toBe(1);
    expect(stats.hitRate).toBe("50.00%");
    expect(stats.storageSize).toBe(1);
    expect(stats.memorySize).toBe(1);

    cache.resetStats();
    expect(cache.getStats().totalRequests).toBe(0);

    expect(() => cache.printStats()).not.toThrow();
    expect(tableSpy).toHaveBeenCalled();
  });

  it("单例：getDualLevelCache 复用实例，clearDualLevelCache 销毁", () => {
    clearDualLevelCache();
    const a = getDualLevelCache<string>();
    const b = getDualLevelCache<string>();
    expect(a).toBe(b);
    clearDualLevelCache();
    expect(getDualLevelCache<string>()).not.toBe(a);
    // 清理单例留下的定时器与数据
    clearDualLevelCache();
  });
});
