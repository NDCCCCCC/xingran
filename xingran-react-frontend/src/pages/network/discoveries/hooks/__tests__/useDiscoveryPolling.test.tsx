/**
 * Phase 88 Batch278 — pages/network/discoveries/hooks/useDiscoveryPolling 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useDiscoveryPolling } from "../useDiscoveryPolling";

describe("network/discoveries/hooks/useDiscoveryPolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("无 running 任务 → 不启动 polling", () => {
    const onPoll = vi.fn();
    renderHook(() => useDiscoveryPolling({ discoveries: [], onPoll }));
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(onPoll).not.toHaveBeenCalled();
  });

  it("有 running 任务 → 启动 polling 并 onPoll", () => {
    const onPoll = vi.fn();
    const discoveries: any[] = [
      { id: "d1", status: "running" },
      { id: "d2", status: "completed" },
    ];
    renderHook(() => useDiscoveryPolling({ discoveries, onPoll }));
    act(() => {
      vi.advanceTimersByTime(3000);
    });
    expect(onPoll).toHaveBeenCalled();
  });

  it("running 任务完成 → 停止 polling", () => {
    const onPoll = vi.fn();
    const discoveries: any[] = [{ id: "d1", status: "running" }];
    const { rerender } = renderHook(
      ({ d }: any) => useDiscoveryPolling({ discoveries: d, onPoll }),
      { wrapper: ({ children }) => <>{children}</>, initialProps: { d: discoveries } }
    );
    act(() => {
      vi.advanceTimersByTime(3000);
    });
    expect(onPoll).toHaveBeenCalledTimes(1);

    // running 完成后清空
    rerender({ d: [{ id: "d1", status: "completed" }] });
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    // 停止后不再调用
    expect(onPoll).toHaveBeenCalledTimes(1);
  });

  it("卸载 → clearInterval", () => {
    const onPoll = vi.fn();
    const discoveries: any[] = [{ id: "d1", status: "running" }];
    const { unmount } = renderHook(() => useDiscoveryPolling({ discoveries, onPoll }));
    unmount();
    // clearInterval 已调用,后续不报错
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(onPoll).not.toHaveBeenCalled();
  });
});
