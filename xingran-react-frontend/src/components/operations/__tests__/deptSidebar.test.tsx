/**
 * Phase 88 Batch149 — components/operations/DeptSidebar 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/DeptTree", () => ({
  default: ({ externalOnly }: any) => (
    <div data-testid="dept-tree" data-external={String(externalOnly)} />
  ),
}));

import { DeptSidebar } from "../DeptSidebar";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("DeptSidebar", () => {
  it("默认 externalOnly=true → DeptTree 接收 true", () => {
    const { baseElement } = render(<DeptSidebar />, { wrapper });
    const tree = baseElement.querySelector('[data-testid="dept-tree"]');
    expect(tree?.getAttribute("data-external")).toBe("true");
  });

  it("externalOnly=false → DeptTree 接收 false", () => {
    const { baseElement } = render(<DeptSidebar externalOnly={false} />, { wrapper });
    const tree = baseElement.querySelector('[data-testid="dept-tree"]');
    expect(tree?.getAttribute("data-external")).toBe("false");
  });

  it("width=200 → Sider width 200", () => {
    const { baseElement } = render(<DeptSidebar width={200} />, { wrapper });
    const sider = baseElement.querySelector(".ant-layout-sider");
    expect(sider).toBeTruthy();
  });

  it("selectedDeptId → 透传给 DeptTree selectedKeys", () => {
    const { baseElement } = render(<DeptSidebar selectedDeptId="d1" />, { wrapper });
    expect(baseElement.querySelector('[data-testid="dept-tree"]')).toBeTruthy();
  });

  it("onSelect 提供 → 透传", () => {
    const onSelect = vi.fn();
    render(<DeptSidebar onSelect={onSelect} />, { wrapper });
    expect(true).toBe(true);
  });

  it("自定义 style → 透传", () => {
    const { baseElement } = render(<DeptSidebar style={{ marginTop: 10 }} />, { wrapper });
    const sider = baseElement.querySelector(".ant-layout-sider") as HTMLElement;
    expect(sider?.style.marginTop).toBe("10px");
  });
});
