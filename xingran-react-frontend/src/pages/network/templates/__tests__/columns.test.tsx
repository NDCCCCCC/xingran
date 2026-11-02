/**
 * Phase 88 Batch413 — pages/network/templates/columns 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/components/shared/ActionButtons", () => ({
  default: () => null,
}));

describe("pages/network/templates/columns", () => {
  it("getTemplateColumns 导出", async () => {
    const mod = await import("../columns");
    expect(typeof mod.getTemplateColumns).toBe("function");
  });

  it("getTemplateColumns 返回数组", async () => {
    const { getTemplateColumns } = await import("../columns");
    const cols = getTemplateColumns({
      handlePreview: () => {},
      handleClone: () => {},
      handleDelete: () => {},
      openModal: () => {},
    });
    expect(Array.isArray(cols)).toBe(true);
  });
});
