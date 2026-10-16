/**
 * Phase 88 Batch390 — utils/geocodingCache 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// Mock localStorage
const storage: Record<string, string> = {};
vi.stubGlobal("localStorage", {
  getItem: vi.fn((key: string) => storage[key] ?? null),
  setItem: vi.fn((key: string, val: string) => {
    storage[key] = val;
  }),
  removeItem: vi.fn((key: string) => {
    delete storage[key];
  }),
  clear: vi.fn(() => {
    Object.keys(storage).forEach((k) => delete storage[k]);
  }),
});

describe("utils/geocodingCache", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.keys(storage).forEach((k) => delete storage[k]);
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("导入 GeocodingCache/getGeocodingCache/clearGeocodingCache", async () => {
    const mod = await import("../geocodingCache");
    expect(typeof mod.GeocodingCache).toBe("function");
    expect(typeof mod.getGeocodingCache).toBe("function");
    expect(typeof mod.clearGeocodingCache).toBe("function");
  });

  it("getGeocodingCache 返回实例", async () => {
    const { getGeocodingCache, clearGeocodingCache } = await import("../geocodingCache");
    clearGeocodingCache();
    const cache = getGeocodingCache();
    expect(cache).toBeDefined();
    expect(typeof cache.get).toBe("function");
  });

  it("set/get 基本操作", async () => {
    const { GeocodingCache } = await import("../geocodingCache");
    const cache = new GeocodingCache();
    cache.set("k1", { lat: 39.9, lng: 116.4 });
    expect(cache.get("k1")).toEqual({ lat: 39.9, lng: 116.4 });
  });

  it("get 不存在 key 返回 null", async () => {
    const { GeocodingCache } = await import("../geocodingCache");
    const cache = new GeocodingCache();
    expect(cache.get("nonexistent")).toBeNull();
  });

  it("has 返回 boolean", async () => {
    const { GeocodingCache } = await import("../geocodingCache");
    const cache = new GeocodingCache();
    expect(typeof cache.has("k")).toBe("boolean");
    cache.set("k", { lat: 1, lng: 2 });
    expect(cache.has("k")).toBe(true);
  });

  it("delete 移除条目", async () => {
    const { GeocodingCache } = await import("../geocodingCache");
    const cache = new GeocodingCache();
    cache.set("k", { lat: 1, lng: 2 });
    cache.delete("k");
    expect(cache.has("k")).toBe(false);
  });

  it("clear 不抛错", async () => {
    const { GeocodingCache } = await import("../geocodingCache");
    const cache = new GeocodingCache();
    cache.set("k", { lat: 1, lng: 2 });
    expect(() => cache.clear()).not.toThrow();
  });

  it("clearGeocodingCache 不抛错", async () => {
    const { clearGeocodingCache } = await import("../geocodingCache");
    expect(() => clearGeocodingCache()).not.toThrow();
  });
});
