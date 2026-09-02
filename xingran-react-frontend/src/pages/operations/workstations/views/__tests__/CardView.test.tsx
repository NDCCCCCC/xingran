/**
 * Phase 88 Batch412 — pages/operations/workstations/views/CardView 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { WorkstationCardView } from "../CardView";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("WorkstationCardView", () => {
  it("空数据渲染", () => {
    const { container } = render(
      <WorkstationCardView workstations={[]} onEdit={vi.fn()} onDelete={vi.fn()} />,
      { wrapper }
    );
    expect(container.textContent).toContain("暂无数据");
  });

  it("有数据时不抛错", () => {
    expect(() =>
      render(
        <WorkstationCardView
          workstations={[
            {
              id: "1",
              workstationCode: "WS001",
              workstationName: "工位1",
              workstationType: 1,
              status: 0,
              orgId: "org1",
              createTime: "",
            } as any,
          ]}
          onEdit={vi.fn()}
          onDelete={vi.fn()}
        />,
        { wrapper }
      )
    ).not.toThrow();
  });
});