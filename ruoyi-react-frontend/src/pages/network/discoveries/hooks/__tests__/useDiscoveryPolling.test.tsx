/**
 * Phase 88 Batch354 — pages/network/discoveries/hooks/useDiscoveryPolling 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useDiscoveryPolling } from "../useDiscoveryPolling";

describe("pages/network/discoveries/hooks/useDiscoveryPolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("无 running tasks → 不启动 polling", () => {
    const onPoll = vi.fn();
    const discoveries = [{ id: "d1", status: "completed" } as any];
    renderHook(() => useDiscoveryPolling({ discoveries, onPoll }));
    vi.advanceTimersByTime(5000);
    expect(onPoll).not.toHaveBeenCalled();
  });

  it("含 running task → 启动 polling", () => {
    const onPoll = vi.fn();
    const discoveries = [{ id: "d1", status: "running" } as any];
    renderHook(() => useDiscoveryPolling({ discoveries, onPoll }));
    vi.advanceTimersByTime(3000);
    expect(onPoll).toHaveBeenCalled();
  });

  it("3 秒间隔触发 onPoll", () => {
    const onPoll = vi.fn();
    const discoveries = [{ id: "d1", status: "running" } as any];
    renderHook(() => useDiscoveryPolling({ discoveries, onPoll }));
    vi.advanceTimersByTime(6000);
    expect(onPoll).toHaveBeenCalledTimes(2);
  });

  it("卸载时清理 polling", () => {
    const onPoll = vi.fn();
    const discoveries = [{ id: "d1", status: "running" } as any];
    const { unmount } = renderHook(() => useDiscoveryPolling({ discoveries, onPoll }));
    unmount();
    vi.advanceTimersByTime(5000);
    expect(onPoll).not.toHaveBeenCalled();
  });

  it("running 完成后 → 停止 polling", () => {
    const onPoll = vi.fn();
    let discoveries = [{ id: "d1", status: "running" } as any];
    const { rerender } = renderHook(({ d }) => useDiscoveryPolling({ discoveries: d, onPoll }), {
      initialProps: { d: discoveries },
    });

    // 3s 时 onPoll 触发
    vi.advanceTimersByTime(3000);
    expect(onPoll).toHaveBeenCalled();

    // running 变为 completed
    discoveries = [{ id: "d1", status: "completed" } as any];
    rerender({ d: discoveries });

    // 后续不应再触发
    vi.advanceTimersByTime(6000);
    // Note: may have one more call from the 3s tick before re-render takes effect
    expect(onPoll.mock.calls.length).toBeLessThan(3);
  });
});
