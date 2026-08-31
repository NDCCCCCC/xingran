/**
 * Phase 88 Batch368 — hooks/useWebSocket 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// Mock WebSocket
class MockWebSocket {
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSED = 3;
  readyState = MockWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((e: any) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: ((e: any) => void) | null = null;
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

import { useWebSocket } from "../useWebSocket";

describe("hooks/useWebSocket", () => {
  beforeEach(() => {
    MockWebSocket.reset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("返回初始状态 disconnected", () => {
    const { result } = renderHook(() => useWebSocket({ url: "ws://test" }));
    expect(result.current.status).toBe("disconnected");
    expect(result.current.reconnectAttempts).toBe(0);
  });

  it("connect → status=connecting + 创建 WebSocket", () => {
    const { result } = renderHook(() => useWebSocket({ url: "ws://test" }));
    act(() => result.current.connect());
    expect(result.current.status).toBe("connecting");
    expect(MockWebSocket.instances.length).toBe(1);
    expect(MockWebSocket.instances[0].url).toBe("ws://test");
  });

  it("WebSocket onopen → status=connected + onOpen callback", () => {
    const onOpen = vi.fn();
    const { result } = renderHook(() => useWebSocket({ url: "ws://test", onOpen }));
    act(() => result.current.connect());
    act(() => {
      MockWebSocket.instances[0].readyState = MockWebSocket.OPEN;
      MockWebSocket.instances[0].onopen?.();
    });
    expect(result.current.status).toBe("connected");
    expect(onOpen).toHaveBeenCalled();
  });

  it("WebSocket onmessage → 解析 JSON + onMessage", () => {
    const onMessage = vi.fn();
    const { result } = renderHook(() => useWebSocket({ url: "ws://test", onMessage }));
    act(() => result.current.connect());
    act(() => {
      MockWebSocket.instances[0].onmessage?.({ data: JSON.stringify({ value: 1 }) });
    });
    expect(onMessage).toHaveBeenCalledWith({ value: 1 });
  });

  it("WebSocket onmessage 非 JSON → 静默 (不抛错)", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const onMessage = vi.fn();
    const { result } = renderHook(() => useWebSocket({ url: "ws://test", onMessage }));
    act(() => result.current.connect());
    act(() => {
      MockWebSocket.instances[0].onmessage?.({ data: "not-json" });
    });
    expect(onMessage).not.toHaveBeenCalled();
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  it("WebSocket onerror → status=error", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useWebSocket({ url: "ws://test", onError }));
    act(() => result.current.connect());
    act(() => {
      MockWebSocket.instances[0].onerror?.(new Event("error"));
    });
    expect(result.current.status).toBe("error");
    expect(onError).toHaveBeenCalled();
  });

  it("WebSocket onclose + 自动重连 → reconnectAttempts 增加", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://test", reconnect: true, reconnectInterval: 100 })
    );
    act(() => result.current.connect());
    act(() => {
      MockWebSocket.instances[0].onclose?.();
    });
    expect(result.current.reconnectAttempts).toBe(1);
  });

  it("maxReconnectAttempts=0 → 不重连", () => {
    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { result } = renderHook(() =>
      useWebSocket({
        url: "ws://test",
        reconnect: true,
        reconnectInterval: 100,
        maxReconnectAttempts: 0,
      })
    );
    act(() => result.current.connect());
    act(() => {
      MockWebSocket.instances[0].onclose?.();
    });
    expect(result.current.reconnectAttempts).toBe(0);
    consoleSpy.mockRestore();
  });

  it("disconnect → 停止重连 + close WebSocket", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://test", reconnect: true, reconnectInterval: 100 })
    );
    act(() => result.current.connect());
    act(() => result.current.disconnect());
    expect(result.current.status).toBe("disconnected");
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled();
  });

  it("send: connected → 调用 ws.send", () => {
    const { result } = renderHook(() => useWebSocket({ url: "ws://test" }));
    act(() => result.current.connect());
    act(() => {
      MockWebSocket.instances[0].readyState = MockWebSocket.OPEN;
      MockWebSocket.instances[0].onopen?.();
    });
    act(() => result.current.send({ foo: "bar" }));
    expect(MockWebSocket.instances[0].send).toHaveBeenCalledWith(JSON.stringify({ foo: "bar" }));
  });

  it("send: not connected → 警告 + 不发送", () => {
    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { result } = renderHook(() => useWebSocket({ url: "ws://test" }));
    act(() => result.current.send({ x: 1 }));
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  it("无 options → 默认值", () => {
    const { result } = renderHook(() => useWebSocket());
    expect(result.current.status).toBe("disconnected");
  });

  it("卸载时清理", () => {
    const { result, unmount } = renderHook(() => useWebSocket({ url: "ws://test" }));
    act(() => result.current.connect());
    unmount();
    // WebSocket close should have been called on unmount
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled();
  });

  it("connect 已在连接 → 不重复", () => {
    const { result } = renderHook(() => useWebSocket({ url: "ws://test" }));
    act(() => result.current.connect());
    act(() => result.current.connect());
    expect(MockWebSocket.instances.length).toBe(1);
  });
});
