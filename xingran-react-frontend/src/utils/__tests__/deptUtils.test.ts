/**
 * Phase 88 Batch391 — utils/deptUtils 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("utils/deptUtils", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("deptUtils 有导出", async () => {
    const mod = await import("../deptUtils");
    expect(typeof mod).toBe("object");
  });

  it("getDeptNodeId 存在", async () => {
    const { getDeptNodeId } = await import("../deptUtils");
    expect(typeof getDeptNodeId).toBe("function");
  });

  it("findDeptNode 对空数组返回 null", async () => {
    const { findDeptNode } = await import("../deptUtils");
    const result = findDeptNode([], "1");
    expect(result).toBeNull();
  });

  it("collectDescendantIds 存在", async () => {
    const { collectDescendantIds } = await import("../deptUtils");
    expect(typeof collectDescendantIds).toBe("function");
  });

  it("collectDescendantIds 对空数组返回空", async () => {
    const { collectDescendantIds } = await import("../deptUtils");
    expect(collectDescendantIds([], "1")).toEqual([]);
  });

  it("toFullPathTree 存在", async () => {
    const { toFullPathTree } = await import("../deptUtils");
    expect(typeof toFullPathTree).toBe("function");
  });

  it("toShortNameDataNode 存在", async () => {
    const { toShortNameDataNode } = await import("../deptUtils");
    expect(typeof toShortNameDataNode).toBe("function");
  });

  it("dedupTreeByKey 存在", async () => {
    const { dedupTreeByKey } = await import("../deptUtils");
    expect(typeof dedupTreeByKey).toBe("function");
  });
});
