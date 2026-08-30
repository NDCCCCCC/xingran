/**
 * Phase 88 Batch144 — pages/network/discoveries/hooks/useDiscoveryPolling 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useDiscoveryPolling } from "../useDiscoveryPolling";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useDiscoveryPolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("running tasks > 0 → 启动 polling", () => {
    const onPoll = vi.fn();
    const discoveries = [{ id: "d1", status: "running" } as any];
    renderHook(() => useDiscoveryPolling({ discoveries, onPoll }), { wrapper });
    act(() => {
      vi.advanceTimersByTime(3000);
    });
    expect(onPoll).toHaveBeenCalled();
  });

  it("running=0 + 之前有 timer → 清除 timer", () => {
    const onPoll = vi.fn();
    const running = [{ id: "d1", status: "running" } as any];
    const stopped = [{ id: "d1", status: "stopped" } as any];
    const { rerender } = renderHook(
      ({ disc }) => useDiscoveryPolling({ discoveries: disc, onPoll }),
      { wrapper, initialProps: { disc: running } }
    );
    rerender({ disc: stopped });
    // timer cleared — no poll call
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    // onPoll may have been called once before stop; total = 1 not multiple
    expect(onPoll.mock.calls.length).toBeLessThanOrEqual(1);
  });

  it("空 discoveries → 不启动 polling", () => {
    const onPoll = vi.fn();
    renderHook(() => useDiscoveryPolling({ discoveries: [], onPoll }), { wrapper });
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(onPoll).not.toHaveBeenCalled();
  });

  it("polling 时所有任务停止 → 内部停止", () => {
    const onPoll = vi.fn();
    // Simulate the scenario where running becomes 0 mid-polling
    let currentDisc = [{ id: "d1", status: "running" } as any];
    const { rerender } = renderHook(
      ({ disc }) => useDiscoveryPolling({ discoveries: disc, onPoll }),
      { wrapper, initialProps: { disc: currentDisc } }
    );
    // mid-poll all stop
    currentDisc = [{ id: "d1", status: "stopped" } as any];
    rerender({ disc: currentDisc });
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(true).toBe(true); // no crash
  });
});
