/**
 * Phase 88 Batch63 — useDeviceData hook 测试(小 hook 快速覆盖)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
}));

import { useDeviceData } from "../hooks/useDeviceData";

beforeEach(() => {
  vi.clearAllMocks();
});

const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
const wrap = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={qc}>
    <ConfigProvider>
      <App>{children}</App>
    </ConfigProvider>
  </QueryClientProvider>
);

describe("useDeviceData", () => {
  it("initial state + 3 handler", () => {
    const { result } = renderHook(() => useDeviceData(), { wrapper: wrap });
    expect(result.current.departments).toEqual([]);
    expect(result.current.credentials).toEqual([]);
    expect(result.current.statistics).toEqual({ total: 0, online: 0, offline: 0, unknown: 0 });
    expect(typeof result.current.loadStatistics).toBe("function");
    expect(typeof result.current.loadCredentials).toBe("function");
    expect(typeof result.current.ensureCredential).toBe("function");
  });

  it("loadStatistics 调 post + setStatistics", async () => {
    const { result } = renderHook(() => useDeviceData(), { wrapper: wrap });
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics.total).toBe(0);
  });

  it("loadCredentials 调 post", async () => {
    const { result } = renderHook(() => useDeviceData(), { wrapper: wrap });
    await act(async () => {
      await result.current.loadCredentials();
    });
    expect(result.current.credentials).toEqual([]);
  });

  it("ensureCredential 追加新凭证(id 不存在)", () => {
    const { result } = renderHook(() => useDeviceData(), { wrapper: wrap });
    act(() => {
      result.current.ensureCredential({
        id: "c1",
        credentialName: "cred1",
      } as any);
    });
    expect(result.current.credentials.length).toBe(1);
  });

  it("ensureCredential 不重复(同 id 已在列表)", () => {
    const { result } = renderHook(() => useDeviceData(), { wrapper: wrap });
    act(() => {
      result.current.ensureCredential({ id: "c1", credentialName: "cred1" } as any);
    });
    act(() => {
      result.current.ensureCredential({ id: "c1", credentialName: "cred1-updated" } as any);
    });
    expect(result.current.credentials.length).toBe(1);
  });
});
