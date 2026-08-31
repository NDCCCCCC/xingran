/**
 * Phase 88 Batch230 — components/operations/StatisticsCards 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
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

describe("operations/StatisticsCards", () => {
  it("渲染 items", () => {
    const items = [
      { title: "Total", value: 100 },
      { title: "Active", value: 80 },
    ];
    render(<StatisticsCards items={items} />, { wrapper });
    expect(screen.getByText("Total")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("100")).toBeInTheDocument();
    expect(screen.getByText("80")).toBeInTheDocument();
  });

  it("show=false → 不渲染", () => {
    const items = [{ title: "X", value: 1 }];
    render(<StatisticsCards items={items} show={false} />, { wrapper });
    expect(screen.queryByText("X")).toBeNull();
  });

  it("columns 自定义", () => {
    const items = [
      { title: "A", value: 1 },
      { title: "B", value: 2 },
      { title: "C", value: 3 },
    ];
    const { container } = render(<StatisticsCards items={items} columns={2} />, { wrapper });
    expect(container.querySelectorAll(".ant-col").length).toBeGreaterThanOrEqual(2);
  });

  it("items.length = columns 默认", () => {
    const items = [
      { title: "A", value: 1 },
      { title: "B", value: 2 },
    ];
    const { container } = render(<StatisticsCards items={items} />, { wrapper });
    expect(container.querySelectorAll(".ant-col").length).toBe(2);
  });

  it("空 items", () => {
    const { container } = render(<StatisticsCards items={[]} />, { wrapper });
    expect(container.querySelectorAll(".ant-col").length).toBe(0);
  });
});
