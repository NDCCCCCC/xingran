/**
 * Phase 88 Batch407 — pages/monitor/logs 测试
 */
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";
import { mockApiBatch, resetApiMocks } from "@/test/utils/createApiMock";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

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

describe("pages/monitor/logs", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", async () => {
    const { default: Comp } = await import("../index");
    expect(() => render(<Comp />, { wrapper })).not.toThrow();
  }, 15000);

  it("点击清空按钮 → 确认弹窗 → 发出 /clean 而非 /clear（冒烟：页面实际渲染组件）", async () => {
    // RBAC：permGate 需要 menuStore 就绪位 + monitor:operlog:remove（清空端点对应码）
    const { useMenuStore } = await import("@/store/menuStore");
    useMenuStore.setState({
      permissions: ["monitor:operlog:remove"],
      permissionsLoaded: true,
      permissionsError: null,
    } as never);

    // list 端点注册（页面 mount 即拉列表）；clean 端点 mock 成功
    const handles = mockApiBatch([
      { endpoint: "/monitor/oper-logs/list", response: { code: 0, data: { list: [], total: 0 } } },
      { endpoint: "/monitor/login-logs/list", response: { code: 0, data: { list: [], total: 0 } } },
      { endpoint: "/monitor/oper-logs/clean", response: { code: 0, data: null } },
    ]);
    const cleanHandle = handles["/monitor/oper-logs/clean"];

    const { default: Comp } = await import("../index");
    const { getByRole } = render(<Comp />, { wrapper });

    // 点击工具栏「清空日志」按钮
    const clearBtn = getByRole("button", { name: "清空日志" });
    fireEvent.click(clearBtn);

    // confirmDelete 弹出 Modal.confirm，点确认按钮（文案「清空」）
    const okBtn = await waitFor(() => {
      const btn = document.querySelector(".ant-modal-confirm .ant-btn-dangerous");
      expect(btn).toBeTruthy();
      return btn as HTMLElement;
    });
    fireEvent.click(okBtn);

    // 断言发出的请求 URL 是 /clean（后端真实路由），不是 /clear
    await waitFor(() => {
      const urls = cleanHandle.endpoint.mock.calls.map((c) => c[0]);
      expect(urls).toContain("/monitor/oper-logs/clean");
    });
    const allPostUrls = cleanHandle.endpoint.mock.calls.map((c) => String(c[0]));
    expect(allPostUrls.some((u) => u.includes("/clear"))).toBe(false);
  }, 15000);

  afterEach(() => {
    resetApiMocks();
  });
});
