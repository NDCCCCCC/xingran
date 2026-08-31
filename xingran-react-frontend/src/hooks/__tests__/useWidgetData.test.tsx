/**
 * Phase 88 Batch372 — hooks/useWidgetData 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockGetCached = vi.fn();
const mockCache = vi.fn();
vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({
    getCachedWidgetData: mockGetCached,
    cacheWidgetData: mockCache,
  })),
}));

vi.mock("@/components/dashboard/utils/dataFetcher", () => ({
  dataFetcher: {
    fetch: vi.fn(async () => ({ data: { value: 42 }, error: null })),
  },
}));

import { useWidgetData } from "../useWidgetData";

const sampleWidget: any = {
  id: "w1",
  type: "stat",
  title: "Test",
  enabled: true,
  refreshInterval: 60,
  dataSource: { type: "api", url: "/test" },
};

describe("hooks/useWidgetData", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetCached.mockReturnValue(null);
  });

  it("初始 loading=true", () => {
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    expect(result.current.loading).toBe(true);
    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it("挂载后调 dataFetcher.fetch + 返回 data", async () => {
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    expect(result.current.data).toEqual({ value: 42 });
  });

  it("widget.enabled=false → 不拉取", async () => {
    const { result } = renderHook(() => useWidgetData({ ...sampleWidget, enabled: false }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(result.current.data).toBeNull();
  });

  it("options.disabled=true → 不拉取", async () => {
    const { result } = renderHook(() => useWidgetData(sampleWidget, { disabled: true }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(result.current.data).toBeNull();
  });

  it("refresh() 调 fetchData(false)", async () => {
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    await waitFor(() => expect(result.current.loading).toBe(false));
    await act(async () => {
      await result.current.refresh();
    });
    expect(result.current.isRefreshing).toBe(false);
  });

  it("refresh() cached hit → 从缓存返回", async () => {
    mockGetCached.mockReturnValue({ value: "cached-value" });
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    await waitFor(() => expect(result.current.loading).toBe(false));
    // 重新挂载 mock
    mockGetCached.mockReturnValue({ value: "from-cache-2" });
    await act(async () => {
      await result.current.refresh();
    });
    // data 来自缓存而不是 fetch
    expect(result.current.data).toEqual({ value: "from-cache-2" });
  });

  it("fetch 返回 error → error state", async () => {
    const { dataFetcher } = await import("@/components/dashboard/utils/dataFetcher");
    vi.mocked(dataFetcher.fetch).mockResolvedValueOnce({
      data: null,
      error: "网络错误",
    });
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    await waitFor(() => {
      expect(result.current.error).toBe("网络错误");
    });
  });

  it("fetch throw → error state", async () => {
    const { dataFetcher } = await import("@/components/dashboard/utils/dataFetcher");
    vi.mocked(dataFetcher.fetch).mockRejectedValueOnce(new Error("crash"));
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    await waitFor(() => {
      expect(result.current.error).toBe("crash");
    });
  });

  it("成功 fetch 后缓存", async () => {
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    expect(mockCache).toHaveBeenCalledWith("w1", { value: 42 });
  });

  it("refreshInterval 自定义 → useMemo 缓存", () => {
    const { result, rerender } = renderHook(({ widget, opts }) => useWidgetData(widget, opts), {
      initialProps: { widget: sampleWidget, opts: { refreshInterval: 30 } as any },
    });
    expect(result.current.isRefreshing).toBe(false);
    rerender({ widget: sampleWidget, opts: { refreshInterval: 30 } as any });
    expect(result.current.isRefreshing).toBe(false);
  });

  it("isRefreshing 字段存在", () => {
    const { result } = renderHook(() => useWidgetData(sampleWidget));
    expect(typeof result.current.isRefreshing).toBe("boolean");
  });
});
