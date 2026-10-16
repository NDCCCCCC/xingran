/**
 * Phase 88 Batch253 — pages/operations/floors/components/FloorStatisticsCards 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { FloorStatisticsCards } from "../FloorStatisticsCards";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("operations/floors/FloorStatisticsCards", () => {
  it("show=false → 不渲染", () => {
    const { container } = render(
      <FloorStatisticsCards statistics={{ total: 0, active: 0, inactive: 0 }} show={false} />,
      { wrapper }
    );
    expect(screen.queryByText("总楼层数")).toBeNull();
  });

  it("show=true → 渲染 3 卡片", () => {
    render(
      <FloorStatisticsCards statistics={{ total: 10, active: 8, inactive: 2 }} show={true} />,
      { wrapper }
    );
    expect(screen.getByText("总楼层数")).toBeInTheDocument();
    expect(screen.getByText("正常楼层")).toBeInTheDocument();
    expect(screen.getByText("停用楼层")).toBeInTheDocument();
  });

  it("显示 statistics 数值", () => {
    render(
      <FloorStatisticsCards statistics={{ total: 50, active: 40, inactive: 10 }} show={true} />,
      { wrapper }
    );
    expect(screen.getByText("50")).toBeInTheDocument();
    expect(screen.getByText("40")).toBeInTheDocument();
    expect(screen.getByText("10")).toBeInTheDocument();
  });

  it("statistics 0 全部为 0", () => {
    render(<FloorStatisticsCards statistics={{ total: 0, active: 0, inactive: 0 }} show={true} />, {
      wrapper,
    });
    const zeroes = screen.getAllByText("0");
    expect(zeroes.length).toBeGreaterThanOrEqual(3);
  });
});
