/**
 * 表格类轻量 hooks 组合测试
 *
 * 覆盖:useTableQuery(React Query 数据面,与 useTableManager 互补)、
 * useTableSettings(分页配置聚合)。
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { useTableQuery } from "./useTableQuery";
import { useTableSettings } from "./useTableSettings";

function createQueryWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { wrapper, queryClient };
}

describe("useTableQuery", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it("按 resource+分页+filters 组装 queryKey 并调用 queryFn", async () => {
    const queryFn = vi.fn().mockResolvedValue({ list: [{ id: "1" }], total: 1 });
    const { wrapper } = createQueryWrapper();

    const { result } = renderHook(
      () =>
        useTableQuery({
          resource: "workstations",
          current: 2,
          pageSize: 20,
          filters: { status: 0 },
          queryFn,
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(queryFn).toHaveBeenCalledWith({ current: 2, pageSize: 20, status: 0 });
    expect(result.current.data).toEqual({ list: [{ id: "1" }], total: 1 });
  });

  it("filters 缺省为空对象", async () => {
    const queryFn = vi.fn().mockResolvedValue({ list: [], total: 0 });
    const { wrapper } = createQueryWrapper();

    const { result } = renderHook(
      () =>
        useTableQuery({
          resource: "r",
          current: 1,
          pageSize: 10,
          queryFn,
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(queryFn).toHaveBeenCalledWith({ current: 1, pageSize: 10 });
  });

  it("queryFn 失败 → isError", async () => {
    const queryFn = vi.fn().mockRejectedValue(new Error("fetch fail"));
    const { wrapper } = createQueryWrapper();

    const { result } = renderHook(
      () =>
        useTableQuery({
          resource: "r",
          current: 1,
          pageSize: 10,
          queryFn,
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it("翻页时 keepPreviousData 保持上一页数据(isPlaceholderData)", async () => {
    const queryFn = vi.fn(async (params: { current: number }) => ({
      list: [{ page: params.current }],
      total: 100,
    }));
    const { wrapper } = createQueryWrapper();

    const { result, rerender } = renderHook(
      ({ current }) =>
        useTableQuery({
          resource: "paged",
          current,
          pageSize: 10,
          queryFn,
        }),
      { wrapper, initialProps: { current: 1 } }
    );

    await waitFor(() => expect(result.current.data?.list[0]).toEqual({ page: 1 }));

    rerender({ current: 2 });
    // 过渡期 placeholder:旧数据仍在且 isPlaceholderData=true
    await waitFor(() => expect(result.current.isPlaceholderData).toBe(true));
    await waitFor(() => expect(result.current.data?.list[0]).toEqual({ page: 2 }));
  });
});

describe("useTableSettings", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  const routerWrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={["/system/user"]}>{children}</MemoryRouter>
  );

  it("默认启用全局分页,pagination 暴露 current/pageSize", () => {
    const { result } = renderHook(() => useTableSettings(), {
      wrapper: routerWrapper,
    });

    expect(result.current.pagination.current).toBe(1);
    expect(result.current.pagination.pageSize).toBe(10);

    act(() => result.current.pagination.setCurrent(3));
    expect(result.current.pagination.current).toBe(3);
    expect(sessionStorage.getItem("xingran_table_state_system_user_current")).toBe("3");
  });

  it("getTableProps 返回分页配置可直接喂 Table", () => {
    const { result } = renderHook(() => useTableSettings(), {
      wrapper: routerWrapper,
    });

    const props = result.current.getTableProps();
    expect(props.pagination).toBe(result.current.pagination.paginationProps);
  });

  it("enableGlobalPagination=false 时不覆盖 pageSize", () => {
    sessionStorage.setItem("xingran_table_state_system_user_pageSize", "50");
    const { result } = renderHook(() => useTableSettings({ enableGlobalPagination: false }), {
      wrapper: routerWrapper,
    });
    // pageSize 未被 options 覆盖,保留持久化值 50
    expect(result.current.pagination.pageSize).toBe(50);
  });

  it("pageSize 选项覆盖持久化值", () => {
    sessionStorage.setItem("xingran_table_state_system_user_pageSize", "50");
    const { result } = renderHook(() => useTableSettings({ pageSize: 20 }), {
      wrapper: routerWrapper,
    });
    expect(result.current.pagination.pageSize).toBe(20);
  });
});
