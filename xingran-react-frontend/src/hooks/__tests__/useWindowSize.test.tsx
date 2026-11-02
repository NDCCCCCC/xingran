/**
 * Phase 88 Batch398 — hooks/useWindowSize 测试
 */
import { describe, it, expect } from "vitest";

describe("hooks/useWindowSize", () => {
  it("useWindowSize 导出", async () => {
    const mod = await import("../useWindowSize");
    expect(typeof mod).toBe("object");
  });

  it("useWindowSize 是函数", async () => {
    const { useWindowSize } = await import("../useWindowSize");
    expect(typeof useWindowSize).toBe("function");
  });
});
