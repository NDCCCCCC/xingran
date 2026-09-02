/**
 * Phase 88 Batch412 — pages/operations/rpa/tasks/columns 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/components/shared/ActionButtons", () => ({
  default: () => null,
}));

describe("pages/operations/rpa/tasks/columns", () => {
  it("getTaskColumns 导出", async () => {
    const mod = await import("../columns");
    expect(typeof mod.getTaskColumns).toBe("function");
  });

  it("getTaskColumns 返回数组", async () => {
    const { getTaskColumns } = await import("../columns");
    const cols = getTaskColumns({
      handleEdit: () => {},
      handleDelete: () => {},
      handleExecute: () => {},
      getSortOrder: () => null,
    });
    expect(Array.isArray(cols)).toBe(true);
  });
});