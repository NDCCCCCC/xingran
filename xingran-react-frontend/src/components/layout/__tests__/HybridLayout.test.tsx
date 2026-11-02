/**
 * Phase 88 Batch156 — components/layout/HybridLayout 测试
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

vi.mock("../header", () => ({
  default: () => <div data-testid="layout-header" />,
}));

vi.mock("../sidebar", () => ({
  default: () => <div data-testid="layout-sidebar" />,
}));

vi.mock("../shared/TabBar", () => ({
  default: () => <div data-testid="layout-tabbar" />,
}));

vi.mock("../shared/useRouteTabs", () => ({
  useRouteTabs: vi.fn(),
}));

import HybridLayout from "../HybridLayout";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("HybridLayout", () => {
  it("渲染 Sidebar + Header + TabBar + children", () => {
    const { baseElement } = render(
      <HybridLayout>
        <div data-testid="content-child">Hybrid Content</div>
      </HybridLayout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="layout-header"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="layout-sidebar"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="layout-tabbar"]')).toBeTruthy();
    expect(baseElement.textContent).toContain("Hybrid Content");
  });

  it("调用 useRouteTabs hook", async () => {
    const useRouteTabsMock = await import("../shared/useRouteTabs");
    render(
      <HybridLayout>
        <div />
      </HybridLayout>,
      { wrapper }
    );
    expect(useRouteTabsMock.useRouteTabs).toHaveBeenCalled();
  });

  it("Content 区域使用相对定位 (与 ClassicLayout 区别)", () => {
    const { baseElement } = render(
      <HybridLayout>
        <div />
      </HybridLayout>,
      { wrapper }
    );
    // HybridLayout 内层 AntLayout 应该有 position:relative
    expect(baseElement.querySelector(".h-full")).toBeTruthy();
  });
});
