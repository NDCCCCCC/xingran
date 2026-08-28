/**
 * Phase 88 Batch24 — useWorkstationData 钩子单元测试
 *
 * 通过 renderHook 隔离测试,不触发 workstations/index.tsx 渲染。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({ data: [] }),
}));

import { useWorkstationData } from "../hooks/useWorkstationData";
import { workstationApi, floorApi } from "@/lib/opsApi";
import * as apiModule from "@/lib/api";

describe("useWorkstationData", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns all expected functions", () => {
    const { result } = renderHook(() => useWorkstationData(vi.fn(), vi.fn(), vi.fn(), vi.fn()));

    expect(result.current.loadStatistics).toBeTypeOf("function");
    expect(result.current.loadFloorOptions).toBeTypeOf("function");
    expect(result.current.loadDeptOptions).toBeTypeOf("function");
    expect(result.current.loadUserOptions).toBeTypeOf("function");
    expect(result.current.loadFloorPlanWorkstations).toBeTypeOf("function");
    expect(result.current.ensureUser).toBeTypeOf("function");
  });

  it("loadStatistics 调用 api 并写入 state", async () => {
    const setStatistics = vi.fn();
    const spy = vi
      .spyOn(workstationApi, "statistics")
      .mockResolvedValue({ total: 10, available: 3, occupied: 5, maintain: 2 } as any);

    const { result } = renderHook(() =>
      useWorkstationData(setStatistics, vi.fn(), vi.fn(), vi.fn())
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(spy).toHaveBeenCalledWith({});
    expect(setStatistics).toHaveBeenCalledWith({
      total: 10,
      available: 3,
      occupied: 5,
      maintain: 2,
    });
  });

  it("loadStatistics 传 orgId 透传到 api", async () => {
    const spy = vi
      .spyOn(workstationApi, "statistics")
      .mockResolvedValue({ total: 0, available: 0, occupied: 0, maintain: 0 } as any);

    const { result } = renderHook(() => useWorkstationData(vi.fn(), vi.fn(), vi.fn(), vi.fn()));

    await act(async () => {
      await result.current.loadStatistics("org-1");
    });

    expect(spy).toHaveBeenCalledWith({ orgId: "org-1" });
  });

  it("loadFloorOptions 调用 floorApi.searchOptions 并设置 options", async () => {
    const setFloorOptions = vi.fn();
    const spy = vi.spyOn(floorApi, "searchOptions").mockResolvedValue([
      { value: "f-1", label: "1F" },
      { value: "f-2", label: "2F" },
    ] as any);

    const { result } = renderHook(() =>
      useWorkstationData(vi.fn(), setFloorOptions, vi.fn(), vi.fn())
    );

    await act(async () => {
      await result.current.loadFloorOptions();
    });

    expect(spy).toHaveBeenCalledWith({});
    expect(setFloorOptions).toHaveBeenCalledWith([
      { id: "f-1", code: "f-1", name: "1F" },
      { id: "f-2", code: "f-2", name: "2F" },
    ]);
  });

  it("loadFloorOptions 透传 name keyword", async () => {
    const spy = vi.spyOn(floorApi, "searchOptions").mockResolvedValue([] as any);
    const { result } = renderHook(() => useWorkstationData(vi.fn(), vi.fn(), vi.fn(), vi.fn()));

    await act(async () => {
      await result.current.loadFloorOptions("org-1", "3F");
    });

    expect(spy).toHaveBeenCalledWith({ orgId: "org-1", name: "3F" });
  });

  it("loadUserOptions 调用 system/users/list + 设置 state", async () => {
    const setUserOptions = vi.fn();
    const postSpy = vi.fn().mockResolvedValue({
      data: { list: [{ id: "u1", username: "alice", nickname: "Al" }] },
    });
    vi.spyOn(apiModule, "post" as any).mockImplementation(postSpy as any);

    const { result } = renderHook(() =>
      useWorkstationData(vi.fn(), vi.fn(), vi.fn(), setUserOptions)
    );

    await act(async () => {
      await result.current.loadUserOptions("dept-1");
    });

    expect(postSpy).toHaveBeenCalledWith("/system/users/list", {
      current: 1,
      pageSize: 50,
      recursiveDeptId: "dept-1",
    });
    expect(setUserOptions).toHaveBeenCalledWith([{ id: "u1", username: "alice", nickname: "Al" }]);
  });

  it("loadFloorPlanWorkstations 空 floorCode 返回 []", async () => {
    const { result } = renderHook(() => useWorkstationData(vi.fn(), vi.fn(), vi.fn(), vi.fn()));

    await act(async () => {
      const list = await result.current.loadFloorPlanWorkstations("");
      expect(list).toEqual([]);
    });
  });

  it("loadFloorPlanWorkstations 有值时调用 api + 返回 list", async () => {
    const spy = vi.spyOn(workstationApi, "list").mockResolvedValue({
      data: { list: [{ id: "w1" }], total: 1 },
    } as any);

    const { result } = renderHook(() => useWorkstationData(vi.fn(), vi.fn(), vi.fn(), vi.fn()));

    await act(async () => {
      const list = await result.current.loadFloorPlanWorkstations("F1");
      expect(list).toEqual([{ id: "w1" }]);
    });

    expect(spy).toHaveBeenCalledWith({ floorCode: "F1", current: 1, pageSize: 1000 });
  });

  it("ensureUser 重复 id 保持原引用", () => {
    let state: any[] = [{ id: "u-1", username: "u1" }];
    const setter = (updater: any) => {
      state = typeof updater === "function" ? updater(state) : updater;
    };
    const { result } = renderHook(() => useWorkstationData(vi.fn(), vi.fn(), vi.fn(), setter));

    act(() => {
      result.current.ensureUser({ id: "u-1", username: "other" });
    });
    expect(state).toEqual([{ id: "u-1", username: "u1" }]);

    act(() => {
      result.current.ensureUser({ id: "u-2", username: "u2" });
    });
    expect(state).toEqual([
      { id: "u-1", username: "u1" },
      { id: "u-2", username: "u2" },
    ]);
  });

  it("ensureUser 无 username 兜底 '未命名用户'", () => {
    let state: any[] = [];
    const setter = (updater: any) => {
      state = typeof updater === "function" ? updater(state) : updater;
    };
    const { result } = renderHook(() => useWorkstationData(vi.fn(), vi.fn(), vi.fn(), setter));

    act(() => {
      result.current.ensureUser({ id: "u-3" });
    });
    expect(state).toEqual([{ id: "u-3", username: "未命名用户", nickname: undefined }]);
  });

  it("error 路径: loadStatistics catch 静默", async () => {
    vi.spyOn(workstationApi, "statistics").mockRejectedValue(new Error("boom"));
    const setStatistics = vi.fn();
    const { result } = renderHook(() =>
      useWorkstationData(setStatistics, vi.fn(), vi.fn(), vi.fn())
    );
    await act(async () => {
      await result.current.loadStatistics();
    });
    // setStatistics 不抛,且不调用 — 静默处理
    expect(setStatistics).not.toHaveBeenCalled();
  });
});
