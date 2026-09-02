/**
 * Phase 88 Batch412 — pages/operations/assets/columnsSchema 测试
 */
import { describe, it, expect } from "vitest";

describe("pages/operations/assets/columnsSchema", () => {
  it("导出 defaultAssetColumns", async () => {
    const mod = await import("../columnsSchema");
    expect(Array.isArray(mod.defaultAssetColumns)).toBe(true);
  });

  it("列数大于 10", async () => {
    const { defaultAssetColumns } = await import("../columnsSchema");
    expect(defaultAssetColumns.length).toBeGreaterThan(10);
  });

  it("每个列都有 key/label/visible/order 字段", async () => {
    const { defaultAssetColumns } = await import("../columnsSchema");
    defaultAssetColumns.forEach((c) => {
      expect(typeof c.key).toBe("string");
      expect(typeof c.label).toBe("string");
      expect(typeof c.visible).toBe("boolean");
      expect(typeof c.order).toBe("number");
    });
  });

  it("key 唯一", async () => {
    const { defaultAssetColumns } = await import("../columnsSchema");
    const keys = defaultAssetColumns.map((c) => c.key);
    expect(new Set(keys).size).toBe(keys.length);
  });
});