/**
 * Phase 87 — knowledge 模块导入断言
 */
import { describe, it, expect } from "vitest";

describe("knowledge page modules", () => {
  it("articles imports", { timeout: 30000 }, async () => {
    const m = await import("../articles");
    expect(m.default).toBeDefined();
  });
});
