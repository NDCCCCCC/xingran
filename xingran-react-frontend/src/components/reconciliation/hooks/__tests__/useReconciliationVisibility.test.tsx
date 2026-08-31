/**
 * Phase 88 Batch235 — components/reconciliation/hooks/useReconciliationVisibility 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockPermissions: string[] = [];

vi.mock("@/store/menuStore", () => ({
  useMenuStore: vi.fn((selector: any) => selector({ permissions: mockPermissions })),
}));

import { useReconciliationVisibility } from "../useReconciliationVisibility";

describe("reconciliation/hooks/useReconciliationVisibility", () => {
  it("空 permissions → false", () => {
    mockPermissions.length = 0;
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(false);
  });

  it("无权限 perm → false", () => {
    mockPermissions.length = 0;
    mockPermissions.push("other:perm");
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(false);
  });

  it("有 asset:reconciliation:list → true", () => {
    mockPermissions.length = 0;
    mockPermissions.push("asset:reconciliation:list");
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(true);
  });

  it("混合权限 含目标 perm → true", () => {
    mockPermissions.length = 0;
    mockPermissions.push("user:read", "asset:reconciliation:list", "role:read");
    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(true);
  });
});
