/**
 * Phase 88 Batch387 — hooks/useWebSocket 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// Mock WebSocket class
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  readyState = 1;
  url: string;
  onopen: ((e: any) => void) | null = null;
  onclose: ((e: any) => void) | null = null;
  onerror: ((e: any) => void) | null = null;
  onmessage: ((e: any) => void) | null = null;
  static instances: MockWebSocket[] = [];
  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }
  send = vi.fn();
  close = vi.fn();
}
(globalThis as any).WebSocket = MockWebSocket;

vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn(() => ({
    preferences: { data: { theme: { mode: "light" } } },
  })),
}));

import { useWebSocket } from "../useWebSocket";

describe("hooks/useWebSocket", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    MockWebSocket.instances = [];
  });

  it("返回状态字段", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://localhost:8080/ws", onMessage: vi.fn() })
    );
    expect(typeof result.current.status).toBe("string");
    expect(typeof result.current.connect).toBe("function");
    expect(typeof result.current.disconnect).toBe("function");
    expect(typeof result.current.send).toBe("function");
    expect(typeof result.current.reconnectAttempts).toBe("number");
  });

  it("send 是函数", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://localhost:8080/ws", onMessage: vi.fn() })
    );
    expect(typeof result.current.send).toBe("function");
  });

  it("disconnect 是函数", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://localhost:8080/ws", onMessage: vi.fn() })
    );
    expect(typeof result.current.disconnect).toBe("function");
  });

  it("无 url 时不抛错", () => {
    const { result } = renderHook(() => useWebSocket({ onMessage: vi.fn() }));
    expect(typeof result.current.status).toBe("string");
  });

  it("connect 是函数", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://localhost:8080/ws", onMessage: vi.fn() })
    );
    expect(typeof result.current.connect).toBe("function");
  });
});
