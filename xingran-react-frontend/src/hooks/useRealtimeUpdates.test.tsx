/**
 * useRealtimeUpdates 测试
 *
 * 覆盖：连接建立与 URL 派生 / onopen 订阅频道 / onmessage 写缓存 /
 * onerror 回调 / onclose 指数退避重连 / 重连次数上限后停止 /
 * enabled=false 短路 / 无 WebSocket 数据源时不连接。
 *
 * 背景：后端当前只注册了 /api/v1/system/ws/notices，仪表盘频道端点尚未提供，
 * 因此重连策略（退避 + 次数上限）是这里的关键行为 —— 旧实现会无限重连。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import type { WidgetConfig } from "@/types/dashboard";

const cacheWidgetData = vi.hoisted(() => vi.fn());

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: () => ({ cacheWidgetData }),
}));

import { useRealtimeUpdates } from "./useRealtimeUpdates";

class MockWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 3;
  static instances: MockWebSocket[] = [];

  readyState = MockWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  onerror: ((e: Event) => void) | null = null;
  onclose: (() => void) | null = null;
  sent: string[] = [];

  constructor(public url: string) {
    MockWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
  }

  triggerOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.();
  }

  triggerMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }

  triggerError() {
    this.onerror?.(new Event("error"));
  }

  triggerClose() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  }
}

function makeWidget(overrides: Partial<WidgetConfig> = {}): WidgetConfig {
  return {
    id: "w1",
    type: "stat-card",
    title: "测试 Widget",
    position: { x: 0, y: 0, w: 4, h: 3 },
    dataSource: { type: "websocket", channel: "ops-1" },
    display: { type: "stat-card" },
    enabled: true,
    refreshInterval: 0,
    ...overrides,
  } as WidgetConfig;
}

// 模块级常量 —— connect 的依赖是 [widgets, options, ...]，引用必须稳定，
// 否则 effect 会反复重跑（disconnect + connect）。
const WS_WIDGETS = [makeWidget()];
const STATIC_WIDGETS = [
  makeWidget({ dataSource: { type: "static", data: { value: 1 } } as never }),
];

describe("useRealtimeUpdates", () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    cacheWidgetData.mockClear();
    vi.stubGlobal("WebSocket", MockWebSocket);
    vi.spyOn(console, "error").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("建立连接，URL 由 API 基址派生（带 /system/ws/dashboard 路径）", () => {
    renderHook(() => useRealtimeUpdates(WS_WIDGETS));

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toContain("/system/ws/dashboard");
  });

  it("连接建立后订阅 WebSocket 数据源的频道", () => {
    renderHook(() => useRealtimeUpdates(WS_WIDGETS));

    const ws = MockWebSocket.instances[0];
    act(() => ws.triggerOpen());

    expect(ws.sent).toHaveLength(1);
    expect(JSON.parse(ws.sent[0])).toEqual({ action: "subscribe", channel: "ops-1" });
  });

  it("收到 data_update 消息时写缓存并回调", () => {
    const onMessage = vi.fn();
    const options = { onMessage };
    renderHook(() => useRealtimeUpdates(WS_WIDGETS, options));

    const ws = MockWebSocket.instances[0];
    act(() => ws.triggerOpen());
    act(() => ws.triggerMessage({ type: "data_update", widgetId: "w1", data: { value: 7 } }));

    expect(cacheWidgetData).toHaveBeenCalledWith("w1", { value: 7 });
    expect(onMessage).toHaveBeenCalledWith("w1", { value: 7 });
  });

  it("连接错误时触发 onError 与连接状态回调", () => {
    const onError = vi.fn();
    const onConnectionChange = vi.fn();
    const options = { onError, onConnectionChange };
    renderHook(() => useRealtimeUpdates(WS_WIDGETS, options));

    const ws = MockWebSocket.instances[0];
    act(() => ws.triggerError());

    expect(onError).toHaveBeenCalled();
    expect(onConnectionChange).toHaveBeenCalledWith(false);
  });

  it("连接关闭后按指数退避重连", () => {
    vi.useFakeTimers();
    renderHook(() => useRealtimeUpdates(WS_WIDGETS));

    const first = MockWebSocket.instances[0];
    act(() => first.triggerClose());
    expect(MockWebSocket.instances).toHaveLength(1);

    // 首个退避为 5s
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it("重连次数达到上限后停止,不再新建连接", () => {
    vi.useFakeTimers();
    renderHook(() => useRealtimeUpdates(WS_WIDGETS));

    // 每次关闭触发一次退避重连，累计到上限
    for (let i = 0; i < 6; i++) {
      const ws = MockWebSocket.instances[MockWebSocket.instances.length - 1];
      act(() => ws.triggerClose());
      act(() => {
        vi.advanceTimersByTime(60000);
      });
    }

    expect(MockWebSocket.instances.length).toBeLessThanOrEqual(6);
    expect(console.warn).toHaveBeenCalledWith(expect.stringContaining("停止重连"));
  });

  it("enabled=false 时不建立连接", () => {
    const options = { enabled: false };
    renderHook(() => useRealtimeUpdates(WS_WIDGETS, options));

    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("没有 WebSocket 数据源时不建立连接", () => {
    renderHook(() => useRealtimeUpdates(STATIC_WIDGETS));

    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("手动 disconnect 后关闭当前连接", () => {
    const { result } = renderHook(() => useRealtimeUpdates(WS_WIDGETS));

    const ws = MockWebSocket.instances[0];
    act(() => ws.triggerOpen());
    act(() => result.current.disconnect());

    expect(ws.readyState).toBe(MockWebSocket.CLOSED);
  });
});
