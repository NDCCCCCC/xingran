/**
 * Phase 86 — notice hooks 测试(mock noticeApi)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useNoticeStatistics } from "../hooks/useNoticeStatistics";

vi.mock("@/lib/noticeApi", () => ({
  getNoticeStatusStatistics: vi.fn(() =>
    Promise.resolve({ data: { total: 20, published: 12, draft: 6, scheduled: 2 } })
  ),
}));

describe("useNoticeStatistics", () => {
  beforeEach(() => vi.clearAllMocks());

  it("initializes with zero statistics", () => {
    const { result } = renderHook(() => useNoticeStatistics());
    expect(result.current.statistics).toEqual({
      total: 0,
      published: 0,
      draft: 0,
      scheduled: 0,
    });
  });

  it("loadStatistics sets stats from API", async () => {
    const { result } = renderHook(() => useNoticeStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    await waitFor(() => {
      expect(result.current.statistics.total).toBe(20);
      expect(result.current.statistics.published).toBe(12);
    });
  });

  it("handles API rejection gracefully", async () => {
    const { getNoticeStatusStatistics } = await import("@/lib/noticeApi");
    (getNoticeStatusStatistics as any).mockRejectedValueOnce(new Error("network"));
    const { result } = renderHook(() => useNoticeStatistics());
    await act(async () => {
      await result.current.loadStatistics();
    });
    // 失败时保持初始值不崩溃
    expect(result.current.statistics.total).toBe(0);
  });
});
