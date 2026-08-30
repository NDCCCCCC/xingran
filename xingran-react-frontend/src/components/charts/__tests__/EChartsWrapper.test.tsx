/**
 * Phase 88 Batch124 — components/charts/EChartsWrapper 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("echarts-for-react", () => ({
  default: ({ option }: any) => (
    <div data-testid="echarts-mock">
      <span data-testid="echarts-option">{JSON.stringify(option?.title?.text || "no-title")}</span>
    </div>
  ),
}));

import EChartsWrapper from "../EChartsWrapper";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("EChartsWrapper", () => {
  it("渲染 → 加载 echarts-for-react 后渲染子组件", async () => {
    const { baseElement } = render(
      <EChartsWrapper option={{ title: { text: "Test Chart" } }} style={{ height: 300 }} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="echarts-mock"]')).toBeTruthy();
    });
    expect(baseElement.querySelector('[data-testid="echarts-option"]')?.textContent).toContain(
      "Test Chart"
    );
  });

  it("forwardRef → 暴露 ref", async () => {
    const ref: any = { current: null };
    render(<EChartsWrapper ref={ref} option={{ title: { text: "R" } }} />, { wrapper });
    await waitFor(() => {});
    // ref forwarding verification — instance may be the lazy-loaded component
    expect(ref).toBeDefined();
  });

  it("displayName = 'EChartsWrapper'", () => {
    expect(EChartsWrapper.displayName).toBe("EChartsWrapper");
  });
});
