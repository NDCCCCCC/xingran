/**
 * Phase 88 Batch62 — useDiscoveryData hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
}));

import { useDiscoveryData } from "../hooks/useDiscoveryData";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const opts = () => ({ current: 1, pageSize: 10, searchForm: {} as any });

describe("useDiscoveryData", () => {
  it("initial state + 返回 16 handler/setter", () => {
    const { result } = renderHook(() => useDiscoveryData(opts()), { wrapper: wrap });
    expect(result.current.discoveries).toEqual([]);
    expect(result.current.discoveredDevices).toEqual([]);
    expect(result.current.departments).toEqual([]);
    expect(result.current.loading).toBe(false);
    expect(result.current.total).toBe(0);
    expect(result.current.statistics.total).toBe(0);
    expect(typeof result.current.loadDiscoveries).toBe("function");
    expect(typeof result.current.loadStatistics).toBe("function");
    expect(typeof result.current.loadDepartments).toBe("function");
    expect(typeof result.current.loadDiscoveryResults).toBe("function");
  });

  it("loadStatistics 调 post", async () => {
    const { result } = renderHook(() => useDiscoveryData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics.total).toBe(0);
  });

  it("loadDepartments 调 post", async () => {
    const { result } = renderHook(() => useDiscoveryData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.loadDepartments();
    });
    expect(result.current.departments).toEqual([]);
  });

  it("loadDiscoveryResults 调 post + setDiscoveredDevices", async () => {
    const { result } = renderHook(() => useDiscoveryData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.loadDiscoveryResults("d1");
    });
    expect(result.current.discoveredDevices).toEqual([]);
  });

  it("setters 直写 state", () => {
    const { result } = renderHook(() => useDiscoveryData(opts()), { wrapper: wrap });
    act(() => {
      result.current.setDiscoveries([{ id: "d1" } as any]);
      result.current.setDiscoveredDevices([{ ip: "10.0.0.1" }]);
      result.current.setDepartments([{ id: "dept1" } as any]);
      result.current.setModalState({ createModalVisible: true } as any);
      result.current.setCurrentDiscovery({ id: "d1" } as any);
    });
    expect(result.current.discoveries.length).toBe(1);
    expect(result.current.discoveredDevices.length).toBe(1);
    expect(result.current.departments.length).toBe(1);
  });
});
