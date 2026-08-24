/**
 * 网络类 hooks 组合测试
 *
 * 覆盖:useNetworkStatus / useRealtimeUpdates / useRPAProgress / useWebSocket。
 * WebSocket 用 FakeWebSocket stub(vi.stubGlobal)精确控制 open/message/close 事件。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useNetworkStatus } from "./useNetworkStatus";
import { useRealtimeUpdates } from "./useRealtimeUpdates";
import { useRPAProgress } from "./useRPAProgress";
import { useWebSocket } from "./useWebSocket";
import { useDashboardStore } from "@/store/dashboardStore";
import { useNoticeStore } from "@/store/noticeStore";
import type { WidgetConfig } from "@/types/dashboard";

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

function makeWsWidget(id: string, channel: string): WidgetConfig {
  return {
    id,
    type: "stat-card",
    title: id,
    position: { x: 0, y: 0, w: 4, h: 3 },
    dataSource: { type: "websocket", channel },
    display: { type: "stat-card" },
    enabled: true,
    refreshInterval: 0,
  } as WidgetConfig;
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

describe("useRealtimeUpdates", () => {
  beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    vi.spyOn(console, "error").mockImplementation(() => {});
    useDashboardStore.setState({ widgetDataCache: new Map() });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    useDashboardStore.setState({ widgetDataCache: new Map() });
  });

  it("有 websocket 数据源时自动连接,open 后按 channel 订阅", () => {
    const widgets = [
      makeWsWidget("w1", "ch-a"),
      makeWsWidget("w2", "ch-a"),
      makeWsWidget("w3", "ch-b"),
    ];
    const { result, rerender } = renderHook(() => useRealtimeUpdates(widgets));

    expect(FakeWebSocket.instances).toHaveLength(1);
    const ws = FakeWebSocket.instances[0];
    expect(ws.url).toContain("ws://");
    expect(ws.url).toContain("/ws/dashboard");

    act(() => ws.simulateOpen());
    // 同 channel 去重:ch-a 订阅一次 + ch-b 一次
    expect(ws.sent).toHaveLength(2);
    expect(ws.sent[0]).toBe(JSON.stringify({ action: "subscribe", channel: "ch-a" }));
    expect(ws.sent[1]).toBe(JSON.stringify({ action: "subscribe", channel: "ch-b" }));
    // connected 是渲染期快照,手动 rerender 后读取最新 ref
    rerender();
    expect(result.current.connected).toBe(true);
  });

  it("data_update 消息写缓存并触发 onMessage 回调", () => {
    useDashboardStore.getState().cacheWidgetData("seed", "old");
    const widgets = [makeWsWidget("w1", "ch-a")];
    const onMessage = vi.fn();
    const onConnectionChange = vi.fn();

    renderHook(() => useRealtimeUpdates(widgets, { onMessage, onConnectionChange }));
    const ws = FakeWebSocket.instances[0];
    act(() => ws.simulateOpen());

    act(() => ws.simulateMessage({ type: "data_update", widgetId: "w1", data: { v: 9 } }));
    expect(onMessage).toHaveBeenCalledWith("w1", { v: 9 });
    expect(useDashboardStore.getState().getCachedWidgetData("w1")).toEqual({ v: 9 });

    // 非 data_update / 非法 JSON 不触发回调
    act(() => ws.simulateRaw("not-json{{{"));
    expect(onMessage).toHaveBeenCalledTimes(1);
  });

  it("enabled=false 或无 websocket 数据源时不连接", () => {
    const apiWidget = {
      ...makeWsWidget("api", "x"),
      dataSource: { type: "static", data: 1 },
    } as unknown as WidgetConfig;

    renderHook(() => useRealtimeUpdates([apiWidget]));
    expect(FakeWebSocket.instances).toHaveLength(0);

    const { result: r2 } = renderHook(() =>
      useRealtimeUpdates([makeWsWidget("w", "c")], { enabled: false })
    );
    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(r2.current.connected).toBe(false);
  });

  it("onerror 触发 onError/onConnectionChange 回调", () => {
    const onError = vi.fn();
    const onConnectionChange = vi.fn();
    renderHook(() =>
      useRealtimeUpdates([makeWsWidget("w1", "ch")], { onError, onConnectionChange })
    );
    const ws = FakeWebSocket.instances[0];

    act(() => ws.simulateOpen());
    expect(onConnectionChange).toHaveBeenCalledWith(true);

    act(() => {
      ws.onerror?.(new Error("boom"));
    });
    expect(onError).toHaveBeenCalledTimes(1);
    expect(onConnectionChange).toHaveBeenCalledWith(false);
  });

  it("意外断线 5 秒后自动重连;主动 disconnect 不重连", async () => {
    vi.useFakeTimers();
    const widgets = [makeWsWidget("w1", "ch")];
    const { result, unmount } = renderHook(() => useRealtimeUpdates(widgets));

    const ws = FakeWebSocket.instances[0];
    act(() => ws.simulateOpen());
    await act(async () => {
      ws.simulateClose();
    });
    expect(FakeWebSocket.instances).toHaveLength(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(FakeWebSocket.instances).toHaveLength(2); // 自动重连

    // 主动断开(含卸载清理)不重连
    act(() => result.current.disconnect());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(FakeWebSocket.instances).toHaveLength(2);
    unmount();
  });

  it("refreshSubscriptions 在 OPEN 时对全部 ws 数据源重新订阅", () => {
    const widgets = [makeWsWidget("w1", "ch-a"), makeWsWidget("w2", "ch-b")];
    const { result } = renderHook(() => useRealtimeUpdates(widgets));
    const ws = FakeWebSocket.instances[0];
    act(() => ws.simulateOpen());
    expect(ws.sent).toHaveLength(2);

    act(() => result.current.refreshSubscriptions());
    expect(ws.sent).toHaveLength(4); // 重新订阅全部
  });
});

describe("useRPAProgress", () => {
  beforeEach(() => {
    useNoticeStore.setState({
      unreadCount: 0,
      notifications: [],
      loading: false,
      wsConnected: false,
    });
  });

  function makeMsg(overrides: Record<string, unknown> = {}) {
    return {
      type: "rpa_progress",
      executionId: "exec-1",
      taskId: "task-1",
      taskName: "任务",
      step: 1,
      total: 3,
      message: "running",
      status: "running",
      timestamp: 1,
      ...overrides,
    };
  }

  function pushMessage(msg: ReturnType<typeof makeMsg>) {
    useNoticeStore.getState().handleWsMessage({
      type: msg.type,
      content: JSON.stringify(msg),
    } as never);
  }

  it("订阅 rpa_progress 事件并记录进度,可按 executionId 查询", () => {
    const onProgress = vi.fn();
    const { result } = renderHook(() => useRPAProgress({ onProgress }));

    act(() => pushMessage(makeMsg()));
    expect(onProgress).toHaveBeenCalledTimes(1);
    expect(onProgress).toHaveBeenCalledWith(
      expect.objectContaining({ executionId: "exec-1", step: 1 })
    );
    expect(result.current.getProgress("exec-1")).toMatchObject({ step: 1 });
    expect(result.current.getAllProgress()).toHaveLength(1);
    expect(result.current.isConnected).toBe(false);
  });

  it("rpa_completed/rpa_failed 分别触发对应回调", () => {
    const onCompleted = vi.fn();
    const onFailed = vi.fn();
    renderHook(() => useRPAProgress({ onCompleted, onFailed }));

    act(() => pushMessage(makeMsg({ type: "rpa_completed", executionId: "e2" })));
    expect(onCompleted).toHaveBeenCalledTimes(1);

    act(() => pushMessage(makeMsg({ type: "rpa_failed", executionId: "e3" })));
    expect(onFailed).toHaveBeenCalledTimes(1);
  });

  it("executionId/taskId 过滤条件生效", () => {
    const onProgress = vi.fn();
    renderHook(() => useRPAProgress({ onProgress, executionId: "exec-1", taskId: "task-1" }));

    act(() => pushMessage(makeMsg({ executionId: "other" })));
    expect(onProgress).not.toHaveBeenCalled();

    act(() =>
      pushMessage(makeMsg({ executionId: "exec-1", taskId: "nope", type: "rpa_progress" }))
    );
    expect(onProgress).not.toHaveBeenCalled();
  });

  it("clearProgress/clearAllProgress 清理进度", () => {
    const { result } = renderHook(() => useRPAProgress());

    act(() => pushMessage(makeMsg({ executionId: "e1" })));
    act(() => pushMessage(makeMsg({ executionId: "e2" })));
    expect(result.current.getAllProgress()).toHaveLength(2);

    act(() => result.current.clearProgress("e1"));
    expect(result.current.getAllProgress()).toHaveLength(1);

    act(() => result.current.clearAllProgress());
    expect(result.current.getAllProgress()).toHaveLength(0);
  });

  it("enabled=false 不订阅", () => {
    const onProgress = vi.fn();
    const { result } = renderHook(() => useRPAProgress({ onProgress, enabled: false }));

    act(() => pushMessage(makeMsg()));
    expect(onProgress).not.toHaveBeenCalled();
    expect(result.current.getProgress("exec-1")).toBeUndefined();
  });
});
