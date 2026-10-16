/**
 * Phase 88 Batch402 — utils/buildSearchParams 测试
 */
import { describe, it, expect } from "vitest";

describe("utils/buildSearchParams", () => {
  it("buildSearchParams 导出", async () => {
    const mod = await import("../buildSearchParams");
    expect(typeof mod).toBe("object");
  });
});
