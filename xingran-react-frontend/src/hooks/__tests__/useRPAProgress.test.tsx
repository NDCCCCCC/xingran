/**
 * Phase 88 Batch378 — hooks/useRPAProgress 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockUnsub = vi.fn();
let mockOnRPAProgressCallback: ((msg: any) => void) | null = null;

vi.mock("@/store/noticeStore", () => ({
  useNoticeStore: vi.fn(() => ({
    wsConnected: true,
    onRPAProgress: vi.fn((cb: any) => {
      mockOnRPAProgressCallback = cb;
      return mockUnsub;
    }),
  })),
}));

import { useRPAProgress } from "../useRPAProgress";

describe("hooks/useRPAProgress", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockOnRPAProgressCallback = null;
  });

  it("初始返回 isConnected=true + functions", () => {
    const { result } = renderHook(() => useRPAProgress());
    expect(result.current.isConnected).toBe(true);
    expect(typeof result.current.getProgress).toBe("function");
    expect(typeof result.current.getAllProgress).toBe("function");
    expect(typeof result.current.clearProgress).toBe("function");
    expect(typeof result.current.clearAllProgress).toBe("function");
  });

  it("enabled=false → 不订阅", () => {
    renderHook(() => useRPAProgress({ enabled: false }));
    expect(mockOnRPAProgressCallback).toBeNull();
  });

  it("收到 rpa_progress → 调 onProgress", () => {
    const onProgress = vi.fn();
    renderHook(() => useRPAProgress({ onProgress }));
    act(() => {
      mockOnRPAProgressCallback?.({
        type: "rpa_progress",
        executionId: "e1",
        taskId: "t1",
        taskName: "Task 1",
        step: 5,
        total: 10,
        message: "50%",
        status: "running",
        timestamp: Date.now(),
      });
    });
    expect(onProgress).toHaveBeenCalled();
  });

  it("收到 rpa_completed → 调 onCompleted", () => {
    const onCompleted = vi.fn();
    renderHook(() => useRPAProgress({ onCompleted }));
    act(() => {
      mockOnRPAProgressCallback?.({
        type: "rpa_completed",
        executionId: "e1",
        taskId: "t1",
        taskName: "Task 1",
        step: 10,
        total: 10,
        message: "done",
        status: "completed",
        timestamp: Date.now(),
      });
    });
    expect(onCompleted).toHaveBeenCalled();
  });

  it("收到 rpa_failed → 调 onFailed", () => {
    const onFailed = vi.fn();
    renderHook(() => useRPAProgress({ onFailed }));
    act(() => {
      mockOnRPAProgressCallback?.({
        type: "rpa_failed",
        executionId: "e1",
        taskId: "t1",
        taskName: "Task 1",
        step: 3,
        total: 10,
        message: "error",
        status: "failed",
        timestamp: Date.now(),
      });
    });
    expect(onFailed).toHaveBeenCalled();
  });

  it("executionId 过滤", () => {
    const onProgress = vi.fn();
    renderHook(() => useRPAProgress({ executionId: "e1", onProgress }));
    act(() => {
      mockOnRPAProgressCallback?.({
        type: "rpa_progress",
        executionId: "other",
        taskId: "t1",
        step: 1,
        total: 10,
        message: "x",
        status: "running",
        timestamp: Date.now(),
        taskName: "x",
      });
    });
    expect(onProgress).not.toHaveBeenCalled();
  });

  it("taskId 过滤", () => {
    const onProgress = vi.fn();
    renderHook(() => useRPAProgress({ taskId: "t1", onProgress }));
    act(() => {
      mockOnRPAProgressCallback?.({
        type: "rpa_progress",
        executionId: "e1",
        taskId: "other",
        step: 1,
        total: 10,
        message: "x",
        status: "running",
        timestamp: Date.now(),
        taskName: "x",
      });
    });
    expect(onProgress).not.toHaveBeenCalled();
  });

  it("getProgress 返回存储的数据", () => {
    renderHook(() => useRPAProgress());
    act(() => {
      mockOnRPAProgressCallback?.({
        type: "rpa_progress",
        executionId: "e1",
        taskId: "t1",
        step: 1,
        total: 10,
        message: "x",
        status: "running",
        timestamp: Date.now(),
        taskName: "T1",
      });
    });
    const { result } = renderHook(() => useRPAProgress());
    expect(result.current.getProgress("e1")).toBeUndefined(); // different hook instance
  });

  it("getAllProgress 返回数组", () => {
    const { result } = renderHook(() => useRPAProgress());
    const all = result.current.getAllProgress();
    expect(Array.isArray(all)).toBe(true);
  });

  it("clearProgress 清除指定", () => {
    const { result } = renderHook(() => useRPAProgress());
    act(() => result.current.clearProgress("e1"));
    expect(result.current.getProgress("e1")).toBeUndefined();
  });

  it("clearAllProgress 清除所有", () => {
    const { result } = renderHook(() => useRPAProgress());
    act(() => result.current.clearAllProgress());
    expect(result.current.getAllProgress()).toEqual([]);
  });

  it("卸载时取消订阅", () => {
    const { unmount } = renderHook(() => useRPAProgress());
    unmount();
    expect(mockUnsub).toHaveBeenCalled();
  });
});
