/**
 * Phase 88 Batch389 — utils/dualLevelCache 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// Mock setInterval/clearInterval before importing the module
const timers: any[] = [];
vi.stubGlobal("setInterval", vi.fn((fn: any, _ms: number) => {
  const id = timers.length;
  timers.push(fn);
  return id;
}));
vi.stubGlobal("clearInterval", vi.fn(() => {}));

// Mock localStorage
const storage: Record<string, string> = {};
vi.stubGlobal("localStorage", {
  getItem: vi.fn((key: string) => storage[key] ?? null),
  setItem: vi.fn((key: string, val: string) => { storage[key] = val; }),
  removeItem: vi.fn((key: string) => { delete storage[key]; }),
  clear: vi.fn(() => { Object.keys(storage).forEach(k => delete storage[k]); }),
});

describe("utils/dualLevelCache", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    timers.length = 0;
    Object.keys(storage).forEach(k => delete storage[k]);
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("导入 DualLevelCache / getDualLevelCache / clearDualLevelCache", async () => {
    const mod = await import("../dualLevelCache");
    expect(typeof mod.DualLevelCache).toBe("function");
    expect(typeof mod.getDualLevelCache).toBe("function");
    expect(typeof mod.clearDualLevelCache).toBe("function");
  });

  it("generateKey 生成稳定 key", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    const key1 = cache.generateKey({ a: 1, b: 2 });
    const key2 = cache.generateKey({ b: 2, a: 1 });
    expect(key1).toBe(key2);
    expect(typeof key1).toBe("string");
  });

  it("generateKey 不同参数生成不同 key", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    const key1 = cache.generateKey({ a: 1 });
    const key2 = cache.generateKey({ a: 2 });
    expect(key1).not.toBe(key2);
  });

  it("set/get 基本操作", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    cache.set("k1", { name: "test" });
    expect(cache.get("k1")).toEqual({ name: "test" });
  });

  it("get 不存在的 key 返回 null", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    expect(cache.get("nonexistent")).toBeNull();
  });

  it("has 返回 boolean", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    expect(typeof cache.has("k1")).toBe("boolean");
    cache.set("k1", { v: 1 });
    expect(cache.has("k1")).toBe(true);
  });

  it("delete 移除缓存", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    cache.set("k1", { v: 1 });
    cache.delete("k1");
    expect(cache.get("k1")).toBeNull();
  });

  it("clear 不抛错", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    cache.set("k1", { v: 1 });
    expect(() => cache.clear()).not.toThrow();
  });

  it("getStats 返回统计信息结构", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    const stats = cache.getStats();
    expect(typeof stats.memoryHits).toBe("number");
    expect(typeof stats.storageHits).toBe("number");
    expect(typeof stats.misses).toBe("number");
    expect(typeof stats.totalHits).toBe("number");
    expect(typeof stats.totalRequests).toBe("number");
    expect(typeof stats.hitRate).toBe("string");
    expect(typeof stats.memorySize).toBe("number");
    expect(typeof stats.storageSize).toBe("number");
  });

  it("resetStats 重置统计", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    cache.set("k1", { v: 1 });
    cache.get("k1");
    cache.resetStats();
    const stats = cache.getStats();
    expect(stats.memoryHits).toBe(0);
    expect(stats.misses).toBe(0);
  });

  it("destroy 不抛错", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache();
    expect(() => cache.destroy()).not.toThrow();
  });

  it("getDualLevelCache 返回实例", async () => {
    const { getDualLevelCache, clearDualLevelCache } = await import("../dualLevelCache");
    clearDualLevelCache();
    const cache = getDualLevelCache();
    expect(cache).toBeDefined();
    expect(typeof cache.get).toBe("function");
  });

  it("clearDualLevelCache 不抛错", async () => {
    const { clearDualLevelCache } = await import("../dualLevelCache");
    expect(() => clearDualLevelCache()).not.toThrow();
  });

  it("构造函数接受自定义 config", async () => {
    const { DualLevelCache } = await import("../dualLevelCache");
    const cache = new DualLevelCache({ memoryTTL: 60000, storageTTL: 86400000 });
    expect(() => cache.set("k", { v: 1 })).not.toThrow();
    expect(cache.get("k")).toEqual({ v: 1 });
  });
});
