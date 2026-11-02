/**
 * Phase 88 Batch75 — ad-domain AccountPoolTab 静态导出检查
 * (避免 render 触发 jsdom teardown ReferenceError: window is not defined)
 */
import { describe, it, expect } from "vitest";

describe("AccountPoolTab 模块导出", () => {
  it("默认导出函数组件", async () => {
    const mod = await import("../AccountPoolTab");
    expect(typeof mod.default).toBe("function");
  });
});
