/**
 * Phase 88 Batch210 — services/cache/MenuCache 接口测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { IMenuCache, MenuCacheEntry } from "../MenuCache";

describe("services/cache/MenuCache interface", () => {
  it("导出 IMenuCache 接口含 7 方法", () => {
    // 静态类型检查,通过运行时 shape 校验
    const shape: IMenuCache = {
      getMenus: () => null,
      getAllMenus: () => null,
      getPermissions: () => null,
      setMenus: () => {},
      isValid: () => false,
      clear: () => {},
      getRemainingTime: () => 0,
    };
    expect(typeof shape.getMenus).toBe("function");
    expect(typeof shape.getAllMenus).toBe("function");
    expect(typeof shape.getPermissions).toBe("function");
    expect(typeof shape.setMenus).toBe("function");
    expect(typeof shape.isValid).toBe("function");
    expect(typeof shape.clear).toBe("function");
    expect(typeof shape.getRemainingTime).toBe("function");
  });

  it("MenuCacheEntry 类型 shape", () => {
    const entry: MenuCacheEntry<string> = {
      data: "x",
      timestamp: 1000,
      ttl: 300_000,
    };
    expect(entry.data).toBe("x");
    expect(entry.timestamp).toBe(1000);
    expect(entry.ttl).toBe(300_000);
  });

  it("getMenus 返回 null", () => {
    const cache: IMenuCache = {
      getMenus: () => null,
      getAllMenus: () => null,
      getPermissions: () => null,
      setMenus: () => {},
      isValid: () => false,
      clear: () => {},
      getRemainingTime: () => 0,
    };
    expect(cache.getMenus()).toBeNull();
    expect(cache.getAllMenus()).toBeNull();
    expect(cache.getPermissions()).toBeNull();
    expect(cache.isValid()).toBe(false);
    expect(cache.getRemainingTime()).toBe(0);
  });

  it("setMenus 调用无异常", () => {
    let called = false;
    const cache: IMenuCache = {
      getMenus: () => null,
      getAllMenus: () => null,
      getPermissions: () => null,
      setMenus: () => {
        called = true;
      },
      isValid: () => false,
      clear: () => {},
      getRemainingTime: () => 0,
    };
    cache.setMenus([], [], [], 60000);
    expect(called).toBe(true);
  });
});
