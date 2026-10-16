/**
 * Phase 88 Batch259 — pages/system/notice/hooks/useAPIConfig 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockList = vi.fn();
vi.mock("@/lib/notificationConfigApi", () => ({
  getAPINotificationConfigList: (...args: any[]) => mockList(...args),
}));

import { useAPIConfig } from "../useAPIConfig";

describe("system/notice/hooks/useAPIConfig", () => {
  beforeEach(() => {
    mockList.mockReset();
  });

  it("初始 apiConfigs=[] + loading=false", () => {
    const { result } = renderHook(() => useAPIConfig());
    expect(result.current.apiConfigs).toEqual([]);
    expect(result.current.loadingAPIConfigs).toBe(false);
  });

  it("loadAPIConfigs 成功 → 设置 status=0", async () => {
    mockList.mockResolvedValue({
      data: {
        list: [
          { id: "1", name: "A", status: 0 },
          { id: "2", name: "B", status: 1 },
        ],
      },
    });
    const { result } = renderHook(() => useAPIConfig());
    await act(async () => {
      await result.current.loadAPIConfigs();
    });
    expect(result.current.apiConfigs.length).toBe(1);
    expect(result.current.apiConfigs[0].id).toBe("1");
    expect(result.current.loadingAPIConfigs).toBe(false);
  });

  it("loadAPIConfigs 失败 → 静默 + loading=false", async () => {
    mockList.mockRejectedValue(new Error("net"));
    const { result } = renderHook(() => useAPIConfig());
    await act(async () => {
      await result.current.loadAPIConfigs();
    });
    expect(result.current.apiConfigs).toEqual([]);
    expect(result.current.loadingAPIConfigs).toBe(false);
  });

  it("loadAPIConfigs 空 list", async () => {
    mockList.mockResolvedValue({ data: { list: [] } });
    const { result } = renderHook(() => useAPIConfig());
    await act(async () => {
      await result.current.loadAPIConfigs();
    });
    expect(result.current.apiConfigs).toEqual([]);
  });
});
