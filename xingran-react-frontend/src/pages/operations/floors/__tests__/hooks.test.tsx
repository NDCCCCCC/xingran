/**
 * Phase 85 — floors hooks 测试(mock opsApi)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useBuildingOptions } from "../useBuildingOptions";
import { useFloorStatistics } from "../useFloorStatistics";

vi.mock("@/lib/opsApi", () => ({
  buildingApi: {
    list: vi.fn(() =>
      Promise.resolve({
        data: {
          list: [
            { id: "b1", code: "B1", name: "一号楼" },
            { id: "b2", code: "B2", name: "二号楼" },
          ],
        },
      })
    ),
  },
  floorApi: {
    statistics: vi.fn(() => Promise.resolve({ total: 10, active: 8, inactive: 2 })),
  },
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

describe("useBuildingOptions", () => {
  beforeEach(() => vi.clearAllMocks());

  it("initializes with empty options", () => {
    const { result } = renderHook(() => useBuildingOptions());
    expect(result.current.buildingOptions).toEqual([]);
  });

  it("loadBuildingOptions maps API result to options", async () => {
    const { result } = renderHook(() => useBuildingOptions());
    await act(async () => {
      await result.current.loadBuildingOptions();
    });
    await waitFor(() => {
      expect(result.current.buildingOptions).toHaveLength(2);
      expect(result.current.buildingOptions[0]).toEqual({
        id: "b1",
        code: "B1",
        name: "一号楼",
      });
    });
  });
});

describe("useFloorStatistics", () => {
  beforeEach(() => vi.clearAllMocks());

  it("initializes with zero statistics", () => {
    const { result } = renderHook(() => useFloorStatistics());
    expect(result.current.statistics).toEqual({ total: 0, active: 0, inactive: 0 });
  });

  it("loadStatistics sets stats from API", async () => {
    const { result } = renderHook(() => useFloorStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    await waitFor(() => {
      expect(result.current.statistics).toEqual({ total: 10, active: 8, inactive: 2 });
    });
  });
});
