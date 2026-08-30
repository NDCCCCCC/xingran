/**
 * Phase 88 Batch125 — components/operations/StatisticsCards 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { StatisticsCards } from "../StatisticsCards";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("StatisticsCards", () => {
  it("items=[{title:'A',value:1}] → 渲染卡片", () => {
    const { baseElement } = render(<StatisticsCards items={[{ title: "A", value: 1 }]} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("A");
    expect(baseElement.textContent).toContain("1");
  });

  it("show=false → 不渲染 Row", () => {
    const { baseElement } = render(
      <StatisticsCards items={[{ title: "A", value: 1 }]} show={false} />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-row")).toBeNull();
  });

  it("多 items + columns=2 → 渲染多列", () => {
    const { baseElement } = render(
      <StatisticsCards
        items={[
          { title: "A", value: 1 },
          { title: "B", value: 2 },
          { title: "C", value: 3 },
          { title: "D", value: 4 },
        ]}
        columns={2}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("A");
    expect(baseElement.textContent).toContain("B");
    expect(baseElement.textContent).toContain("C");
    expect(baseElement.textContent).toContain("D");
  });

  it("自定义 prefix + styles.content", () => {
    const { baseElement } = render(
      <StatisticsCards
        items={[
          {
            title: "X",
            value: 100,
            prefix: <span data-testid="prefix">★</span>,
            styles: { content: { color: "red" } },
          },
        ]}
      />,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="prefix"]')).toBeTruthy();
    expect(baseElement.textContent).toContain("X");
  });

  it("valueStyle (deprecated) → 仍生效", () => {
    const { baseElement } = render(
      <StatisticsCards items={[{ title: "Y", value: 50, valueStyle: { color: "blue" } }]} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("Y");
    expect(baseElement.textContent).toContain("50");
  });

  it("style 自定义样式透传", () => {
    const { container } = render(
      <StatisticsCards items={[{ title: "S", value: 0 }]} style={{ marginBottom: 32 }} />,
      { wrapper }
    );
    expect(container.firstChild).toBeTruthy();
  });

  it("空 items → 仍渲染 Row", () => {
    const { container } = render(<StatisticsCards items={[]} />, { wrapper });
    expect(container.firstChild).toBeTruthy();
  });
});
