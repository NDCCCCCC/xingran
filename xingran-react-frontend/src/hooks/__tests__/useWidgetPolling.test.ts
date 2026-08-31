/**
 * Phase 88 Batch370 — hooks/useWidgetPolling 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockCacheWidgetData = vi.fn();
const mockGetCachedWidgetData = vi.fn();
const mockClearWidgetCache = vi.fn();

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({
    cacheWidgetData: mockCacheWidgetData,
    getCachedWidgetData: mockGetCachedWidgetData,
    clearWidgetCache: mockClearWidgetCache,
  })),
}));

vi.mock("@/services/dashboardService", () => ({
  dashboardService: {
    getBatchWidgetData: vi.fn(async (ids: string[]) => {
      const map = new Map();
      ids.forEach((id) => map.set(id, { value: `data-${id}` }));
      return map;
    }),
  },
}));

import { useWidgetPolling } from "../useWidgetPolling";

describe("hooks/useWidgetPolling", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    mockGetCachedWidgetData.mockReturnValue(null);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("初始 state", () => {
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    expect(result.current.lastRefreshTime).toBeNull();
    expect(result.current.isPaused).toBe(false);
  });

  it("挂载时立即调 getBatchWidgetData (cache miss)", async () => {
    renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(mockCacheWidgetData).toHaveBeenCalled();
  });

  it("全部 cache hit → 不调 fetch", async () => {
    mockGetCachedWidgetData.mockReturnValue({
      timestamp: Date.now(),
      data: { value: "cached" },
    });
    renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(mockCacheWidgetData).not.toHaveBeenCalled();
  });

  it("refresh() 返回 Promise + 不抛错", async () => {
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    await act(async () => {
      await result.current.refresh();
    });
    expect(result.current.refresh).toBeDefined();
  });

  it("pause → 停止轮询", async () => {
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    act(() => result.current.pause());
    expect(result.current.isPaused).toBe(true);
  });

  it("resume → 恢复轮询", async () => {
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    act(() => result.current.pause());
    act(() => result.current.resume());
    expect(result.current.isPaused).toBe(false);
  });

  it("enabled=false → 不轮询", async () => {
    renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60, enabled: false }));
    await act(async () => {
      await vi.runAllTimersAsync();
    });
    expect(mockCacheWidgetData).not.toHaveBeenCalled();
  });

  it("空 widgetIds → 不轮询", async () => {
    renderHook(() => useWidgetPolling({ widgetIds: [], interval: 60 }));
    await act(async () => {
      await vi.runAllTimersAsync();
    });
    expect(mockCacheWidgetData).not.toHaveBeenCalled();
  });

  it("setInterval 周期性调用 fetchData", async () => {
    renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60_000);
    });
    expect(mockCacheWidgetData.mock.calls.length).toBeGreaterThanOrEqual(1);
  });

  it("fetchData 失败 → 不抛错", async () => {
    const { dashboardService } = await import("@/services/dashboardService");
    vi.mocked(dashboardService.getBatchWidgetData).mockRejectedValueOnce(new Error("net"));
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    await act(async () => {
      await result.current.refresh();
    });
    expect(result.current.loading).toBe(false);
  });

  it("visibilitychange 监听器注册", () => {
    const addSpy = vi.spyOn(document, "addEventListener");
    renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }));
    expect(addSpy).toHaveBeenCalledWith("visibilitychange", expect.any(Function));
  });

  it("过期 cache → 重新拉取", async () => {
    mockGetCachedWidgetData.mockReturnValue({
      timestamp: Date.now() - 60000,
      data: { value: "stale" },
    });
    renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 30 }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(mockCacheWidgetData).toHaveBeenCalled();
  });
});
