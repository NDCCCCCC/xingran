/**
 * Phase 88 Batch90 — operations/floors/useFloorPlanEditor 测试(81 stmts, 16.0% → 高)
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
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
  workstationApi: {
    list: vi.fn(),
    getFloorWorkstationsAll: vi.fn(() => Promise.resolve([])),
    updatePositions: vi.fn(),
  },
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
    } as unknown);
    vi.mocked(doorApi.list).mockResolvedValueOnce({ data: { list: [] } } as unknown);
    // V130R-09 D-03-6: 楼层全集改走 getFloorWorkstationsAll(直接返回 WorkstationOps[])
    vi.mocked(workstationApi.getFloorWorkstationsAll).mockResolvedValueOnce([]);
    vi.mocked(floorPlanTextApi.list).mockResolvedValueOnce({ data: { list: [] } } as unknown);

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

/**
 * 回归测试：一次保存只弹一次「保存成功」toast。
 *
 * 缺陷现象：平面图编辑点一次「保存」弹出两个「保存成功」——
 * 一处来自 CADFloorPlanEditor.handleSave（未经 await，保存失败也会误报成功），
 * 一处来自 useFloorPlanEditor.saveFloorPlan（await 真实保存成功后）。
 * 修复后，成功提示只允许由 useFloorPlanEditor 在真实保存完成后发出。
 */
describe("saveFloorPlan 的保存提示（一次保存只弹一次 toast）", () => {
  const messageMock = {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
    loading: vi.fn(),
  };

  const floor = { id: "f1", name: "一层", floorNo: 1 };

  let useAppSpy: ReturnType<typeof vi.spyOn> | undefined;

  beforeEach(() => {
    messageMock.success.mockClear();
    messageMock.error.mockClear();
    messageMock.warning.mockClear();
    messageMock.info.mockClear();
    messageMock.loading.mockClear();
    useAppSpy = vi.spyOn(AntdApp, "useApp").mockReturnValue({
      message: messageMock,
    } as unknown as ReturnType<typeof AntdApp.useApp>);
  });

  afterEach(() => {
    useAppSpy?.mockRestore();
  });

  it("保存成功 → 恰好弹一次「保存成功」，且不弹「保存失败」", async () => {
    const { wallApi, doorApi, workstationApi, floorPlanTextApi } = await import("@/lib/opsApi");
    vi.mocked(wallApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(doorApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(workstationApi.getFloorWorkstationsAll).mockResolvedValue([]);
    vi.mocked(floorPlanTextApi.list).mockResolvedValue({ data: { list: [] } } as unknown);
    vi.mocked(workstationApi.updatePositions).mockResolvedValue(undefined as never);

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.saveFloorPlan(
        { walls: [], doors: [], texts: [], workstations: [] } as never,
        floor as never
      );
    });

    const successCalls = messageMock.success.mock.calls.filter((call) => call[0] === "保存成功");
    expect(successCalls).toHaveLength(1);
    expect(messageMock.error).not.toHaveBeenCalledWith("保存失败");
  });

  it("保存失败 → 弹「保存失败」，且不再误报「保存成功」", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.updatePositions).mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useFloorPlanEditor(), { wrapper });
    await act(async () => {
      await result.current.saveFloorPlan(
        { walls: [], doors: [], texts: [], workstations: [{ id: "ws1", x: 1, y: 2 }] } as never,
        floor as never
      );
    });

    const errorCalls = messageMock.error.mock.calls.filter((call) => call[0] === "保存失败");
    expect(errorCalls).toHaveLength(1);
    expect(messageMock.success).not.toHaveBeenCalledWith("保存成功");
  });
});
