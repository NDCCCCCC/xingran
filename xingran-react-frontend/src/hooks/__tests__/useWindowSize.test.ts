/**
 * Phase 88 Batch216 — hooks/useWindowSize 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useWindowSize } from "../useWindowSize";

describe("hooks/useWindowSize", () => {
  beforeEach(() => {
    Object.defineProperty(window, "innerWidth", { value: 1024, writable: true });
    Object.defineProperty(window, "innerHeight", { value: 768, writable: true });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("返回初始尺寸", () => {
    const { result } = renderHook(() => useWindowSize());
    expect(result.current.width).toBe(1024);
    expect(result.current.height).toBe(768);
  });

  it("resize 触发更新", () => {
    const { result } = renderHook(() => useWindowSize());
    act(() => {
      Object.defineProperty(window, "innerWidth", { value: 1280, writable: true });
      Object.defineProperty(window, "innerHeight", { value: 800, writable: true });
      window.dispatchEvent(new Event("resize"));
    });
    expect(result.current.width).toBe(1280);
    expect(result.current.height).toBe(800);
  });

  it("卸载移除监听器", () => {
    const removeSpy = vi.spyOn(window, "removeEventListener");
    const { unmount } = renderHook(() => useWindowSize());
    unmount();
    expect(removeSpy).toHaveBeenCalledWith("resize", expect.any(Function));
  });
});
