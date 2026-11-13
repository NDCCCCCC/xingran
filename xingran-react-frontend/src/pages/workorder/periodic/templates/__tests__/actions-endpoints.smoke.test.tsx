/**
 * 冒烟：周期工单模板「启停 / 立即生成 / 执行记录」按钮 → 调用 4 个恢复端点。
 *
 * 背景：后端 workorder_router.go 恢复了 4 条曾丢失 HTTP 通路的 86 新端点
 *   POST /workorder/periodic/templates/:id/enable   （启用）
 *   POST /workorder/periodic/templates/:id/disable  （停用）
 *   POST /workorder/periodic/templates/:id/generate （立即生成）
 *   POST /workorder/periodic/templates/:id/logs     （执行记录）
 * 前端 workorderApi.ts 的 enablePeriodicTemplate / disablePeriodicTemplate /
 * generateWorkOrderNow / getPeriodicLogs 分别对应，useTemplateActions.ts 的
 * handleToggleEnabled / handleGenerateNow 与 useTemplateData.ts 的 fetchLogs
 * 是按钮到端点的桥梁。本测试用真实 hook（renderHook + App/QueryClient context）
 * 驱动，断言 4 个 URL 全部被 post 命中，防止路由再次断链。
 *
 * 端点 URL 用 createApiMock 端点工厂的 endpoint spy 断言，零新依赖。
 */
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <AntdApp>{children}</AntdApp>
    </QueryClientProvider>
  );
}

const ID = "tpl-smoke-1";
const URL = {
  enable: `/workorder/periodic/templates/${ID}/enable`,
  disable: `/workorder/periodic/templates/${ID}/disable`,
  generate: `/workorder/periodic/templates/${ID}/generate`,
  logs: `/workorder/periodic/templates/${ID}/logs`,
} as const;

/** 每个用例前把 4 个端点 spy 注册好并给成功返回值 */
async function registerEndpoints() {
  const { mockApiBatch } = await import("@/test/utils/createApiMock");
  const ok = { code: 0, data: { message: "ok" } };
  return mockApiBatch([
    { endpoint: URL.enable, response: ok },
    { endpoint: URL.disable, response: ok },
    { endpoint: URL.generate, response: ok },
    { endpoint: URL.logs, response: ok },
  ]);
}

describe("workorder/periodic/templates — 4 个恢复端点冒烟", () => {
  beforeEach(async () => {
    const { setGenericFallback } = await import("@/test/utils/createApiMock");
    // 未登记端点（useUserOptions 等 mount 期查询）给安全空结构
    setGenericFallback({ code: 0, data: { list: [], total: 0 } });
  });

  afterEach(async () => {
    const { resetApiMocks } = await import("@/test/utils/createApiMock");
    resetApiMocks();
  });

  it("启用（isEnabled=false）→ /enable；停用（isEnabled=true）→ /disable；立即生成 → /generate", async () => {
    const handles = await registerEndpoints();
    const { useTemplateActions } = await import("../hooks/useTemplateActions");
    const onLoad = vi.fn();
    const { result } = renderHook(() => useTemplateActions({ onLoad }), { wrapper });

    const disabledRecord = { id: ID, isEnabled: false } as never;
    const enabledRecord = { id: ID, isEnabled: true } as never;

    await act(async () => {
      await result.current.handleToggleEnabled(disabledRecord);
    });
    await act(async () => {
      await result.current.handleToggleEnabled(enabledRecord);
    });
    await act(async () => {
      await result.current.handleGenerateNow(ID);
    });

    expect(handles[URL.enable].endpoint).toHaveBeenCalledTimes(1);
    expect(handles[URL.disable].endpoint).toHaveBeenCalledTimes(1);
    expect(handles[URL.generate].endpoint).toHaveBeenCalledTimes(1);
    // URL 形态断言（第一个参数就是原始 url）
    expect(String(handles[URL.enable].endpoint.mock.calls[0][0])).toBe(URL.enable);
    expect(String(handles[URL.disable].endpoint.mock.calls[0][0])).toBe(URL.disable);
    expect(String(handles[URL.generate].endpoint.mock.calls[0][0])).toBe(URL.generate);
    expect(onLoad).toHaveBeenCalledTimes(3);
  });

  it("执行记录 → /logs（第 4 个恢复端点）", async () => {
    const handles = await registerEndpoints();
    const { useTemplateData } = await import("../hooks/useTemplateData");
    const searchForm = {
      getFieldsValue: () => ({}),
      setFieldsValue: () => {},
      resetFields: () => {},
    } as never;
    const { result } = renderHook(() => useTemplateData(searchForm, 1, 10), { wrapper });

    await act(async () => {
      await result.current.fetchLogs(ID);
    });

    expect(handles[URL.logs].endpoint).toHaveBeenCalledTimes(1);
    expect(String(handles[URL.logs].endpoint.mock.calls[0][0])).toBe(URL.logs);
  });

  it("四个 API 函数各自命中对应端点 URL（静态契约锁）", async () => {
    const handles = await registerEndpoints();
    const mod = await import("@/lib/workorderApi");
    expect(typeof mod.enablePeriodicTemplate).toBe("function");
    expect(typeof mod.disablePeriodicTemplate).toBe("function");
    expect(typeof mod.generateWorkOrderNow).toBe("function");
    expect(typeof mod.getPeriodicLogs).toBe("function");

    await mod.enablePeriodicTemplate(ID);
    await mod.disablePeriodicTemplate(ID);
    await mod.generateWorkOrderNow(ID);
    await mod.getPeriodicLogs(ID);

    expect(handles[URL.enable].endpoint).toHaveBeenCalledTimes(1);
    expect(handles[URL.disable].endpoint).toHaveBeenCalledTimes(1);
    expect(handles[URL.generate].endpoint).toHaveBeenCalledTimes(1);
    expect(handles[URL.logs].endpoint).toHaveBeenCalledTimes(1);
  });
});
