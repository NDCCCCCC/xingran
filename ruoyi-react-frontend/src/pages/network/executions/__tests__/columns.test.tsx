/**
 * Phase 88 Batch413 — pages/network/executions/columns 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/components/shared/ActionButtons", () => ({
  default: () => null,
}));

describe("pages/network/executions/columns", () => {
  it("deviceColumns 导出", async () => {
    const mod = await import("../columns");
    expect(Array.isArray(mod.deviceColumns)).toBe(true);
  });

  it("getExecutionColumns 是函数", async () => {
    const { getExecutionColumns } = await import("../columns");
    expect(typeof getExecutionColumns).toBe("function");
  });

  it("getExecutionColumns 返回数组", async () => {
    const { getExecutionColumns } = await import("../columns");
    const cols = getExecutionColumns({
      handleViewDetail: () => {},
      handleCancelExecution: async () => {},
    });
    expect(Array.isArray(cols)).toBe(true);
  });

  it("getDetailColumns 是函数", async () => {
    const { getDetailColumns } = await import("../columns");
    expect(typeof getDetailColumns).toBe("function");
  });

  it("getDetailColumns 返回数组", async () => {
    const { getDetailColumns } = await import("../columns");
    const cols = getDetailColumns({
      handleViewOutput: () => {},
    });
    expect(Array.isArray(cols)).toBe(true);
  });
});
