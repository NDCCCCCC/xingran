/**
 * Phase 88 Batch131 — operations/workstations/hooks/useWorkstationView 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

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

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

import { useWorkstationView } from "../useWorkstationView";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useWorkstationView", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    expect(result.current.viewMode).toBe("table");
    expect(result.current.selectedFloorForPlan).toBe("");
    expect(result.current.floorPlanWorkstations).toEqual([]);
  });

  it("setViewMode 写入", () => {
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    act(() => result.current.setViewMode("floorplan"));
    expect(result.current.viewMode).toBe("floorplan");
  });

  it("handleFloorChangeForPlan 设置 selectedFloorForPlan + 触发加载", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.list).mockClear();
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    await act(async () => {
      result.current.handleFloorChangeForPlan("F1");
    });
    expect(result.current.selectedFloorForPlan).toBe("F1");
    expect(workstationApi.list).toHaveBeenCalled();
  });

  it("handleFloorChangeForPlan + 空 floorCode → 清空数据", async () => {
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    act(() => result.current.handleFloorChangeForPlan(""));
    expect(result.current.floorPlanWorkstations).toEqual([]);
  });

  it("handlePositionUpdate → 调用 updatePositions", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.updatePositions).mockClear();
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    await act(async () => {
      await result.current.handlePositionUpdate([{ id: "w1", positionX: 10, positionY: 20 }]);
    });
    expect(workstationApi.updatePositions).toHaveBeenCalled();
  });

  it("handlePositionUpdate 失败 → handleApiError + throw", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.updatePositions).mockRejectedValueOnce(new Error("net"));
    const { handleApiError } = await import("@/utils/errorHandler");
    vi.mocked(handleApiError).mockClear();
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    await expect(
      act(async () => {
        await result.current.handlePositionUpdate([{ id: "w1", positionX: 1, positionY: 1 }]);
      })
    ).rejects.toThrow("net");
    expect(handleApiError).toHaveBeenCalled();
  });

  it("handleFloorPlanEdit → 调用 openModal(record)", () => {
    const openModal = vi.fn();
    const { result } = renderHook(() => useWorkstationView([]), { wrapper });
    result.current.handleFloorPlanEdit({ id: "w1" } as any, openModal);
    expect(openModal).toHaveBeenCalledWith({ id: "w1" });
  });

  it("viewMode=floorplan + 有 floorOptions + 未选楼层 → 默认加载", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.list).mockClear();
    const { result } = renderHook(() => useWorkstationView([{ code: "F1", name: "1F" }]), {
      wrapper,
    });
    act(() => result.current.setViewMode("floorplan"));
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
    expect(workstationApi.list).toHaveBeenCalled();
  });
});
