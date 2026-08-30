/**
 * Phase 88 Batch167 — design-system/components/LayoutProvider 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let mockLayout = "classic";
let mockDensity = "normal";
let mockCollapsed = false;

vi.mock("@/store/layoutStore", () => ({
  useLayoutStore: () => ({
    currentLayout: mockLayout,
    sidebarCollapsed: mockCollapsed,
    density: mockDensity,
  }),
}));

import LayoutProvider, { useLayoutContext } from "../LayoutProvider";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

function Consumer() {
  const ctx = useLayoutContext();
  return <div data-testid="consumer">{JSON.stringify(ctx)}</div>;
}

describe("LayoutProvider", () => {
  beforeEach(() => {
    mockLayout = "classic";
    mockDensity = "normal";
    mockCollapsed = false;
  });

  it("渲染 children", () => {
    const { baseElement } = render(
      <LayoutProvider>
        <div data-testid="child">Child</div>
      </LayoutProvider>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="child"]')).toBeTruthy();
  });

  it("data-layout 属性设置到 documentElement", () => {
    mockLayout = "hybrid";
    render(
      <LayoutProvider>
        <div />
      </LayoutProvider>,
      { wrapper }
    );
    expect(document.documentElement.getAttribute("data-layout")).toBe("hybrid");
  });

  it("data-density 属性设置", () => {
    mockDensity = "compact";
    render(
      <LayoutProvider>
        <div />
      </LayoutProvider>,
      { wrapper }
    );
    expect(document.documentElement.getAttribute("data-density")).toBe("compact");
  });

  it("sidebarCollapsed=true → data-sidebar-collapsed=true", () => {
    mockCollapsed = true;
    render(
      <LayoutProvider>
        <div />
      </LayoutProvider>,
      { wrapper }
    );
    expect(document.documentElement.getAttribute("data-sidebar-collapsed")).toBe("true");
  });

  it("sidebarCollapsed=false → 移除 data-sidebar-collapsed", () => {
    mockCollapsed = false;
    document.documentElement.setAttribute("data-sidebar-collapsed", "true");
    render(
      <LayoutProvider>
        <div />
      </LayoutProvider>,
      { wrapper }
    );
    expect(document.documentElement.getAttribute("data-sidebar-collapsed")).toBeNull();
  });

  it("useLayoutContext 提供 context 值", () => {
    mockLayout = "innovative";
    mockCollapsed = true;
    const { baseElement } = render(
      <LayoutProvider>
        <Consumer />
      </LayoutProvider>,
      { wrapper }
    );
    const text = baseElement.querySelector('[data-testid="consumer"]')?.textContent ?? "";
    expect(text).toContain("innovative");
    expect(text).toContain("true");
  });
});
