/**
 * Phase 88 Batch94 — operations/workstations/hooks/useWorkstationView 测试
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
  workstationApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
    updatePositions: vi.fn(() => Promise.resolve({ code: 0 })),
  },
}));

import { useWorkstationView } from "../useWorkstationView";

function wrapper({ children }: { children: React.ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const floorOptions = [{ code: "F1", name: "F1" } as any, { code: "F2", name: "F2" } as any];

describe("useWorkstationView", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    expect(result.current.viewMode).toBe("table");
    expect(result.current.selectedFloorForPlan).toBe("");
    expect(result.current.floorPlanWorkstations).toEqual([]);
  });

  it("setViewMode 切换视图", () => {
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    act(() => result.current.setViewMode("floorplan"));
    expect(result.current.viewMode).toBe("floorplan");
  });

  it("handleFloorChangeForPlan → 切换楼层并加载", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.list).mockResolvedValueOnce({
      data: {
        list: [{ id: "w1", workstationCode: "WS001", x: 0, y: 0 }],
      },
    } as any);

    const { result } = renderHook(() => useWorkstationView(floorOptions), { wrapper });

    await act(async () => {
      result.current.handleFloorChangeForPlan("F1");
    });
    expect(result.current.selectedFloorForPlan).toBe("F1");
  });

  it("handleFloorChangeForPlan 空 code → 不发请求", async () => {
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    await act(async () => {
      result.current.handleFloorChangeForPlan("");
    });
    expect(result.current.floorPlanWorkstations).toEqual([]);
  });

  it("handlePositionUpdate → 调用 updatePositions + 更新本地", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.updatePositions).mockResolvedValueOnce({ code: 0 } as any);

    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    const items = [{ id: "w1", positionX: 10, positionY: 20 }];
    await act(async () => {
      await result.current.handlePositionUpdate(items);
    });
    expect(workstationApi.updatePositions).toHaveBeenCalledWith(items);
  });

  it("handlePositionUpdate 抛错 → rethrow", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.updatePositions).mockRejectedValueOnce(new Error("net"));

    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    await expect(async () => {
      await act(async () => {
        await result.current.handlePositionUpdate([{ id: "w1", positionX: 0, positionY: 0 }]);
      });
    }).rejects.toThrow("net");
  });

  it("handleFloorPlanEdit → 打开 modal", () => {
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    const openModal = vi.fn();
    act(() => {
      result.current.handleFloorPlanEdit({ id: "w1" } as any, openModal);
    });
    expect(openModal).toHaveBeenCalledWith({ id: "w1" });
  });

  it("切换到 floorplan 视图 + 有 floorOptions → 自动加载第一个", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.list).mockResolvedValueOnce({
      data: { list: [{ id: "w1", workstationCode: "WS001" }] },
    } as any);

    const { result } = renderHook(() => useWorkstationView(floorOptions), { wrapper });
    act(() => result.current.setViewMode("floorplan"));

    // 等 setTimeout(0) 触发
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
    expect(result.current.selectedFloorForPlan).toBe("F1");
  });
});
