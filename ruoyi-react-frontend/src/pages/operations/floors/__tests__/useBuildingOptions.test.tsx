/**
 * Phase 88 Batch260 — pages/operations/floors/useBuildingOptions 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockList = vi.fn();
vi.mock("@/lib/opsApi", () => ({
  buildingApi: {
    list: (...args: any[]) => mockList(...args),
  },
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

import { useBuildingOptions } from "../useBuildingOptions";

describe("operations/floors/useBuildingOptions", () => {
  beforeEach(() => {
    mockList.mockReset();
  });

  it("初始 buildingOptions=[]", () => {
    const { result } = renderHook(() => useBuildingOptions());
    expect(result.current.buildingOptions).toEqual([]);
  });

  it("loadBuildingOptions 成功 → 设置 options", async () => {
    mockList.mockResolvedValue({
      data: {
        list: [
          { id: "b1", code: "C1", name: "B1" },
          { id: "b2", code: "C2", name: "B2" },
        ],
      },
    });
    const { result } = renderHook(() => useBuildingOptions());
    await act(async () => {
      await result.current.loadBuildingOptions();
    });
    expect(result.current.buildingOptions.length).toBe(2);
    expect(result.current.buildingOptions[0].name).toBe("B1");
  });

  it("loadBuildingOptions 失败 → handleApiError", async () => {
    mockList.mockRejectedValue(new Error("net"));
    const { result } = renderHook(() => useBuildingOptions());
    await act(async () => {
      await result.current.loadBuildingOptions();
    });
    const errorHandler = await import("@/utils/errorHandler");
    expect(errorHandler.handleApiError).toHaveBeenCalled();
    expect(result.current.buildingOptions).toEqual([]);
  });

  it("loadBuildingOptions 空 list", async () => {
    mockList.mockResolvedValue({ data: { list: [] } });
    const { result } = renderHook(() => useBuildingOptions());
    await act(async () => {
      await result.current.loadBuildingOptions();
    });
    expect(result.current.buildingOptions).toEqual([]);
  });
});
