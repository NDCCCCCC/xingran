/**
 * Phase 88 Batch256 — hooks/useTableQuery 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useTableQuery } from "../useTableQuery";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("hooks/useTableQuery", () => {
  it("调用 queryFn 并返回数据", async () => {
    const queryFn = vi.fn(async (params: any) => ({
      list: [{ id: "1", name: "Item 1" }],
      total: 1,
      current: params.current,
      pageSize: params.pageSize,
    }));
    const { result } = renderHook(
      () =>
        useTableQuery({
          resource: "test",
          current: 1,
          pageSize: 10,
          queryFn,
        }),
      { wrapper }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryFn).toHaveBeenCalled();
    expect(result.current.data?.total).toBe(1);
  });

  it("filters 传递到 queryFn", async () => {
    const queryFn = vi.fn(async (params: any) => ({
      list: [],
      total: 0,
    }));
    renderHook(
      () =>
        useTableQuery({
          resource: "test",
          current: 1,
          pageSize: 20,
          filters: { name: "abc", status: 0 },
          queryFn,
        }),
      { wrapper }
    );
    await waitFor(() => {
      expect(queryFn).toHaveBeenCalled();
    });
    const lastCall = queryFn.mock.calls[queryFn.mock.calls.length - 1][0];
    expect(lastCall.name).toBe("abc");
    expect(lastCall.status).toBe(0);
  });

  it("current/pageSize 变化 → 重新请求", async () => {
    const queryFn = vi.fn(async () => ({ list: [], total: 0 }));
    const { rerender } = renderHook(
      ({ current }: any) =>
        useTableQuery({
          resource: "test",
          current,
          pageSize: 10,
          queryFn,
        }),
      { wrapper, initialProps: { current: 1 } }
    );
    await waitFor(() => {
      expect(queryFn).toHaveBeenCalled();
    });
    const firstCalls = queryFn.mock.calls.length;
    rerender({ current: 2 });
    await waitFor(() => {
      expect(queryFn.mock.calls.length).toBeGreaterThan(firstCalls);
    });
  });
});
