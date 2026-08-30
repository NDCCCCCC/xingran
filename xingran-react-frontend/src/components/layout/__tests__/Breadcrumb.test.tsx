/**
 * Phase 88 Batch194 — components/layout/breadcrumb 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const state = {
  initialized: true,
  items: [
    { path: "/", title: "首页" },
    { path: "/system", title: "系统管理" },
  ] as Array<{ path: string; title: string }>,
};

vi.mock("@/router/routeConfigManager", () => ({
  routeConfigManager: {
    isInitialized: () => state.initialized,
    buildBreadcrumb: () => state.items,
  },
}));

import BreadcrumbComponent from "../breadcrumb";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter initialEntries={["/system"]}>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("layout/Breadcrumb", () => {
  beforeEach(() => {
    state.initialized = true;
    state.items = [
      { path: "/", title: "首页" },
      { path: "/system", title: "系统管理" },
    ];
  });

  it("渲染 Breadcrumb items", () => {
    render(<BreadcrumbComponent />, { wrapper });
    expect(screen.getByText("首页")).toBeInTheDocument();
    expect(screen.getByText("系统管理")).toBeInTheDocument();
  });

  it("最后一项不含 Link (span)", () => {
    render(<BreadcrumbComponent />, { wrapper });
    const links = screen.getAllByRole("link");
    expect(links.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("系统管理").tagName).toBe("SPAN");
  });

  it("空 breadcrumb → 不渲染 Breadcrumb", () => {
    state.items = [];
    const { container } = render(<BreadcrumbComponent />, { wrapper });
    // AntdApp 外壳存在但 Breadcrumb 项无
    expect(container.querySelector(".ant-breadcrumb")).toBeNull();
  });

  it("未初始化 → 不渲染 Breadcrumb", () => {
    state.initialized = false;
    const { container } = render(<BreadcrumbComponent />, { wrapper });
    expect(container.querySelector(".ant-breadcrumb")).toBeNull();
  });
});
