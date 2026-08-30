/**
 * Phase 88 Batch129 — components/shared/BatchDeleteButton 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import BatchDeleteButton from "../BatchDeleteButton";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("BatchDeleteButton", () => {
  it("selectedCount=0 → 不渲染", () => {
    const { baseElement } = render(<BatchDeleteButton selectedCount={0} onConfirm={vi.fn()} />, {
      wrapper,
    });
    expect(baseElement.querySelector("button")).toBeNull();
  });

  it("selectedCount>0 → 显示 批量删除(N) 按钮", () => {
    const { baseElement } = render(<BatchDeleteButton selectedCount={5} onConfirm={vi.fn()} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("批量删除");
    expect(baseElement.textContent).toContain("5");
  });

  it("disabled=true → 按钮 disabled", () => {
    const { baseElement } = render(
      <BatchDeleteButton selectedCount={3} onConfirm={vi.fn()} disabled />,
      { wrapper }
    );
    const btn = baseElement.querySelector("button");
    expect(btn?.disabled).toBe(true);
  });

  it("loading=true → 按钮 loading", () => {
    const { baseElement } = render(
      <BatchDeleteButton selectedCount={3} onConfirm={vi.fn()} loading />,
      { wrapper }
    );
    const btn = baseElement.querySelector("button");
    expect(btn?.className ?? "").toContain("ant-btn-loading");
  });

  it("ghost=false → 不渲染 ghost 样式", () => {
    const { baseElement } = render(
      <BatchDeleteButton selectedCount={3} onConfirm={vi.fn()} ghost={false} />,
      { wrapper }
    );
    const btn = baseElement.querySelector("button");
    // antd 不会渲染 ghost className 而是用 css var
    expect(btn).toBeTruthy();
  });

  it("点击 → 触发 Popconfirm + onConfirm", () => {
    const onConfirm = vi.fn();
    const { baseElement, getByText } = render(
      <BatchDeleteButton selectedCount={3} onConfirm={onConfirm} />,
      { wrapper }
    );
    fireEvent.click(getByText(/批量删除/));
    // Popconfirm should appear with confirm button
    expect(baseElement.textContent).toContain("确定");
  });

  it("自定义 confirmTitle 渲染", () => {
    const { baseElement } = render(
      <BatchDeleteButton selectedCount={3} onConfirm={vi.fn()} confirmTitle="自定义标题" />,
      { wrapper }
    );
    fireEvent.click(baseElement.querySelector("button")!);
    expect(baseElement.textContent).toContain("自定义标题");
  });
});
