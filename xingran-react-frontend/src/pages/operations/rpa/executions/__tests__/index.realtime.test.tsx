/**
 * CONTRACT-05（D16）执行页实时化测试。
 *
 * 断言契约（红绿变异点）：
 *   1. 执行页接入了 useRPAProgress（模块被真实调用）
 *   2. 不再用 setInterval(5000) 轮询 —— 每次 5s 触发只可能是旧实现
 *   3. WS 进度消息到达时，列表请求确实被重新发起（实时刷新链路成立）
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, waitFor, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { createApiMock, resetApiMocks } from "@/test/utils/createApiMock";
import { useNoticeStore } from "@/store/noticeStore";
import type { RPAProgressMessage } from "@/types/rpa";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/operations/rpa/executions"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

/** 把一条 RPA 进度消息通过 noticeStore 的订阅链派发出去。 */
function dispatchProgress(overrides: Partial<RPAProgressMessage> = {}) {
  const listener = useNoticeStore.getState() as unknown as {
    onRPAProgress: (cb: (m: RPAProgressMessage) => void) => () => void;
  };
  // 直接调用 store 的 handleWsMessage 需要构造完整 WS 消息；
  // 这里走更直接的路径：取当前所有订阅者并派发。
  void listener;
  const message: RPAProgressMessage = {
    type: "rpa_progress",
    executionId: "exec-1",
    taskId: "task-1",
    taskName: "任务一",
    step: 2,
    total: 5,
    message: "步骤 2",
    status: "running",
    timestamp: Date.now(),
    ...overrides,
  };
  // noticeStore 的 RPA 派发入口（handleWsMessage 解析 content 后回调订阅者）
  useNoticeStore.getState().handleWsMessage({
    type: "rpa_progress",
    content: JSON.stringify(message),
  } as never);
}

beforeEach(() => {
  resetApiMocks();
  vi.useFakeTimers({ shouldAdvanceTime: true });
});

afterEach(() => {
  vi.useRealTimers();
  resetApiMocks();
});

describe("RPA 执行页实时进度（useRPAProgress 接线）", () => {
  it("存在 running 任务时也不以 5 秒间隔轮询列表", async () => {
    const { default: ExecutionManagement } = await import("../index");
    const listApi = createApiMock("/rpa/executions/list");
    // ⚠ 刻意返回一条 running 记录：旧实现的 5s 轮询**只在有 running 时才刷新**，
    // 空列表下新旧实现都不会重载，测不出差异（红绿变异会失效）。
    listApi.endpoint.mockResolvedValue({
      code: 0,
      data: {
        list: [{ id: "exec-running", taskName: "任务一", status: "running" }],
        total: 1,
      },
    });
    const statApi = createApiMock("/rpa/executions/statistics");
    statApi.endpoint.mockResolvedValue({ code: 0, data: { total: 1, running: 1 } });

    render(<ExecutionManagement />, { wrapper });

    await waitFor(() => {
      expect(listApi.endpoint).toHaveBeenCalledTimes(1);
    });

    // 推进 12 秒虚拟时钟：旧实现（setInterval 5000 + running 判定）会再打 2~3 次；
    // 新实现只有 30s 断线兜底，12s 内不应触发。
    await act(async () => {
      await vi.advanceTimersByTimeAsync(12000);
    });

    expect(listApi.endpoint).toHaveBeenCalledTimes(1);
  });

  it("WebSocket 进度消息到达时刷新列表（实时链路成立）", async () => {
    const { default: ExecutionManagement } = await import("../index");
    const listApi = createApiMock("/rpa/executions/list");
    listApi.endpoint.mockResolvedValue({ code: 0, data: { list: [], total: 0 } });
    const statApi = createApiMock("/rpa/executions/statistics");
    statApi.endpoint.mockResolvedValue({ code: 0, data: { total: 0 } });

    render(<ExecutionManagement />, { wrapper });

    await waitFor(() => {
      expect(listApi.endpoint).toHaveBeenCalledTimes(1);
    });

    await act(async () => {
      dispatchProgress();
    });

    // WS 事件 → loadExecutions 被再次调用
    await waitFor(() => {
      expect(listApi.endpoint.mock.calls.length).toBeGreaterThanOrEqual(2);
    });
  });
});
