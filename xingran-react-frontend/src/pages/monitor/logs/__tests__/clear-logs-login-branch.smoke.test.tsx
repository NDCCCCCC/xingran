/**
 * 冒烟：日志清空按钮 → 二次确认 → 请求 clean 端点。
 *
 * 现有 index.test.tsx 已覆盖 oper 分支（/monitor/oper-logs/clean），本文件补齐
 * login 分支（/monitor/login-logs/clean）—— 切到「登录日志」Tab 后点「清空」，
 * 断言二次确认弹窗走 confirmDelete、确认后只发 login-logs/clean 且绝不落到
 * oper-logs/clean 或 /clear。用 createApiTestingModule 端点工厂断言真实 URL。
 */
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, fireEvent, waitFor, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";
import { mockApiBatch, resetApiMocks, setGenericFallback } from "@/test/utils/createApiMock";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// 页面 loadOperLogs/loadLoginLogs 走 @/lib/api 的 post（已由端点工厂 mock），
// 但组件树里其它模块可能引 opsLogApi；给出安全空实现避免渲染期抛错。
vi.mock("@/lib/opsLogApi", () => ({
  getOperLogList: vi.fn(async () => ({ list: [], total: 0 })),
  getLoginLogList: vi.fn(async () => ({ list: [], total: 0 })),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/monitor/logs"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("monitor/logs — 清空日志 login 分支冒烟", () => {
  beforeEach(async () => {
    // activeTab 默认值 "oper" 由 usePersistedStateController 从 sessionStorage 恢复，
    // 预置 "login" 让页面首屏即落在登录日志分支（无需点 Tab，规避 antd Tabs 时序）。
    window.sessionStorage.setItem(
      "xingran_table_state_monitor_logs_activeTab",
      JSON.stringify("login")
    );
    // ready + 持登录日志删除码 → 「清空日志」按钮可见可用
    const { useMenuStore } = await import("@/store/menuStore");
    useMenuStore.setState({
      permissions: ["monitor:loginlog:remove"],
      permissionsLoaded: true,
      permissionsError: null,
    } as never);
    // 未登记端点的通用兜底（页面 mount 会拉若干统计/字典），给安全空结构
    setGenericFallback({ code: 0, data: { list: [], total: 0 } });
  });

  afterEach(() => {
    resetApiMocks();
    window.sessionStorage.clear();
  });

  it("登录日志 Tab 点清空 → 二次确认 → 仅请求 /monitor/login-logs/clean", async () => {
    const handles = mockApiBatch([
      {
        endpoint: "/monitor/oper-logs/list",
        response: { code: 0, data: { list: [], total: 0 } },
      },
      {
        endpoint: "/monitor/login-logs/list",
        response: { code: 0, data: { list: [], total: 0 } },
      },
      {
        endpoint: "/monitor/login-logs/clean",
        response: { code: 0, data: null },
      },
      {
        endpoint: "/monitor/oper-logs/clean",
        response: { code: 0, data: null },
      },
    ]);
    const loginClean = handles["/monitor/login-logs/clean"];
    const operClean = handles["/monitor/oper-logs/clean"];

    const { default: Comp } = await import("../index");
    render(<Comp />, { wrapper });

    // 首屏落在登录日志分支：搜索表单里的「IP地址」占位符存在
    expect(screen.getByPlaceholderText("IP地址")).toBeTruthy();

    // 点「清空日志」按钮（aria-label / title 均为「清空日志」）
    const clearBtn = await screen.findByRole("button", { name: "清空日志" });
    fireEvent.click(clearBtn);

    // confirmDelete 弹出 Modal.confirm：标题「确认清空」（页面另有一个同名
    // 「日志详情」Modal 标题节点，故用 getAllByText 避免多命中）+ target「全部登录日志」
    await waitFor(() => {
      expect(screen.getAllByText("确认清空").length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText(/全部登录日志/).length).toBeGreaterThan(0);

    // 点确认（危险按钮）
    const okBtn = await waitFor(() => {
      const btn = document.querySelector(".ant-modal-confirm .ant-btn-dangerous");
      expect(btn).toBeTruthy();
      return btn as HTMLElement;
    });
    fireEvent.click(okBtn);

    // 确认后真正发出 login-logs/clean
    await waitFor(() => {
      const urls = loginClean.endpoint.mock.calls.map((c) => String(c[0]));
      expect(urls).toContain("/monitor/login-logs/clean");
    });

    // 负向：oper-logs/clean 一次都不该被调，且任何请求都不含 /clear
    const operUrls = operClean.endpoint.mock.calls.map((c) => String(c[0]));
    expect(operUrls).toHaveLength(0);
    const allUrls = loginClean.endpoint.mock.calls.map((c) => String(c[0]));
    expect(allUrls.some((u) => u.includes("/clear"))).toBe(false);
  }, 20000);
});
