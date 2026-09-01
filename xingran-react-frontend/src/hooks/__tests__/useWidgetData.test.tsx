/**
 * Phase 88 Batch385 — hooks/useWidgetData 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/dashboard/utils/dataFetcher", () => ({
  dataFetcher: {
    fetch: vi.fn(async () => ({ data: { value: 42 }, error: null })),
  },
}));

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({
    getCachedWidgetData: vi.fn(() => null),
    cacheWidgetData: vi.fn(),
  })),
}));

import { useWidgetData, useBatchWidgetData } from "../useWidgetData";
import type { WidgetConfig } from "@/types/dashboard";

const mockWidget: WidgetConfig = {
  id: "w1",
  type: "stat-card",
  title: "Test",
  dataSource: { endpoint: "/api/test", method: "GET" },
  refreshInterval: 60,
  enabled: true,
  layout: { x: 0, y: 0, w: 4, h: 2 },
};

describe("hooks/useWidgetData", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("返回 data/loading/error/refresh/isRefreshing", () => {
    const { result } = renderHook(() => useWidgetData(mockWidget));
    expect(result.current.data).toBeNull();
    expect(typeof result.current.loading).toBe("boolean");
    expect(result.current.error).toBeNull();
    expect(typeof result.current.refresh).toBe("function");
    expect(typeof result.current.isRefreshing).toBe("boolean");
  });

  it("disabled=true 不抛错", () => {
    const { result } = renderHook(() => useWidgetData(mockWidget, { disabled: true }));
    expect(result.current.data).toBeNull();
  });

  it("refresh 是函数", () => {
    const { result } = renderHook(() => useWidgetData(mockWidget));
    expect(typeof result.current.refresh).toBe("function");
  });
});

describe("hooks/useBatchWidgetData", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("返回 dataMap/loading", () => {
    const { result } = renderHook(() => useBatchWidgetData([mockWidget]));
    expect(typeof result.current.dataMap).toBe("object");
    expect(typeof result.current.loading).toBe("boolean");
  });

  it("空数组不抛错", () => {
    const { result } = renderHook(() => useBatchWidgetData([]));
    expect(typeof result.current.dataMap).toBe("object");
  });

  it("disabled=true 不抛错", () => {
    const { result } = renderHook(() => useBatchWidgetData([mockWidget], { disabled: true }));
    expect(typeof result.current.dataMap).toBe("object");
  });
});
