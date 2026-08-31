/**
 * Phase 88 Batch233 — pages/operations/floors/useFloorStatistics 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockStatistics = vi.fn();
vi.mock("@/lib/opsApi", () => ({
  floorApi: {
    statistics: (...args: any[]) => mockStatistics(...args),
  },
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

import { useFloorStatistics } from "../useFloorStatistics";

describe("operations/floors/useFloorStatistics", () => {
  beforeEach(() => {
    mockStatistics.mockReset();
  });

  it("初始 statistics 全 0", () => {
    const { result } = renderHook(() => useFloorStatistics());
    expect(result.current.statistics).toEqual({ total: 0, active: 0, inactive: 0 });
  });

  it("loadStatistics 成功 → 设置 stats", async () => {
    mockStatistics.mockResolvedValue({ total: 100, active: 80, inactive: 20 });
    const { result } = renderHook(() => useFloorStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics).toEqual({ total: 100, active: 80, inactive: 20 });
  });

  it("loadStatistics 部分字段缺失 → fallback 0", async () => {
    mockStatistics.mockResolvedValue({});
    const { result } = renderHook(() => useFloorStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics).toEqual({ total: 0, active: 0, inactive: 0 });
  });

  it("loadStatistics 失败 → handleApiError", async () => {
    mockStatistics.mockRejectedValue(new Error("network"));
    const { result } = renderHook(() => useFloorStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    const errorHandler = await import("@/utils/errorHandler");
    expect(errorHandler.handleApiError).toHaveBeenCalled();
    expect(result.current.statistics).toEqual({ total: 0, active: 0, inactive: 0 });
  });
});
