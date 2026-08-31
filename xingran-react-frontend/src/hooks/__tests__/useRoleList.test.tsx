/**
 * Phase 88 Batch303 — hooks/useRoleList 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useRoleList, useInvalidateRole } from "../useRoleList";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("hooks/useRoleList", () => {
  it("useRoleList 返回 shape", () => {
    const { result } = renderHook(() => useRoleList(), { wrapper });
    expect(typeof result.current.isError).toBe("boolean");
    expect(typeof result.current.isLoading).toBe("boolean");
    expect(result.current.data === undefined || Array.isArray(result.current.data)).toBe(true);
  });

  it("useInvalidateRole 返回函数", () => {
    const { result } = renderHook(() => useInvalidateRole(), { wrapper });
    expect(typeof result.current).toBe("function");
  });

  it("invalidate 不抛错", () => {
    const { result } = renderHook(() => useInvalidateRole(), { wrapper });
    act(() => {
      result.current();
    });
  });

  it("多次调用 invalidate 不抛错", () => {
    const { result } = renderHook(() => useInvalidateRole(), { wrapper });
    act(() => {
      result.current();
      result.current();
      result.current();
    });
  });

  it("数据初始为 undefined 或 []", () => {
    const { result } = renderHook(() => useRoleList(), { wrapper });
    expect(result.current.data === undefined || Array.isArray(result.current.data)).toBe(true);
  });
});
