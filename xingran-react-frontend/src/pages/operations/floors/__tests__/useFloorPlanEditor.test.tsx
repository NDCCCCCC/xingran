/**
 * Phase 88 Batch90 — operations/floors/useFloorPlanEditor 测试(81 stmts, 16.0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  wallApi: { list: vi.fn(), save: vi.fn() },
  doorApi: { list: vi.fn(), save: vi.fn() },
  workstationApi: { list: vi.fn() },
  floorPlanTextApi: { list: vi.fn(), save: vi.fn() },
}));

import { useFloorPlanEditor } from "../useFloorPlanEditor";

function wrapper({ children }: { children: React.ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useFloorPlanEditor", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    expect(result.current.floorPlanData).toBeNull();
    expect(result.current.floorPlanLoading).toBe(false);
    expect(result.current.isEditMode).toBe(false);
  });

  it("setEditMode → 切换状态", () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    act(() => result.current.setEditMode(true));
    expect(result.current.isEditMode).toBe(true);
  });

  it("loadFloorPlanData: 有 floorId → 加载", async () => {
    const { wallApi, doorApi, workstationApi, floorPlanTextApi } = await import("@/lib/opsApi");
    vi.mocked(wallApi.list).mockResolvedValueOnce({
      data: { list: [{ id: "w1", points: "[]" }] },
    } as any);
    vi.mocked(doorApi.list).mockResolvedValueOnce({ data: { list: [] } } as any);
    vi.mocked(workstationApi.list).mockResolvedValueOnce({
      data: { list: [], total: 0 },
    } as any);
    vi.mocked(floorPlanTextApi.list).mockResolvedValueOnce({ data: { list: [] } } as any);

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.loadFloorPlanData({ floorId: "f1" });
    });
    expect(wallApi.list).toHaveBeenCalled();
    expect(result.current.floorPlanData).toBeDefined();
  });

  it("loadFloorPlanData: 失败 → catch 路径", async () => {
    const { wallApi } = await import("@/lib/opsApi");
    vi.mocked(wallApi.list).mockRejectedValueOnce(new Error("net"));

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.loadFloorPlanData({ floorId: "f1" });
    });
    expect(result.current.floorPlanLoading).toBe(false);
  });

  it("resetFloorPlan → 清空数据", () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    act(() => result.current.setEditMode(true));
    act(() => result.current.resetFloorPlan());
    expect(result.current.isEditMode).toBe(false);
    expect(result.current.floorPlanData).toBeNull();
  });

  it("saveFloorPlan: 无 data → 不发请求", async () => {
    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.saveFloorPlan();
    });
    expect(result.current.floorPlanLoading).toBe(false);
  });
});
