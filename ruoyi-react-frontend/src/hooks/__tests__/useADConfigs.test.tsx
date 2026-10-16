/**
 * Phase 88 Batch309 — hooks/useADConfigs 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", () => ({
  getADConfigList: vi.fn(async () => ({
    code: 0,
    data: { list: [{ id: "c1", name: "AD-1" }], total: 1 },
  })),
}));

import { getADConfigList } from "@/lib/adDomainApi";
import { useADConfigs } from "../useADConfigs";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("hooks/useADConfigs", () => {
  beforeEach(() => {
    vi.mocked(getADConfigList).mockReset();
    vi.mocked(getADConfigList).mockResolvedValue({
      code: 0,
      data: { list: [{ id: "c1", name: "AD-1" }], total: 1 },
    } as any);
  });

  it("初始返回 configs=[] + selectedConfig=''", () => {
    const { result } = renderHook(() => useADConfigs(), { wrapper });
    expect(result.current.configs).toEqual([]);
    expect(result.current.selectedConfig).toBe("");
  });

  it("挂载后自动加载 → 自动选第一个", async () => {
    const { result } = renderHook(() => useADConfigs(), { wrapper });
    await waitFor(() => {
      expect(result.current.configs.length).toBe(1);
    });
    expect(result.current.selectedConfig).toBe("c1");
  });

  it("autoSelectFirst=false 不自动选", async () => {
    vi.mocked(getADConfigList).mockResolvedValue({
      code: 0,
      data: { list: [{ id: "c2", name: "AD-2" }], total: 1 },
    } as any);
    const { result } = renderHook(() => useADConfigs({ autoSelectFirst: false }), { wrapper });
    await waitFor(() => {
      expect(result.current.configs.length).toBe(1);
    });
    expect(result.current.selectedConfig).toBe("");
  });

  it("setSelectedConfig 更新", async () => {
    const { result } = renderHook(() => useADConfigs(), { wrapper });
    await waitFor(() => expect(result.current.configs.length).toBe(1));
    act(() => {
      result.current.setSelectedConfig("x");
    });
    expect(result.current.selectedConfig).toBe("x");
  });

  it("clearSelection 清空", async () => {
    const { result } = renderHook(() => useADConfigs(), { wrapper });
    await waitFor(() => expect(result.current.selectedConfig).toBe("c1"));
    act(() => {
      result.current.clearSelection();
    });
    expect(result.current.selectedConfig).toBe("");
  });

  it("refreshConfigs 调用 getADConfigList", async () => {
    const { result } = renderHook(() => useADConfigs(), { wrapper });
    await waitFor(() => expect(result.current.configs.length).toBe(1));
    vi.mocked(getADConfigList).mockClear();
    act(() => {
      result.current.refreshConfigs();
    });
    await waitFor(() => expect(getADConfigList).toHaveBeenCalled());
  });

  it("enabledOnly=false 不传 status", async () => {
    renderHook(() => useADConfigs({ enabledOnly: false }), { wrapper });
    await waitFor(() => expect(getADConfigList).toHaveBeenCalled());
    const call =
      vi.mocked(getADConfigList).mock.calls[vi.mocked(getADConfigList).mock.calls.length - 1];
    expect(call[0]).not.toHaveProperty("status");
  });

  it("fetchConfigs 接受 sort 参数", async () => {
    const { result } = renderHook(() => useADConfigs(), { wrapper });
    await waitFor(() => expect(result.current.configs.length).toBe(1));
    vi.mocked(getADConfigList).mockClear();
    await act(async () => {
      await result.current.fetchConfigs("name", true);
    });
    expect(getADConfigList).toHaveBeenCalled();
    const call =
      vi.mocked(getADConfigList).mock.calls[vi.mocked(getADConfigList).mock.calls.length - 1];
    expect(call[0]).toMatchObject({ orderByColumn: "name", isAsc: true });
  });

  it("error 不抛错 (try/catch)", async () => {
    vi.mocked(getADConfigList).mockRejectedValue(new Error("net"));
    const { result } = renderHook(() => useADConfigs(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.configs).toEqual([]);
  });
});
