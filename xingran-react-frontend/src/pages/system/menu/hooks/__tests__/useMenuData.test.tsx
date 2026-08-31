/**
 * Phase 88 Batch294 — pages/system/menu/hooks/useMenuData 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockPost = vi.fn();
vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return { ...actual, post: (...args: any[]) => mockPost(...args) };
});

import { useMenuData } from "../useMenuData";

describe("system/menu/hooks/useMenuData", () => {
  beforeEach(() => {
    mockPost.mockReset();
  });

  it("初始 menus=[] + loading=false", () => {
    const { result } = renderHook(() => useMenuData());
    expect(result.current.menus).toEqual([]);
    expect(result.current.loading).toBe(false);
    expect(result.current.statistics.total).toBe(0);
  });

  it("loadMenus 成功 → 设置数据", async () => {
    mockPost.mockResolvedValue({
      data: [
        { id: "m1", menuName: "系统", menuType: "M", children: [] },
        { id: "m2", menuName: "用户", menuType: "C", children: [] },
      ],
    });
    const { result } = renderHook(() => useMenuData());
    await act(async () => {
      await result.current.loadMenus();
    });
    expect(result.current.menus.length).toBe(2);
    expect(result.current.statistics.directories).toBe(1);
  });

  it("loadMenus 失败 → 默认值", async () => {
    mockPost.mockRejectedValue(new Error("net"));
    const { result } = renderHook(() => useMenuData());
    await act(async () => {
      await result.current.loadMenus();
    });
    expect(result.current.menus).toEqual([]);
    expect(result.current.statistics.total).toBe(0);
    expect(result.current.loading).toBe(false);
  });

  it("loadMenus 传 searchParams", async () => {
    mockPost.mockResolvedValue({ data: [] });
    const { result } = renderHook(() => useMenuData());
    await act(async () => {
      await result.current.loadMenus({ name: "test" });
    });
    expect(mockPost).toHaveBeenCalledWith("/system/menus/tree", { name: "test" });
  });

  it("loadMenus null data → 空数组", async () => {
    mockPost.mockResolvedValue({ data: null });
    const { result } = renderHook(() => useMenuData());
    await act(async () => {
      await result.current.loadMenus();
    });
    expect(result.current.menus).toEqual([]);
  });
});
