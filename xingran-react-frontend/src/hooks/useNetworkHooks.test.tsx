/**
 * 网络类 hooks 组合测试
 *
 * 覆盖:useNetworkStatus / useWebSocket。
 * WebSocket 用 FakeWebSocket stub(vi.stubGlobal)精确控制 open/message/close 事件。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useNetworkStatus } from "./useNetworkStatus";
import { useWebSocket } from "./useWebSocket";

/** 可控的 WebSocket 假实现:静态常量对齐真实 WebSocket */
class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;

  readyState = FakeWebSocket.CONNECTING;
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  sent: string[] = [];

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.();
  }

  // ---- 测试驱动辅助 ----
  simulateOpen() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }

  simulateMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }

  simulateRaw(raw: string) {
    this.onmessage?.({ data: raw });
  }

  simulateClose() {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.();
  }
}

describe("useNetworkStatus", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("初始读取 navigator.onLine", () => {
    vi.spyOn(navigator, "onLine", "get").mockReturnValue(true);
    const { result } = renderHook(() => useNetworkStatus());
    expect(result.current.isOnline).toBe(true);
    expect(result.current.wasOffline).toBe(false);
  });

  it("offline 事件 → 离线 + wasOffline 置位;online 恢复;resetWasOffline 复位", () => {
    const { result } = renderHook(() => useNetworkStatus());

    act(() => {
      window.dispatchEvent(new Event("offline"));
    });
    expect(result.current.isOnline).toBe(false);
    expect(result.current.wasOffline).toBe(true);

    act(() => {
      window.dispatchEvent(new Event("online"));
    });
    expect(result.current.isOnline).toBe(true);
    expect(result.current.wasOffline).toBe(true); // 保留用于恢复提示

    act(() => result.current.resetWasOffline());
    expect(result.current.wasOffline).toBe(false);
  });
});

describe("useWebSocket", () => {
  beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    vi.spyOn(console, "error").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  function renderWs(overrides: Partial<Parameters<typeof useWebSocket>[0]> = {}) {
    const onMessage = vi.fn();
    const onOpen = vi.fn();
    const onClose = vi.fn();
    const onError = vi.fn();
    const { result } = renderHook(() =>
      useWebSocket({
        url: "ws://test/echo",
        onMessage,
        onOpen,
        onClose,
        onError,
        reconnectInterval: 1000,
        maxReconnectAttempts: 10,
        ...overrides,
      })
    );
    return { result, onMessage, onOpen, onClose, onError };
  }

  it("connect 建立 → onopen 置 connected 并复位重连计数", () => {
    const { result, onOpen } = renderWs();

    act(() => result.current.connect());
    expect(result.current.status).toBe("connecting");
    expect(FakeWebSocket.instances).toHaveLength(1);

    const ws = FakeWebSocket.instances[0];
    act(() => ws.simulateOpen());
    expect(result.current.status).toBe("connected");
    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(result.current.reconnectAttempts).toBe(0);
  });

  it("onmessage JSON 解析后回调 onMessage;非法 JSON 走 error 分支", () => {
    const { result, onMessage } = renderWs();
    act(() => result.current.connect());
    const ws = FakeWebSocket.instances[0];
    act(() => ws.simulateOpen());

    act(() => ws.simulateMessage({ hello: "world" }));
    expect(onMessage).toHaveBeenCalledWith({ hello: "world" });

    act(() => ws.simulateRaw("not-json{{{"));
    expect(onMessage).toHaveBeenCalledTimes(1); // 解析失败不再回调
  });

  it("send 仅在 OPEN 时发送,否则 warn", () => {
    const { result } = renderWs();
    // 未连接
    act(() => result.current.send({ a: 1 }));
    expect(FakeWebSocket.instances[0]?.sent ?? []).toHaveLength(0);

    act(() => result.current.connect());
    const ws = FakeWebSocket.instances[0];
    act(() => ws.simulateOpen());
    act(() => result.current.send({ a: 1 }));
    expect(ws.sent).toEqual([JSON.stringify({ a: 1 })]);
  });

  it("重复 connect 在 OPEN/CONNECTING 时短路", () => {
    const { result } = renderWs();
    act(() => result.current.connect());
    act(() => FakeWebSocket.instances[0].simulateOpen());
    act(() => result.current.connect());
    expect(FakeWebSocket.instances).toHaveLength(1);
  });

  it("断线自动重连:指数退避 + 重连计数", async () => {
    vi.useFakeTimers();
    const { result } = renderWs({ reconnectInterval: 1000, maxReconnectAttempts: 3 });

    act(() => result.current.connect());
    expect(FakeWebSocket.instances).toHaveLength(1);
    const first = FakeWebSocket.instances[0];

    await act(async () => {
      first.simulateClose();
    });
    expect(result.current.status).toBe("disconnected");
    expect(result.current.reconnectAttempts).toBe(1);

    // 退避 delay = 1000 * 2^1 = 2000ms
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  it("达到最大重连次数后放弃", async () => {
    vi.useFakeTimers();
    const { result } = renderWs({ reconnectInterval: 100, maxReconnectAttempts: 2 });

    act(() => result.current.connect());

    // 第一次 close → 重连 1 次
    await act(async () => {
      FakeWebSocket.instances[0].simulateClose();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(FakeWebSocket.instances).toHaveLength(2);

    // 第二次 close → 达到上限不再重连
    await act(async () => {
      FakeWebSocket.instances[1].simulateClose();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  it("手动 disconnect 不触发自动重连并复位状态", () => {
    const { result, onClose } = renderWs();
    act(() => result.current.connect());
    const ws = FakeWebSocket.instances[0];
    act(() => ws.simulateOpen());

    act(() => result.current.disconnect());
    expect(result.current.status).toBe("disconnected");
    expect(result.current.reconnectAttempts).toBe(0);
    expect(onClose).toHaveBeenCalledTimes(1);

    // 之后的 timer 不再触发新连接
    expect(FakeWebSocket.instances).toHaveLength(1);
  });

  it("reconnect=false 时 close 不重连", async () => {
    const { result } = renderWs({ reconnect: false });
    act(() => result.current.connect());
    await act(async () => {
      FakeWebSocket.instances[0].simulateClose();
    });
    expect(FakeWebSocket.instances).toHaveLength(1);
  });

  it("onerror 置 error 状态并回调", () => {
    const { result, onError } = renderWs();
    act(() => result.current.connect());
    const ws = FakeWebSocket.instances[0];

    act(() => {
      ws.onerror?.(new Error("socket err"));
    });
    expect(result.current.status).toBe("error");
    expect(onError).toHaveBeenCalledTimes(1);
  });
});
