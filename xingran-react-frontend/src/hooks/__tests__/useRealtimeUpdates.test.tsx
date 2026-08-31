/**
 * Phase 88 Batch375 — hooks/useRealtimeUpdates 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockCacheWidgetData = vi.fn();
vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({
    cacheWidgetData: mockCacheWidgetData,
  })),
}));

class MockWebSocket {
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSED = 3;
  readyState = MockWebSocket.CONNECTING;
  onopen: ((e?: any) => void) | null = null;
  onmessage: ((e: any) => void) | null = null;
  onclose: ((e?: any) => void) | null = null;
  onerror: ((e?: any) => void) | null = null;
  url = "";

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send = vi.fn();
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) this.onclose();
  });

  static instances: MockWebSocket[] = [];
  static reset() {
    MockWebSocket.instances = [];
  }
}

(globalThis as any).WebSocket = MockWebSocket;

import { useRealtimeUpdates } from "../useRealtimeUpdates";

describe("hooks/useRealtimeUpdates", () => {
  beforeEach(() => {
    MockWebSocket.reset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("无 ws widget → 不创建连接", () => {
    renderHook(() => useRealtimeUpdates([{ id: "w1", dataSource: { type: "api" } } as any]));
    expect(MockWebSocket.instances.length).toBe(0);
  });

  it("有 ws widget → 创建 WebSocket", () => {
    renderHook(() =>
      useRealtimeUpdates([
        {
          id: "w1",
          dataSource: { type: "websocket", channel: "chan1" },
        } as any,
      ])
    );
    expect(MockWebSocket.instances.length).toBe(1);
  });

  it("enabled=false → 不连接", () => {
    renderHook(() =>
      useRealtimeUpdates([{ id: "w1", dataSource: { type: "websocket", channel: "c1" } } as any], {
        enabled: false,
      })
    );
    expect(MockWebSocket.instances.length).toBe(0);
  });

  it("连接打开 → 发送 subscribe 消息", () => {
    renderHook(() =>
      useRealtimeUpdates([
        {
          id: "w1",
          dataSource: { type: "websocket", channel: "chan1" },
        } as any,
      ])
    );
    act(() => {
      MockWebSocket.instances[0].readyState = MockWebSocket.OPEN;
      MockWebSocket.instances[0].onopen?.();
    });
    expect(MockWebSocket.instances[0].send).toHaveBeenCalled();
  });

  it("收到消息 → 调 onMessage", () => {
    const onMessage = vi.fn();
    renderHook(() =>
      useRealtimeUpdates([{ id: "w1", dataSource: { type: "websocket", channel: "c1" } } as any], {
        onMessage,
      })
    );
    act(() => {
      MockWebSocket.instances[0].readyState = MockWebSocket.OPEN;
      MockWebSocket.instances[0].onopen?.();
    });
    act(() => {
      MockWebSocket.instances[0].onmessage?.({
        data: JSON.stringify({ type: "data_update", widgetId: "w1", data: { value: 100 } }),
      });
    });
    expect(onMessage).toHaveBeenCalledWith("w1", { value: 100 });
  });

  it("手动 disconnect → 关闭 WebSocket", () => {
    const { result } = renderHook(() =>
      useRealtimeUpdates([{ id: "w1", dataSource: { type: "websocket", channel: "c1" } } as any])
    );
    act(() => result.current.disconnect());
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled();
  });

  it("onclose (非主动断开) → 5s 后重连", () => {
    renderHook(() =>
      useRealtimeUpdates([{ id: "w1", dataSource: { type: "websocket", channel: "c1" } } as any])
    );
    act(() => {
      MockWebSocket.instances[0].onclose?.();
    });
    expect(MockWebSocket.instances.length).toBe(1);
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    // New ws instance
    expect(MockWebSocket.instances.length).toBeGreaterThan(1);
  });

  it("卸载时清理", () => {
    const { unmount } = renderHook(() =>
      useRealtimeUpdates([{ id: "w1", dataSource: { type: "websocket", channel: "c1" } } as any])
    );
    unmount();
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled();
  });

  it("包装格式 dataSource.websocket", () => {
    renderHook(() =>
      useRealtimeUpdates([{ id: "w1", dataSource: { websocket: { channel: "c1" } } } as any])
    );
    expect(MockWebSocket.instances.length).toBe(1);
  });

  it("refreshSubscriptions 不抛错", () => {
    const { result } = renderHook(() =>
      useRealtimeUpdates([{ id: "w1", dataSource: { type: "websocket", channel: "c1" } } as any])
    );
    act(() => result.current.refreshSubscriptions());
    expect(result.current.refreshSubscriptions).toBeDefined();
  });
});
