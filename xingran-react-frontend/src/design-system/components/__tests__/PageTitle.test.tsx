/**
 * Phase 88 Batch166 — design-system/components/PageTitle 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import PageTitle from "../PageTitle";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <>{children}</>;
}

describe("PageTitle", () => {
  it("pre + post → 显示标题 + dot 分隔符", () => {
    const { baseElement } = render(<PageTitle pre="系统" post="用户" />, { wrapper });
    expect(baseElement.textContent).toContain("系统");
    expect(baseElement.textContent).toContain("用户");
    expect(baseElement.querySelector(".dot")).toBeTruthy();
  });

  it("post 未提供 → 不显示 dot", () => {
    const { baseElement } = render(<PageTitle pre="用户" />, { wrapper });
    expect(baseElement.textContent).toContain("用户");
    expect(baseElement.querySelector(".dot")).toBeNull();
  });

  it("sub 提供 → 显示副标题", () => {
    const { baseElement } = render(<PageTitle pre="系统" post="用户" sub="副标题" />, { wrapper });
    expect(baseElement.textContent).toContain("副标题");
    expect(baseElement.querySelector(".page-sub")).toBeTruthy();
  });

  it("actions 提供 → 显示操作区", () => {
    const { baseElement } = render(
      <PageTitle pre="系统" post="用户" actions={<button data-testid="action-btn">新增</button>} />,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="action-btn"]')).toBeTruthy();
    expect(baseElement.querySelector(".page-actions")).toBeTruthy();
  });

  it("page-head className 渲染", () => {
    const { baseElement } = render(<PageTitle pre="A" />, { wrapper });
    expect(baseElement.querySelector(".page-head")).toBeTruthy();
    expect(baseElement.querySelector(".page-title")).toBeTruthy();
  });
});
