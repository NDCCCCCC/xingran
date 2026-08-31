/**
 * Phase 88 Batch211 — services/cache/TTLMenuCache 测试
 */
import { describe, it, expect, beforeEach, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { TTLMenuCache, getMenuCache, resetMenuCache } from "../TTLMenuCache";

describe("services/cache/TTLMenuCache", () => {
  beforeEach(() => {
    resetMenuCache();
  });

  it("初始 null", () => {
    const c = new TTLMenuCache();
    expect(c.getMenus()).toBeNull();
    expect(c.getAllMenus()).toBeNull();
    expect(c.getPermissions()).toBeNull();
    expect(c.isValid()).toBe(false);
    expect(c.getRemainingTime()).toBe(0);
  });

  it("setMenus + getMenus", () => {
    const c = new TTLMenuCache();
    const menus = [{ id: "1", menuName: "M1" } as any];
    const all = [{ id: "1" } as any, { id: "2" } as any];
    const perms = ["user:add"];
    c.setMenus(menus, all, perms);
    expect(c.getMenus()).toEqual(menus);
    expect(c.getAllMenus()).toEqual(all);
    expect(c.getPermissions()).toEqual(perms);
    expect(c.isValid()).toBe(true);
  });

  it("setMenus 自定义 ttl", () => {
    const c = new TTLMenuCache();
    c.setMenus([], [], [], 60_000);
    expect(c.getRemainingTime()).toBeGreaterThan(50);
    expect(c.getRemainingTime()).toBeLessThanOrEqual(60);
  });

  it("clear 清空", () => {
    const c = new TTLMenuCache();
    c.setMenus([{ id: "1" } as any], [], []);
    c.clear();
    expect(c.getMenus()).toBeNull();
    expect(c.isValid()).toBe(false);
  });

  it("过期 → null", () => {
    vi.useFakeTimers();
    const c = new TTLMenuCache(1000);
    c.setMenus([{ id: "1" } as any], [], []);
    expect(c.getMenus()).not.toBeNull();
    vi.advanceTimersByTime(2000);
    expect(c.getMenus()).toBeNull();
    expect(c.isValid()).toBe(false);
    vi.useRealTimers();
  });

  it("getMenuCache 单例", () => {
    const a = getMenuCache();
    const b = getMenuCache();
    expect(a).toBe(b);
  });

  it("resetMenuCache 后是新实例", () => {
    const a = getMenuCache();
    resetMenuCache();
    const b = getMenuCache();
    expect(a).not.toBe(b);
  });

  it("setMenus 调用时不传 ttl → 用构造 ttl", () => {
    const c = new TTLMenuCache(10_000);
    c.setMenus([], [], []);
    expect(c.getRemainingTime()).toBeGreaterThan(5);
  });
});
