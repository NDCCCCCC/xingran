/**
 * Phase 88 Batch128 — components/shared/ErrorAlertWithRetry 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const logoutMock = vi.fn(() => Promise.resolve());
vi.mock("@/store/authStore", () => ({
  useAuthStore: (sel: any) => sel({ logout: logoutMock }),
}));

import ErrorAlertWithRetry from "../ErrorAlertWithRetry";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("ErrorAlertWithRetry", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("null error → 显示未知错误", () => {
    const { baseElement } = render(<ErrorAlertWithRetry error={null} />, { wrapper });
    expect(baseElement.textContent).toContain("未知错误");
  });

  it("error.code=1006 → 显示设备不存在", () => {
    const { baseElement } = render(<ErrorAlertWithRetry error={{ code: 1006 }} />, { wrapper });
    expect(baseElement.textContent).toContain("设备不存在");
  });

  it("error.code=500 → 显示服务暂不可用", () => {
    const { baseElement } = render(<ErrorAlertWithRetry error={{ code: 500 }} />, { wrapper });
    expect(baseElement.textContent).toContain("服务暂不可用");
  });

  it("error.code=其他 + message → 显示查询失败 + 错误码", () => {
    const { baseElement } = render(
      <ErrorAlertWithRetry error={{ code: 999, message: "自定义消息" }} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("查询失败");
    expect(baseElement.textContent).toContain("自定义消息");
    expect(baseElement.textContent).toContain("999");
  });

  it("Error instance + message → 显示消息", () => {
    const err = new Error("Boom");
    const { baseElement } = render(<ErrorAlertWithRetry error={err} />, { wrapper });
    expect(baseElement.textContent).toContain("Boom");
  });

  it("onRetry 提供 → 显示重新加载按钮", () => {
    const onRetry = vi.fn();
    const { getByText } = render(<ErrorAlertWithRetry error={{ code: 500 }} onRetry={onRetry} />, {
      wrapper,
    });
    fireEvent.click(getByText("重新加载"));
    expect(onRetry).toHaveBeenCalled();
  });

  it("error.code=1007 → 触发 logout + 跳转", async () => {
    // window.location.href mock
    const origLocation = window.location;
    delete (window as any).location;
    (window as any).location = { href: "" };

    render(<ErrorAlertWithRetry error={{ code: 1007 }} />, { wrapper });
    await waitFor(() => {
      expect(logoutMock).toHaveBeenCalled();
    });

    (window as any).location = origLocation;
  });

  it("error.response.data.code 提取", () => {
    const { baseElement } = render(
      <ErrorAlertWithRetry error={{ response: { data: { code: 1006 } } } as any} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("设备不存在");
  });

  it("error.status 提取 (>=400)", () => {
    const { baseElement } = render(<ErrorAlertWithRetry error={{ status: 500 } as any} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("服务暂不可用");
  });

  it("自定义 description 覆盖", () => {
    const { baseElement } = render(
      <ErrorAlertWithRetry error={{ code: 500 }} description="自定义描述" />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("自定义描述");
  });
});
