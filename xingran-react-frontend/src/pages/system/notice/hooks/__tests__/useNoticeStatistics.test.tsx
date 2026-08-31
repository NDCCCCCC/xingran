/**
 * Phase 88 Batch280 — pages/system/notice/hooks/useNoticeStatistics 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockStatistics = vi.fn();
vi.mock("@/lib/noticeApi", () => ({
  getNoticeStatusStatistics: (...args: any[]) => mockStatistics(...args),
}));

import { useNoticeStatistics } from "../useNoticeStatistics";

describe("system/notice/hooks/useNoticeStatistics", () => {
  beforeEach(() => {
    mockStatistics.mockReset();
  });

  it("初始 statistics 全 0", () => {
    const { result } = renderHook(() => useNoticeStatistics());
    expect(result.current.statistics).toEqual({
      total: 0,
      published: 0,
      draft: 0,
      scheduled: 0,
    });
  });

  it("loadStatistics 成功 → 设置 stats", async () => {
    mockStatistics.mockResolvedValue({
      data: {
        total: 100,
        published: 70,
        draft: 20,
        scheduled: 10,
      },
    });
    const { result } = renderHook(() => useNoticeStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics).toEqual({
      total: 100,
      published: 70,
      draft: 20,
      scheduled: 10,
    });
  });

  it("loadStatistics 失败 → 静默 + 默认值", async () => {
    mockStatistics.mockRejectedValue(new Error("net"));
    const { result } = renderHook(() => useNoticeStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics).toEqual({
      total: 0,
      published: 0,
      draft: 0,
      scheduled: 0,
    });
  });

  it("loadStatistics 部分缺失 → fallback 0", async () => {
    mockStatistics.mockResolvedValue({ data: { total: 10 } });
    const { result } = renderHook(() => useNoticeStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    expect(result.current.statistics.total).toBe(10);
    expect(result.current.statistics.published).toBe(0);
  });
});
