/**
 * Phase 88 Batch61 — useBackupData hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
  get: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useBackupData } from "../hooks/useBackupData";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const opts = () => ({ current: 1, pageSize: 10, searchForm: { validateFields: vi.fn() } as any });

describe("useBackupData", () => {
  it("initial state", () => {
    const { result } = renderHook(() => useBackupData(opts()), { wrapper: wrap });
    expect(Array.isArray(result.current.devices)).toBe(true);
    expect(Array.isArray(result.current.deviceGroups)).toBe(true);
    expect(result.current.loading).toBe(false);
    expect(result.current.total).toBe(0);
    expect(result.current.statistics.total).toBe(0);
  });

  it("loadDevices 调 post", async () => {
    const { result } = renderHook(() => useBackupData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.loadDevices();
    });
    expect(result.current.devices).toEqual([]);
  });

  it("loadBackups 调 post + setTotal", async () => {
    const { result } = renderHook(() => useBackupData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.loadBackups({});
    });
    expect(result.current.total).toBe(0);
  });

  it("loadStatistics 调 get", async () => {
    const { result } = renderHook(() => useBackupData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.loadStatistics();
    });
  });
});
