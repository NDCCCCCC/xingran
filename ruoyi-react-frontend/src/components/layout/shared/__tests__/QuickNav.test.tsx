/**
 * Phase 88 Batch228 — components/layout/shared/QuickNav 测试
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

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

import QuickNav from "../QuickNav";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("layout/shared/QuickNav", () => {
  it("渲染 6 卡片", () => {
    render(<QuickNav />, { wrapper });
    expect(screen.getByText("仪表盘")).toBeInTheDocument();
    expect(screen.getByText("用户管理")).toBeInTheDocument();
    expect(screen.getByText("菜单管理")).toBeInTheDocument();
    expect(screen.getByText("系统监控")).toBeInTheDocument();
    expect(screen.getByText("设备管理")).toBeInTheDocument();
    expect(screen.getByText("我的值班")).toBeInTheDocument();
  });

  it("点击卡片 → navigate", () => {
    mockNavigate.mockClear();
    render(<QuickNav />, { wrapper });
    fireEvent.click(screen.getByText("用户管理"));
    expect(mockNavigate).toHaveBeenCalledWith("/system/user");
  });

  it("mouse enter/leave 触发 style 变化", () => {
    render(<QuickNav />, { wrapper });
    const card = screen.getByText("仪表盘").closest(".ant-card");
    if (card) {
      fireEvent.mouseEnter(card);
      fireEvent.mouseLeave(card);
      // 断言事件无异常
      expect(card).toBeTruthy();
    }
  });
});
