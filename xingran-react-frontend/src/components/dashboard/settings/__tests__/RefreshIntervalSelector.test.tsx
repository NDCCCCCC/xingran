/**
 * Phase 88 Batch134 — components/dashboard/settings/RefreshIntervalSelector 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import RefreshIntervalSelector from "../RefreshIntervalSelector";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("RefreshIntervalSelector", () => {
  it("value=300 → 渲染 5 分钟", () => {
    const { baseElement } = render(<RefreshIntervalSelector value={300} />, { wrapper });
    expect(baseElement.textContent).toContain("5 分钟");
  });

  it("value=undefined → 渲染默认（不抛错）", () => {
    const { baseElement } = render(<RefreshIntervalSelector />, { wrapper });
    expect(baseElement).toBeDefined();
  });

  it("value=7200 (自定义 2小时) → 渲染自定义输入", () => {
    const { baseElement } = render(<RefreshIntervalSelector value={7200} />, { wrapper });
    // isCustom → should show InputNumber + unit Select
    expect(baseElement.querySelector(".ant-input-number-input")).toBeTruthy();
  });

  it("value=120 (自定义 2 分钟) → 转换 number=2 unit=分钟", () => {
    const { baseElement } = render(<RefreshIntervalSelector value={120} />, { wrapper });
    const input = baseElement.querySelector(".ant-input-number-input") as HTMLInputElement;
    expect(input?.value).toBe("2");
  });

  it("value=45 (自定义 45 秒) → 转换 number=45 unit=秒", () => {
    const { baseElement } = render(<RefreshIntervalSelector value={45} />, { wrapper });
    const input = baseElement.querySelector(".ant-input-number-input") as HTMLInputElement;
    expect(input?.value).toBe("45");
  });

  it("disabled=true → 禁用", () => {
    const { baseElement } = render(<RefreshIntervalSelector value={300} disabled />, { wrapper });
    expect(baseElement.querySelector(".ant-select-disabled")).toBeTruthy();
  });

  it("onChange 回调 + value 变化触发 useEffect", () => {
    const onChange = vi.fn();
    const { rerender, baseElement } = render(
      <RefreshIntervalSelector value={300} onChange={onChange} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("5 分钟");
    rerender(<RefreshIntervalSelector value={60} onChange={onChange} />);
    expect(baseElement.textContent).toContain("1 分钟");
  });
});
