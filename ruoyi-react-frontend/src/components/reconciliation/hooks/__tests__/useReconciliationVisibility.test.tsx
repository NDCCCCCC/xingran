/**
 * Phase 88 Batch363 — components/reconciliation/hooks/useReconciliationVisibility 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";

let mockPermissions: string[] = [];

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/store/menuStore", () => ({
  useMenuStore: vi.fn((selector: any) => {
    const state = { permissions: mockPermissions };
    return typeof selector === "function" ? selector(state) : state;
  }),
}));

import { useReconciliationVisibility } from "../useReconciliationVisibility";

describe("components/reconciliation/hooks/useReconciliationVisibility", () => {
  it("无 permissions → false", () => {
    mockPermissions = [];
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(false);
  });

  it("包含 asset:reconciliation:list → true", () => {
    mockPermissions = ["asset:reconciliation:list"];
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(true);
  });

  it("包含其他 perm 但不含目标 → false", () => {
    mockPermissions = ["other:perm", "user:list"];
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(false);
  });

  it("permissions 多个含目标 → true", () => {
    mockPermissions = ["a:1", "asset:reconciliation:list", "b:2"];
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(true);
  });
});
