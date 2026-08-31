/**
 * Phase 88 Batch298 — components/shared/EmptyStateWithAction 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
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

describe("shared/EmptyStateWithAction", () => {
  it("只显示 description", () => {
    render(<EmptyStateWithAction description="无数据" />, { wrapper });
    expect(screen.getByText("无数据")).toBeInTheDocument();
  });

  it("含 title → 标题 + 描述", () => {
    render(<EmptyStateWithAction title="提示" description="无数据" />, { wrapper });
    expect(screen.getByText("提示")).toBeInTheDocument();
    expect(screen.getByText("无数据")).toBeInTheDocument();
  });

  it("actionLabel + actionPath → Link 按钮", () => {
    const { container } = render(
      <EmptyStateWithAction description="无数据" actionLabel="新增" actionPath="/create" />,
      { wrapper }
    );
    const link = container.querySelector("a[href='/create']");
    expect(link).toBeTruthy();
    expect(link?.textContent?.replace(/\s/g, "")).toContain("新增");
  });

  it("actionLabel + onAction → 触发回调", () => {
    const onAction = vi.fn();
    const { container } = render(
      <EmptyStateWithAction description="无数据" actionLabel="刷新" onAction={onAction} />,
      { wrapper }
    );
    const btn = container.querySelector("button.ant-btn");
    expect(btn).toBeTruthy();
    if (btn) fireEvent.click(btn);
    expect(onAction).toHaveBeenCalled();
  });

  it("无 actionLabel → 不显示按钮", () => {
    render(<EmptyStateWithAction description="无数据" onAction={vi.fn()} />, { wrapper });
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("actionLabel 但无 actionPath/onAction → 不显示按钮", () => {
    render(<EmptyStateWithAction description="x" actionLabel="go" />, { wrapper });
    expect(screen.queryByRole("button")).toBeNull();
  });
});
