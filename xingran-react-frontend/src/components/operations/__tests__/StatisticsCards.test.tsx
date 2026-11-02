/**
 * Phase 88 Batch346 — components/operations/StatisticsCards 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatisticsCards } from "../StatisticsCards";

describe("components/operations/StatisticsCards", () => {
  it("渲染所有 items 的 title + value", () => {
    render(
      <StatisticsCards
        items={[
          { title: "总楼宇", value: 12 },
          { title: "总楼层", value: 50 },
        ]}
      />
    );
    expect(screen.getByText("总楼宇")).toBeInTheDocument();
    expect(screen.getByText("总楼层")).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument();
    expect(screen.getByText("50")).toBeInTheDocument();
  });

  it("show=false → 不渲染", () => {
    const { container } = render(
      <StatisticsCards items={[{ title: "X", value: 1 }]} show={false} />
    );
    expect(container.firstChild).toBeNull();
  });

  it("默认 columns = items.length", () => {
    render(<StatisticsCards items={[{ title: "A", value: 1 }]} />);
    // Single column - span=24
    expect(screen.getByText("A")).toBeInTheDocument();
  });

  it("自定义 columns=4", () => {
    render(
      <StatisticsCards
        items={[
          { title: "A", value: 1 },
          { title: "B", value: 2 },
          { title: "C", value: 3 },
          { title: "D", value: 4 },
        ]}
        columns={4}
      />
    );
    expect(screen.getByText("A")).toBeInTheDocument();
    expect(screen.getByText("D")).toBeInTheDocument();
  });

  it("valueStyle deprecated path", () => {
    render(<StatisticsCards items={[{ title: "X", value: 42, valueStyle: { color: "red" } }]} />);
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("styles.content 新 path", () => {
    render(
      <StatisticsCards
        items={[{ title: "Y", value: 99, styles: { content: { color: "blue" } } }]}
      />
    );
    expect(screen.getByText("99")).toBeInTheDocument();
  });

  it("空 items → 只渲染 Row", () => {
    const { container } = render(<StatisticsCards items={[]} />);
    expect(container.querySelector(".ant-row")).toBeTruthy();
  });
});
