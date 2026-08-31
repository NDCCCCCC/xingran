/**
 * Phase 88 Batch301 — hooks/useDeptTree 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockTree = [{ id: "d1", deptName: "Root", children: [] }];
vi.mock("@/lib/dutyApi", () => ({
  getDeptTree: vi.fn(async () => ({ code: 0, data: mockTree })),
}));

import { useDeptTree, useInvalidateDept } from "../useDeptTree";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("hooks/useDeptTree", () => {
  it("返回 shape", () => {
    const { result } = renderHook(() => useDeptTree(), { wrapper });
    expect(typeof result.current.isError).toBe("boolean");
    expect(typeof result.current.isLoading).toBe("boolean");
  });

  it("useInvalidateDept 返回函数", () => {
    const { result } = renderHook(() => useInvalidateDept(), { wrapper });
    expect(typeof result.current).toBe("function");
  });

  it("invalidate 函数可调用不抛错", () => {
    const { result } = renderHook(() => useInvalidateDept(), { wrapper });
    act(() => {
      result.current();
    });
  });

  it("invalidate 调用 QueryClient", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(qc, "invalidateQueries");
    const wrap = ({ children: c }: { children: ReactNode }) => (
      <QueryClientProvider client={qc}>{c}</QueryClientProvider>
    );
    const { result } = renderHook(() => useInvalidateDept(), { wrapper: wrap });
    act(() => {
      result.current();
    });
    expect(spy).toHaveBeenCalled();
  });
});
