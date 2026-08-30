/**
 * Phase 88 Batch173 — components/layout/index (Layout 入口) 测试
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

let mockLayout = "classic";

vi.mock("@/store/layoutStore", () => ({
  useLayoutStore: (sel?: any) =>
    sel ? sel({ currentLayout: mockLayout }) : { currentLayout: mockLayout },
}));

vi.mock("../ClassicLayout", () => ({
  default: ({ children }: any) => <div data-testid="classic-layout">{children}</div>,
}));

vi.mock("../HybridLayout", () => ({
  default: ({ children }: any) => <div data-testid="hybrid-layout">{children}</div>,
}));

vi.mock("../InnovativeLayout", () => ({
  default: ({ children }: any) => <div data-testid="innovative-layout">{children}</div>,
}));

import Layout from "../index";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("Layout 入口", () => {
  it("currentLayout='classic' → 渲染 ClassicLayout", () => {
    mockLayout = "classic";
    const { baseElement } = render(
      <Layout>
        <div data-testid="child">Content</div>
      </Layout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="classic-layout"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="hybrid-layout"]')).toBeNull();
    expect(baseElement.querySelector('[data-testid="innovative-layout"]')).toBeNull();
  });

  it("currentLayout='hybrid' → 渲染 HybridLayout", () => {
    mockLayout = "hybrid";
    const { baseElement } = render(
      <Layout>
        <div data-testid="child">Content</div>
      </Layout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="hybrid-layout"]')).toBeTruthy();
  });

  it("currentLayout='innovative' → 渲染 InnovativeLayout", () => {
    mockLayout = "innovative";
    const { baseElement } = render(
      <Layout>
        <div data-testid="child">Content</div>
      </Layout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="innovative-layout"]')).toBeTruthy();
  });

  it("currentLayout 未知值 → fallback ClassicLayout", () => {
    mockLayout = "unknown";
    const { baseElement } = render(
      <Layout>
        <div data-testid="child">Content</div>
      </Layout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="classic-layout"]')).toBeTruthy();
  });

  it("children 透传到对应布局", () => {
    mockLayout = "hybrid";
    const { baseElement } = render(
      <Layout>
        <div data-testid="content">Content</div>
      </Layout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="content"]')).toBeTruthy();
  });
});
