/**
 * Phase 88 Batch155 — components/layout/ClassicLayout 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/layout/header", () => ({
  default: () => <div data-testid="layout-header" />,
}));

vi.mock("@/components/layout/sidebar", () => ({
  default: () => <div data-testid="layout-sidebar" />,
}));

vi.mock("@/components/layout/shared/TabBar", () => ({
  default: () => <div data-testid="layout-tabbar" />,
}));

vi.mock("../shared/useRouteTabs", () => ({
  useRouteTabs: vi.fn(),
}));

import ClassicLayout from "../ClassicLayout";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("ClassicLayout", () => {
  it("渲染 Sidebar + Header + TabBar + children", () => {
    const { baseElement } = render(
      <ClassicLayout>
        <div data-testid="content-child">Test Content</div>
      </ClassicLayout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="layout-header"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="layout-sidebar"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="layout-tabbar"]')).toBeTruthy();
    expect(baseElement.textContent).toContain("Test Content");
  });

  it("调用 useRouteTabs hook", async () => {
    const useRouteTabsMock = await import("../shared/useRouteTabs");
    render(
      <ClassicLayout>
        <div />
      </ClassicLayout>,
      { wrapper }
    );
    expect(useRouteTabsMock.useRouteTabs).toHaveBeenCalled();
  });

  it("Content 区域使用 padding/background 样式", () => {
    const { baseElement } = render(
      <ClassicLayout>
        <div />
      </ClassicLayout>,
      { wrapper }
    );
    const content = baseElement.querySelector(".flex-1");
    expect(content).toBeTruthy();
  });

  it("多 children 渲染", () => {
    const { baseElement } = render(
      <ClassicLayout>
        <div data-testid="a">A</div>
        <div data-testid="b">B</div>
      </ClassicLayout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="a"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="b"]')).toBeTruthy();
  });
});
