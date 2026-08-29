/**
 * Phase 88 Batch64 — useLogActions hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useLogActions } from "../hooks/useLogActions";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const baseParams = () => ({
  activeTab: "oper",
  fetchOperLogs: vi.fn().mockResolvedValue(undefined),
  fetchLoginLogs: vi.fn().mockResolvedValue(undefined),
});

describe("useLogActions", () => {
  it("initial state", () => {
    const { result } = renderHook(() => useLogActions(baseParams()), { wrapper: wrap });
    expect(result.current.detailModalVisible).toBe(false);
    expect(result.current.selectedLog).toBeNull();
    expect(typeof result.current.handleViewDetail).toBe("function");
    expect(typeof result.current.handleClearLogs).toBe("function");
    expect(typeof result.current.handleRefresh).toBe("function");
  });

  it("handleViewDetail 设 selectedLog + 开 modal", () => {
    const { result } = renderHook(() => useLogActions(baseParams()), { wrapper: wrap });
    act(() => result.current.handleViewDetail({ id: "log1", message: "x" }));
    expect(result.current.detailModalVisible).toBe(true);
    expect(result.current.selectedLog?.id).toBe("log1");
  });

  it("handleRefresh activeTab=oper 调 fetchOperLogs", () => {
    const { result } = renderHook(() => useLogActions(baseParams()), { wrapper: wrap });
    act(() => result.current.handleRefresh());
  });

  it("setters 直写 state", () => {
    const { result } = renderHook(() => useLogActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.setDetailModalVisible(true);
      result.current.setSelectedLog({ id: "x" });
    });
    expect(result.current.detailModalVisible).toBe(true);
    expect(result.current.selectedLog?.id).toBe("x");
  });
});
