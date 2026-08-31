/**
 * Phase 88 Batch297 — hooks/useExceptionList 测试
 */
import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useExceptionList } from "../useExceptionList";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("hooks/useExceptionList", () => {
  it("返回 shape", () => {
    const { result } = renderHook(() => useExceptionList({ current: 1, pageSize: 20 }), {
      wrapper,
    });
    expect(result.current.data).toBeUndefined();
    expect(typeof result.current.isError).toBe("boolean");
  });

  it("返回 isError=false 初始", () => {
    const { result } = renderHook(() => useExceptionList({ current: 1, pageSize: 20 }), {
      wrapper,
    });
    expect(result.current.isError).toBe(false);
  });

  it("不同参数调用 hook", () => {
    const { result } = renderHook(
      () => useExceptionList({ current: 2, pageSize: 50, conflictType: "ip" }),
      { wrapper }
    );
    expect(result.current).toBeDefined();
  });
});
