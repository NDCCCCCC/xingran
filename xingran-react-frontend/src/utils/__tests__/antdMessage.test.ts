/**
 * Phase 88 Batch402 — utils/antdMessage 测试
 */
import { describe, it, expect } from "vitest";

describe("utils/antdMessage", () => {
  it("getAppMessage 导出", async () => {
    const mod = await import("../antdMessage");
    expect(typeof mod.getAppMessage).toBe("function");
  });
});
