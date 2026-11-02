/**
 * useWidgetData / useBatchWidgetData 测试
 *
 * 覆盖：初始加载成功/错误 / disabled 与 enabled=false 短路 /
 * 刷新命中缓存 / fetch 抛错 / useBatchWidgetData 批量+失败回退 null。
 * dashboardStore 用真实实现 + setState(initialState) reset(D-05)。
 *
 * Sprint 1 Fix #5: hooks now use React Query internally, so renderHook
 * must wrap with QueryClientProvider (matches __tests__/useWidgetData.test.tsx pattern).
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

const fetchMock = vi.hoisted(() => vi.fn());

vi.mock("@/components/dashboard/utils/dataFetcher", () => ({
  dataFetcher: { fetch: fetchMock },
}));

import { useWidgetData, useBatchWidgetData } from "./useWidgetData";
import { useDashboardStore } from "@/store/dashboardStore";
import type { WidgetConfig } from "@/types/dashboard";

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

function makeWidget(overrides: Partial<WidgetConfig> = {}): WidgetConfig {
  return {
    id: "w1",
    type: "stat-card",
    title: "测试 Widget",
    position: { x: 0, y: 0, w: 4, h: 3 },
    dataSource: { type: "static", data: { value: 1 } },
    display: { type: "stat-card" },
    enabled: true,
    refreshInterval: 0, // 0 = 关闭自动刷新,测试不引入 interval
    ...overrides,
  } as WidgetConfig;
}

describe("useWidgetData", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    // D-05: Zustand store reset,不包 Provider
    useDashboardStore.setState({
      widgetDataCache: new Map(),
      currentDashboard: null,
      hasUnsavedChanges: false,
    });
  });

  it("初始加载成功:写入 data 并缓存到 store", async () => {
    fetchMock.mockResolvedValue({ data: { value: 42 }, timestamp: 1, error: undefined });
    const { result } = renderHook(() => useWidgetData(makeWidget()), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toEqual({ value: 42 });
    expect(result.current.error).toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const cached = useDashboardStore.getState().getCachedWidgetData("w1");
    expect(cached).toEqual({ value: 42 });
  });

  it("数据源返回 error 字段时置 error 不写 data", async () => {
    fetchMock.mockResolvedValue({ data: null, timestamp: 1, error: "source failed" });
    const { result } = renderHook(() => useWidgetData(makeWidget()), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe("source failed");
    expect(result.current.data).toBeNull();
  });

  it("fetch 抛错时置 error message", async () => {
    fetchMock.mockRejectedValue(new Error("network down"));
    const { result } = renderHook(() => useWidgetData(makeWidget()), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe("network down");
    expect(result.current.isRefreshing).toBe(false);
  });

  it("disabled=true 不发起请求;loading 保持初始 false (useQuery disabled 语义)", async () => {
    const { result } = renderHook(() => useWidgetData(makeWidget(), { disabled: true }), {
      wrapper: makeWrapper(),
    });

    // useQuery enabled=false: query is not active, no fetch, isLoading=false
    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.current.loading).toBe(false);
  });

  it("widget.enabled=false 不发起请求", async () => {
    renderHook(() => useWidgetData(makeWidget({ enabled: false })), {
      wrapper: makeWrapper(),
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("refresh 触发重新 fetch (useQuery refetch 不读 dashboardStore L1)", async () => {
    fetchMock.mockResolvedValue({ data: { v: 1 }, timestamp: 1, error: undefined });

    const { result } = renderHook(() => useWidgetData(makeWidget()), { wrapper: makeWrapper() });

    // 初始 fetch
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // 手动 refresh: queryClient.refetch 始终重跑 queryFn (不等 staleTime);
    // dashboardStore L1 只写不读 — React Query 用自己的 cache
    await act(async () => {
      await result.current.refresh();
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(result.current.data).toEqual({ v: 1 });
  });

  it("refresh 后再次 fetch,data 更新", async () => {
    let callIdx = 0;
    fetchMock.mockImplementation(async () => {
      callIdx++;
      return { data: { v: callIdx * 10 }, timestamp: callIdx, error: undefined };
    });

    const { result } = renderHook(() => useWidgetData(makeWidget()), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toEqual({ v: 10 });

    // Trigger refetch and wait for the new data to land
    await act(async () => {
      await result.current.refresh();
    });
    // React Query refetch: queryFn runs, but query.data may lag — wait for next fetch tick
    await waitFor(() => expect(result.current.data).toEqual({ v: 20 }));
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});

describe("useBatchWidgetData", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    useDashboardStore.setState({ widgetDataCache: new Map() });
  });

  it("批量拉取所有 widget 并返回 dataMap", async () => {
    fetchMock.mockImplementation(async (ds: { data: unknown }) => ({
      data: ds.data,
      timestamp: 1,
      error: undefined,
    }));

    const widgets = [
      makeWidget({ id: "a", dataSource: { type: "static", data: "A" } }),
      makeWidget({ id: "b", dataSource: { type: "static", data: "B" } }),
    ];
    const { result } = renderHook(() => useBatchWidgetData(widgets), {
      wrapper: makeWrapper(),
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.dataMap).toEqual({ a: "A", b: "B" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("单个 widget 失败时对应项从 dataMap 省略 不中断整体", async () => {
    fetchMock.mockImplementation(async (ds: { data: { fail?: boolean } }) => {
      if (ds.data.fail) throw new Error("boom");
      return { data: ds.data, timestamp: 1, error: undefined };
    });

    const widgets = [
      makeWidget({ id: "ok", dataSource: { type: "static", data: { v: 1 } } }),
      makeWidget({ id: "bad", dataSource: { type: "static", data: { fail: true } } }),
    ];
    const { result } = renderHook(() => useBatchWidgetData(widgets), {
      wrapper: makeWrapper(),
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    // useQuery 语义:失败项 catch+continue,从 dataMap 省略 (key 不存在)
    expect(result.current.dataMap).toEqual({ ok: { v: 1 } });
    expect("bad" in result.current.dataMap).toBe(false);
  });

  it("disabled=true 不拉取,loading=false (useQuery disabled 语义)", () => {
    const widgets = [makeWidget({ id: "a" })];
    const { result } = renderHook(() => useBatchWidgetData(widgets, { disabled: true }), {
      wrapper: makeWrapper(),
    });
    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.current.loading).toBe(false);
  });
});
