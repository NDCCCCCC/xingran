/**
 * Phase 88 Batch25 — floors useFloorStatistics + useBuildingOptions 钩子补测
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

import { useFloorStatistics } from "../useFloorStatistics";
import { useBuildingOptions } from "../useBuildingOptions";
import { floorApi, buildingApi } from "@/lib/opsApi";

describe("useFloorStatistics", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("initial 全 0", () => {
    const { result } = renderHook(() => useFloorStatistics());
    expect(result.current.statistics).toEqual({ total: 0, active: 0, inactive: 0 });
  });

  it("loadStatistics 写入统计数据", async () => {
    const spy = vi.spyOn(floorApi, "statistics").mockResolvedValue({
      total: 10,
      active: 7,
      inactive: 3,
    } as any);
    const { result } = renderHook(() => useFloorStatistics());

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(spy).toHaveBeenCalled();
    expect(result.current.statistics).toEqual({ total: 10, active: 7, inactive: 3 });
  });

  it("loadStatistics null 字段兜底 0", async () => {
    vi.spyOn(floorApi, "statistics").mockResolvedValue({} as any);
    const { result } = renderHook(() => useFloorStatistics());

    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics).toEqual({ total: 0, active: 0, inactive: 0 });
  });

  it("loadStatistics error 静默不写", async () => {
    vi.spyOn(floorApi, "statistics").mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useFloorStatistics());

    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics).toEqual({ total: 0, active: 0, inactive: 0 });
  });
});

describe("useBuildingOptions", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("initial 空数组", () => {
    const { result } = renderHook(() => useBuildingOptions());
    expect(result.current.buildingOptions).toEqual([]);
  });

  it("loadBuildingOptions 映射 list 到 options", async () => {
    vi.spyOn(buildingApi, "list").mockResolvedValue({
      data: {
        list: [
          { id: "b1", code: "BLD-A", name: "A 栋" },
          { id: "b2", code: "BLD-B", name: "B 栋" },
        ],
        total: 2,
      },
    } as any);
    const { result } = renderHook(() => useBuildingOptions());

    await act(async () => {
      await result.current.loadBuildingOptions();
    });

    expect(result.current.buildingOptions).toEqual([
      { id: "b1", code: "BLD-A", name: "A 栋" },
      { id: "b2", code: "BLD-B", name: "B 栋" },
    ]);
  });

  it("list 空时 options 空", async () => {
    vi.spyOn(buildingApi, "list").mockResolvedValue({ data: { list: [], total: 0 } } as any);
    const { result } = renderHook(() => useBuildingOptions());

    await act(async () => {
      await result.current.loadBuildingOptions();
    });
    expect(result.current.buildingOptions).toEqual([]);
  });

  it("error 静默不写", async () => {
    vi.spyOn(buildingApi, "list").mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useBuildingOptions());

    await act(async () => {
      await result.current.loadBuildingOptions();
    });
    expect(result.current.buildingOptions).toEqual([]);
  });
});
