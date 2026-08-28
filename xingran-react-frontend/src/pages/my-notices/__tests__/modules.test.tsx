/**
 * Phase 87 — my-notices 模块导入断言
 */
import { describe, it, expect } from "vitest";

describe("my-notices page modules", () => {
  it("index imports", { timeout: 30000 }, async () => {
    const m = await import("../index");
    expect(m.default).toBeDefined();
  });
  it("detail imports", { timeout: 30000 }, async () => {
    const m = await import("../detail");
    expect(m.default).toBeDefined();
  });
});
