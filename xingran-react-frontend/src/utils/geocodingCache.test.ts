import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GeocodingCache, getGeocodingCache, clearGeocodingCache } from "./geocodingCache";

const PREFIX = "test_geo_";

function createCache() {
  return new GeocodingCache<{ lng: number; lat: number }>({
    memoryTTL: 1000,
    storageTTL: 10_000,
    storagePrefix: PREFIX,
  });
}

describe("GeocodingCache", () => {
  let cache: GeocodingCache<{ lng: number; lat: number }>;

  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    vi.spyOn(console, "warn").mockImplementation(() => {});
    cache = createCache();
  });

  afterEach(() => {
    cache.destroy();
    vi.useRealTimers();
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("set/get/has 基本读写", () => {
    const value = { lng: 114.3, lat: 30.6 };
    expect(cache.has("wuhan")).toBe(false);
    cache.set("wuhan", value);
    expect(cache.has("wuhan")).toBe(true);
    expect(cache.get("wuhan")).toEqual(value);
  });

  it("未命中返回 null", () => {
    expect(cache.get("nope")).toBeNull();
    expect(cache.has("nope")).toBe(false);
  });

  it("内存过期后回落 localStorage（跨实例 L2 命中）", () => {
    cache.set("wuhan", { lng: 1, lat: 2 });
    vi.advanceTimersByTime(2000); // > memoryTTL, < storageTTL
    const cache2 = createCache();
    expect(cache2.get("wuhan")).toEqual({ lng: 1, lat: 2 });
    cache2.destroy();
  });

  it("localStorage TTL 过期后返回 null", () => {
    cache.set("wuhan", { lng: 1, lat: 2 });
    vi.advanceTimersByTime(11_000);
    expect(cache.get("wuhan")).toBeNull();
    expect(localStorage.getItem(PREFIX + "wuhan")).toBeNull();
  });

  it("generateKey 参数排序后生成确定 key", () => {
    expect(cache.generateKey({ city: "wuhan", q: "x" })).toBe(
      cache.generateKey({ q: "x", city: "wuhan" })
    );
    expect(cache.generateKey({ city: "wuhan" })).not.toBe(cache.generateKey({ city: "beijing" }));
  });

  it("getOrSet：miss 时调用 factory 并缓存，hit 时不调用（缓存穿透保护）", async () => {
    const factory = vi.fn(async () => ({ lng: 9, lat: 9 }));
    const first = await cache.getOrSet("addr-1", factory);
    expect(first).toEqual({ lng: 9, lat: 9 });
    expect(factory).toHaveBeenCalledTimes(1);

    // 第二次命中缓存，factory 不再被调用
    const second = await cache.getOrSet("addr-1", factory);
    expect(second).toEqual({ lng: 9, lat: 9 });
    expect(factory).toHaveBeenCalledTimes(1);
  });

  it("delete / clear 清除缓存", () => {
    cache.set("a", { lng: 1, lat: 1 });
    cache.set("b", { lng: 2, lat: 2 });
    localStorage.setItem("other_key", "keep");
    cache.delete("a");
    expect(cache.has("a")).toBe(false);
    expect(cache.has("b")).toBe(true);
    cache.clear();
    expect(cache.has("b")).toBe(false);
    expect(localStorage.getItem("other_key")).toBe("keep");
  });

  it("getStats 返回内存大小", () => {
    cache.set("a", { lng: 1, lat: 1 });
    expect(cache.getStats().memorySize).toBe(1);
  });

  it("单例：getGeocodingCache 复用实例，clearGeocodingCache 销毁重建", () => {
    clearGeocodingCache();
    const a = getGeocodingCache<{ lng: number; lat: number }>();
    const b = getGeocodingCache<{ lng: number; lat: number }>();
    expect(a).toBe(b);
    clearGeocodingCache();
    expect(getGeocodingCache<{ lng: number; lat: number }>()).not.toBe(a);
    clearGeocodingCache();
  });
});
