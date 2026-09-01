/**
 * Phase 88 Batch393 — constants/storage 测试
 */
import { describe, it, expect } from "vitest";

describe("constants/storage", () => {
  it("导出 storage 相关常量", async () => {
    const mod = await import("../storage");
    expect(typeof mod).toBe("object");
  });

  it("STORAGE_KEYS 是对象", async () => {
    const { STORAGE_KEYS } = await import("../storage");
    expect(typeof STORAGE_KEYS).toBe("object");
  });

  it("TABLE_STATE_PREFIX 是字符串", async () => {
    const { TABLE_STATE_PREFIX } = await import("../storage");
    expect(typeof TABLE_STATE_PREFIX).toBe("string");
  });

  it("sanitizePathForKey 是函数", async () => {
    const { sanitizePathForKey } = await import("../storage");
    expect(typeof sanitizePathForKey).toBe("function");
  });

  it("sanitizePathForKey 规范化路径", async () => {
    const { sanitizePathForKey } = await import("../storage");
    expect(sanitizePathForKey("/system/user")).toBeTruthy();
    expect(typeof sanitizePathForKey("/test")).toBe("string");
  });

  it("ZUSTAND_STORAGE_KEYS 是对象", async () => {
    const { ZUSTAND_STORAGE_KEYS } = await import("../storage");
    expect(typeof ZUSTAND_STORAGE_KEYS).toBe("object");
  });
});
