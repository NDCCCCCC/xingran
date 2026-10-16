import { describe, expect, it } from "vitest";
import { LRUCache } from "./lruCache";

describe("LRUCache", () => {
  it("get/set/has/size 基本操作", () => {
    const cache = new LRUCache<string, number>(3);
    expect(cache.size).toBe(0);
    cache.set("a", 1);
    expect(cache.get("a")).toBe(1);
    expect(cache.has("a")).toBe(true);
    expect(cache.has("b")).toBe(false);
    expect(cache.size).toBe(1);
    expect(cache.get("missing")).toBeUndefined();
  });

  it("容量超限淘汰最久未使用项", () => {
    const cache = new LRUCache<string, number>(2);
    cache.set("a", 1);
    cache.set("b", 2);
    cache.set("c", 3); // 淘汰 a
    expect(cache.has("a")).toBe(false);
    expect(cache.has("b")).toBe(true);
    expect(cache.has("c")).toBe(true);
    expect(cache.size).toBe(2);
  });

  it("get 访问会刷新新鲜度（a 被读取后 b 被淘汰）", () => {
    const cache = new LRUCache<string, number>(2);
    cache.set("a", 1);
    cache.set("b", 2);
    cache.get("a"); // a 变为最近使用
    cache.set("c", 3); // 淘汰 b 而非 a
    expect(cache.has("b")).toBe(false);
    expect(cache.get("a")).toBe(1);
  });

  it("重设已有 key 更新值且不触发淘汰", () => {
    const cache = new LRUCache<string, number>(2);
    cache.set("a", 1);
    cache.set("b", 2);
    cache.set("a", 10);
    expect(cache.size).toBe(2);
    expect(cache.get("a")).toBe(10);
    expect(cache.has("b")).toBe(true);
  });

  it("clear 清空全部", () => {
    const cache = new LRUCache<string, number>(2);
    cache.set("a", 1);
    cache.set("b", 2);
    cache.clear();
    expect(cache.size).toBe(0);
    expect(cache.has("a")).toBe(false);
  });

  it("capacity <= 0 构造时抛错", () => {
    expect(() => new LRUCache<string, number>(0)).toThrow("capacity must be > 0");
    expect(() => new LRUCache<string, number>(-1)).toThrow();
  });
});
