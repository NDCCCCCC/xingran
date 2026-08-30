/**
 * Phase 88 Batch164 — components/layout/InnovativeLayout 测试
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

vi.mock("@/design-system/components/LayoutSwitcher", () => ({
  default: () => <div data-testid="layout-switcher" />,
}));

vi.mock("@/design-system/components/DensitySwitcher", () => ({
  default: () => <div data-testid="density-switcher" />,
}));

vi.mock("@/store/authStore", () => ({
  useAuthStore: () => ({ logout: vi.fn() }),
}));

vi.mock("../shared/QuickNav", () => ({
  default: () => <div data-testid="quick-nav" />,
}));

import InnovativeLayout from "../InnovativeLayout";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("InnovativeLayout", () => {
  it("渲染 LayoutSwitcher + DensitySwitcher + QuickNav + children", () => {
    const { baseElement } = render(
      <InnovativeLayout>
        <div data-testid="content">Content</div>
      </InnovativeLayout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="layout-switcher"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="density-switcher"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="quick-nav"]')).toBeTruthy();
    expect(baseElement.textContent).toContain("Content");
  });

  it("多 children 渲染", () => {
    const { baseElement } = render(
      <InnovativeLayout>
        <div data-testid="a">A</div>
        <div data-testid="b">B</div>
      </InnovativeLayout>,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="a"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="b"]')).toBeTruthy();
  });
});
