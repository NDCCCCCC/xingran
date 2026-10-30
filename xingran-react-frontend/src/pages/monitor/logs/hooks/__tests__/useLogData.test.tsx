/**
 * Phase 88 Batch76 — useLogData 钩子测试(33 stmts)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useLogData, type UseLogDataParams } from "../useLogData";
import { createApiMock } from "@/test/utils/createApiMock";

function wrapper({ children }: { children: React.ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

function defaultParams(): UseLogDataParams {
  return {
    activeTab: "oper",
    searchForm: { timeRange: undefined as unknown } as unknown,
    loginSearchForm: { timeRange: undefined as unknown } as unknown,
    current: 1,
    pageSize: 10,
  };
}

describe("useLogData", () => {
  it("初始化默认值: 空数组/loading=false/total=0", () => {
    const { result } = renderHook(() => useLogData(defaultParams()), { wrapper });
    expect(result.current.operLogs).toEqual([]);
    expect(result.current.loginLogs).toEqual([]);
    expect(result.current.loading).toBe(false);
    expect(result.current.total).toBe(0);
  });

  it("fetchOperLogs 成功 → 写入 operLogs + total", async () => {
    const api = createApiMock("/monitor/oper-logs/list");
    api.endpoint.mockResolvedValueOnce({
      data: { list: [{ id: "1", module: "m", action: "create" } as unknown], total: 5 },
      code: 0,
      message: "",
      timestamp: 0,
      request_id: "",
    } as unknown);

    const { result } = renderHook(() => useLogData(defaultParams()), { wrapper });

    await act(async () => {
      await result.current.fetchOperLogs();
    });

    expect(result.current.operLogs).toHaveLength(1);
    expect(result.current.total).toBe(5);
    expect(api.endpoint).toHaveBeenCalledWith(
      "/monitor/oper-logs/list",
      expect.objectContaining({ current: 1, pageSize: 10 })
    );
  });

  it("fetchOperLogs timeRange 非空 → 追加 startTime/endTime ISO", async () => {
    const api = createApiMock("/monitor/oper-logs/list");
    api.endpoint.mockResolvedValueOnce({
      data: { list: [], total: 0 },
      code: 0,
      message: "",
      timestamp: 0,
      request_id: "",
    } as unknown);

    const params = defaultParams();
    const start = new Date("2026-01-01T00:00:00Z");
    const end = new Date("2026-01-31T23:59:59Z");
    params.searchForm = { ...params.searchForm, timeRange: [start, end] as unknown };

    const { result } = renderHook(() => useLogData(params), { wrapper });

    await act(async () => {
      await result.current.fetchOperLogs();
    });

    expect(api.endpoint).toHaveBeenCalledWith(
      "/monitor/oper-logs/list",
      expect.objectContaining({ startTime: start.toISOString(), endTime: end.toISOString() })
    );
  });

  it("fetchOperLogs 失败 → catch 路径", async () => {
    const api = createApiMock("/monitor/oper-logs/list");
    api.endpoint.mockRejectedValueOnce(new Error("network"));
    const { result } = renderHook(() => useLogData(defaultParams()), { wrapper });
    await act(async () => {
      await result.current.fetchOperLogs();
    });
    expect(result.current.loading).toBe(false);
  });

  it("fetchLoginLogs 成功 → 写入 loginLogs", async () => {
    const api = createApiMock("/monitor/login-logs/list");
    api.endpoint.mockResolvedValueOnce({
      data: { list: [{ id: "l1", userName: "alice" } as unknown], total: 3 },
      code: 0,
      message: "",
      timestamp: 0,
      request_id: "",
    } as unknown);

    const { result } = renderHook(() => useLogData(defaultParams()), { wrapper });

    await act(async () => {
      await result.current.fetchLoginLogs();
    });

    expect(result.current.loginLogs).toHaveLength(1);
    expect(result.current.total).toBe(3);
    expect(api.endpoint).toHaveBeenCalledWith("/monitor/login-logs/list", expect.any(Object));
  });

  it("fetchLoginLogs 失败 → message.error catch", async () => {
    const api = createApiMock("/monitor/login-logs/list");
    api.endpoint.mockRejectedValueOnce(new Error("err"));
    const { result } = renderHook(() => useLogData(defaultParams()), { wrapper });
    await act(async () => {
      await result.current.fetchLoginLogs();
    });
    expect(result.current.loading).toBe(false);
  });

  it("setOperLogs/setLoginLogs/setTotal 直接写入", () => {
    const { result } = renderHook(() => useLogData(defaultParams()), { wrapper });
    act(() => {
      result.current.setOperLogs([{ id: "x" } as unknown]);
      result.current.setLoginLogs([{ id: "y" } as unknown]);
      result.current.setTotal(42);
    });
    expect(result.current.operLogs).toHaveLength(1);
    expect(result.current.loginLogs).toHaveLength(1);
    expect(result.current.total).toBe(42);
  });
});
