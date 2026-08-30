/**
 * Phase 88 Batch127 — components/shared/EmptyStateWithAction 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import EmptyStateWithAction from "../EmptyStateWithAction";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("EmptyStateWithAction", () => {
  it("基础渲染 + description", () => {
    const { baseElement } = render(<EmptyStateWithAction description="暂无数据" />, { wrapper });
    expect(baseElement.textContent).toContain("暂无数据");
  });

  it("title + description → 显示 title 与 description", () => {
    const { baseElement } = render(
      <EmptyStateWithAction title="无结果" description="请检查筛选条件" />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("无结果");
    expect(baseElement.textContent).toContain("请检查筛选条件");
  });

  it("actionLabel + actionPath → 渲染 Link + Button", () => {
    const { baseElement, getByText } = render(
      <EmptyStateWithAction description="空" actionLabel="去添加" actionPath="/add" />,
      { wrapper }
    );
    const link = getByText("去添加").closest("a");
    expect(link?.getAttribute("href")).toBe("/add");
  });

  it("actionLabel + onAction → 渲染 Button + 触发回调", () => {
    const onAction = vi.fn();
    const { baseElement } = render(
      <EmptyStateWithAction description="空" actionLabel="点我" onAction={onAction} />,
      { wrapper }
    );
    const btn = baseElement.querySelector("button") as HTMLButtonElement;
    fireEvent.click(btn);
    expect(onAction).toHaveBeenCalled();
  });

  it("actionLabel 但无 path/onAction → 不渲染按钮", () => {
    const { baseElement, queryByText } = render(
      <EmptyStateWithAction description="空" actionLabel="按钮" />,
      { wrapper }
    );
    expect(queryByText("按钮")).toBeNull();
  });

  it("无 actionLabel → 不渲染按钮", () => {
    const { queryByText } = render(<EmptyStateWithAction description="空" actionPath="/add" />, {
      wrapper,
    });
    expect(queryByText("/add")).toBeNull();
  });

  it("自定义 icon 渲染", () => {
    const { baseElement } = render(
      <EmptyStateWithAction description="空" icon={<span data-testid="custom-icon">★</span>} />,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="custom-icon"]')).toBeTruthy();
  });
});
