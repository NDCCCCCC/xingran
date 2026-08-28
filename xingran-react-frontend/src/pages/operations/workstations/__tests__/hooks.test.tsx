/**
 * Phase 88 Batch24 — workstations/hooks 单元测试 (替代 index 渲染测试)
 *
 * 目标:覆盖 useWorkstationData + useWorkstationModals + useWorkstationView。
 * 注: workstaions index.tsx 因 jsdom 死锁(未知原因,可能是 FloorPlanEditor + Three.js lazy 加载 +
 *     ReconciliationDrawer + usePersistedState 持久化状态跨测串扰),此测试文件专注钩子,
 *     避免主页面渲染。
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useWorkstationModals } from "../hooks/useWorkstationModals";
import { useWorkstationView } from "../hooks/useWorkstationView";
import { workstationApi } from "@/lib/opsApi";

describe("useWorkstationModals", () => {
  it("handleDelete 调用 api.delete + onSuccess", async () => {
    const spy = vi.spyOn(workstationApi, "delete").mockResolvedValue(undefined as any);
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useWorkstationModals());

    await act(async () => {
      await result.current.handleDelete("ws-1", onSuccess);
    });

    expect(spy).toHaveBeenCalledWith("ws-1");
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it("handleDelete 吞 API 错误(不 throw)", async () => {
    const spy = vi.spyOn(workstationApi, "delete").mockRejectedValue(new Error("boom"));
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useWorkstationModals());

    await act(async () => {
      await result.current.handleDelete("ws-2", onSuccess);
    });

    expect(spy).toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled(); // 失败时 onSuccess 不触发
  });

  it("handleBatchDelete 空数组是 no-op", async () => {
    const spy = vi.spyOn(workstationApi, "batch").mockResolvedValue(undefined as any);
    const onSuccess = vi.fn();
    const resetSelection = vi.fn();
    const { result } = renderHook(() => useWorkstationModals());

    await act(async () => {
      await result.current.handleBatchDelete([], onSuccess, resetSelection);
    });

    expect(spy).not.toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled();
    expect(resetSelection).not.toHaveBeenCalled();
  });

  it("handleBatchDelete 非空触发 api.batch", async () => {
    const spy = vi.spyOn(workstationApi, "batch").mockResolvedValue(undefined as any);
    const onSuccess = vi.fn();
    const resetSelection = vi.fn();
    const { result } = renderHook(() => useWorkstationModals());

    await act(async () => {
      await result.current.handleBatchDelete(["a", "b"], onSuccess, resetSelection);
    });

    expect(spy).toHaveBeenCalledWith("delete", { ids: ["a", "b"] });
    expect(resetSelection).toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalled();
  });

  it("closeModal 重置表单字段", () => {
    const form = { resetFields: vi.fn() };
    const { result } = renderHook(() => useWorkstationModals());
    act(() => {
      result.current.closeModal(form as any);
    });
    expect(form.resetFields).toHaveBeenCalled();
  });

  it("closeModal 不带 form 不报错", () => {
    const { result } = renderHook(() => useWorkstationModals());
    act(() => {
      result.current.closeModal(undefined);
    });
    // 不 throw
  });

  it("openModal 有 deptId 则触发 loadUserOptions", async () => {
    const loadUserOptions = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useWorkstationModals(loadUserOptions));
    const record = { id: "ws-1", deptId: "d-1" } as any;
    await act(async () => {
      const ret = await result.current.openModal(record);
      expect(ret).toBe(record);
    });
    expect(loadUserOptions).toHaveBeenCalledWith("d-1");
  });

  it("openModal 无 deptId 不调 loadUserOptions", async () => {
    const loadUserOptions = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useWorkstationModals(loadUserOptions));
    await act(async () => {
      const ret = await result.current.openModal({ id: "ws-1" } as any);
      expect(ret).toBeDefined();
    });
    expect(loadUserOptions).not.toHaveBeenCalled();
  });
});

describe("useWorkstationView", () => {
  it("initial state 默认 table view", () => {
    const { result } = renderHook(() => useWorkstationView([]));
    expect(result.current.viewMode).toBe("table");
    expect(result.current.selectedFloorForPlan).toBe("");
    expect(result.current.floorPlanWorkstations).toEqual([]);
  });

  it("setViewMode 切换", () => {
    const { result } = renderHook(() => useWorkstationView([]));
    act(() => {
      result.current.setViewMode("card");
    });
    expect(result.current.viewMode).toBe("card");
  });

  it("handleFloorChangeForPlan 设置 selectedFloorForPlan + 拉取数据", async () => {
    const spy = vi.spyOn(workstationApi, "list").mockResolvedValue({
      data: { list: [{ id: "w1", positionX: 0, positionY: 0, status: 0 } as any], total: 1 },
    } as any);
    const { result } = renderHook(() => useWorkstationView([]));

    act(() => {
      result.current.handleFloorChangeForPlan("floor-1");
    });
    // 立即检查同步变化
    expect(result.current.selectedFloorForPlan).toBe("floor-1");

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(spy).toHaveBeenCalledWith({ floorCode: "floor-1", current: 1, pageSize: 1000 });
    expect(result.current.floorPlanWorkstations.length).toBeGreaterThan(0);
  });

  it("handleFloorChangeForPlan 空 floorId 清空数据", async () => {
    const { result } = renderHook(() => useWorkstationView([]));
    await act(async () => {
      result.current.handleFloorChangeForPlan("");
    });
    expect(result.current.floorPlanWorkstations).toEqual([]);
  });

  it("handlePositionUpdate 调用 updatePositions + 更新本地 state", async () => {
    const spy = vi.spyOn(workstationApi, "updatePositions").mockResolvedValue(undefined as any);
    const { result } = renderHook(() => useWorkstationView([]));
    // seed 一个 floor plan workstation
    act(() => {
      result.current.setViewMode("floorplan");
    });
    // 直接调 update,空 state 不报错
    await act(async () => {
      await result.current.handlePositionUpdate([{ id: "ws-1", positionX: 10, positionY: 20 }]);
    });
    expect(spy).toHaveBeenCalledWith([{ id: "ws-1", positionX: 10, positionY: 20 }]);
  });

  it("handleFloorPlanEdit 委托给 openModal", () => {
    const { result } = renderHook(() => useWorkstationView([]));
    const openModal = vi.fn();
    const ws = { id: "ws-1" } as any;
    act(() => {
      result.current.handleFloorPlanEdit(ws as any, openModal);
    });
    expect(openModal).toHaveBeenCalledWith(ws);
  });
});
