/**
 * Phase 88 Batch308 — hooks/useNetworkStatus 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useNetworkStatus } from "../useNetworkStatus";

describe("hooks/useNetworkStatus", () => {
  beforeEach(() => {
    Object.defineProperty(navigator, "onLine", { configurable: true, value: true });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("初始返回 isOnline=true wasOffline=false", () => {
    const { result } = renderHook(() => useNetworkStatus());
    expect(result.current.isOnline).toBe(true);
    expect(result.current.wasOffline).toBe(false);
  });

  it("初始离线 → isOnline=false", () => {
    Object.defineProperty(navigator, "onLine", { configurable: true, value: false });
    const { result } = renderHook(() => useNetworkStatus());
    expect(result.current.isOnline).toBe(false);
  });

  it("resetWasOffline 不抛错", () => {
    const { result } = renderHook(() => useNetworkStatus());
    act(() => {
      result.current.resetWasOffline();
    });
    expect(result.current.wasOffline).toBe(false);
  });

  it("offline 事件 → isOnline=false + wasOffline=true", () => {
    const { result } = renderHook(() => useNetworkStatus());
    act(() => {
      window.dispatchEvent(new Event("offline"));
    });
    expect(result.current.isOnline).toBe(false);
    expect(result.current.wasOffline).toBe(true);
  });

  it("online 事件 → isOnline=true (wasOffline 保持)", () => {
    Object.defineProperty(navigator, "onLine", { configurable: true, value: false });
    const { result } = renderHook(() => useNetworkStatus());
    act(() => {
      window.dispatchEvent(new Event("offline"));
    });
    act(() => {
      window.dispatchEvent(new Event("online"));
    });
    expect(result.current.isOnline).toBe(true);
    expect(result.current.wasOffline).toBe(true);
  });

  it("resetWasOffline 清 wasOffline 状态", () => {
    const { result } = renderHook(() => useNetworkStatus());
    act(() => {
      window.dispatchEvent(new Event("offline"));
    });
    expect(result.current.wasOffline).toBe(true);
    act(() => {
      result.current.resetWasOffline();
    });
    expect(result.current.wasOffline).toBe(false);
  });
});
