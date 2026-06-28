/**
 * useReconciliationVisibility 单元测试
 *
 * 锁定行为(B4 修复 + D-A1-03):
 *   - hook 读 useMenuStore.permissions(不读 authStore.perms)
 *   - 当 permissions 含 'asset:reconciliation:list' → 返回 true
 *   - permissions 为空数组 / undefined / 不含该 perm → 返回 false
 *
 * 用 vitest + @testing-library/react renderHook
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";

// mock useMenuStore — 在 import hook 之前 mock,因为 hook 内部直接调用 useMenuStore
const mockUseMenuStore = vi.fn();
vi.mock("@/store/menuStore", () => ({
  useMenuStore: (selector: (s: { permissions: string[] | undefined }) => unknown) =>
    mockUseMenuStore(selector) as unknown,
}));

import { useReconciliationVisibility } from "../useReconciliationVisibility";

describe("useReconciliationVisibility", () => {
  beforeEach(() => {
    mockUseMenuStore.mockReset();
  });

  it("returns true when useMenuStore.permissions includes 'asset:reconciliation:list'", () => {
    // mock selector: store.permissions 数组含目标 perm
    mockUseMenuStore.mockImplementation((selector: (s: { permissions: string[] | undefined }) => unknown) =>
      selector({ permissions: ["asset:reconciliation:list", "other:perm"] })
    );

    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(true);
  });

  it("returns false when useMenuStore.permissions is empty array", () => {
    mockUseMenuStore.mockImplementation((selector: (s: { permissions: string[] | undefined }) => unknown) =>
      selector({ permissions: [] })
    );

    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(false);
  });

  it("returns false when useMenuStore.permissions is undefined", () => {
    mockUseMenuStore.mockImplementation((selector: (s: { permissions: string[] | undefined }) => unknown) =>
      selector({ permissions: undefined })
    );

    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(false);
  });

  it("returns false when useMenuStore.permissions lacks 'asset:reconciliation:list'", () => {
    mockUseMenuStore.mockImplementation((selector: (s: { permissions: string[] | undefined }) => unknown) =>
      selector({ permissions: ["other:perm:list", "yet:another:perm"] })
    );

    const { result } = renderHook(() => useReconciliationVisibility());
    expect(result.current).toBe(false);
  });
});
