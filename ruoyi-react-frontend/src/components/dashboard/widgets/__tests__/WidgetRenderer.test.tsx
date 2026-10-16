/**
 * Phase 88 Batch340 — components/dashboard/widgets/WidgetRenderer 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../configs/widgetRegistry", () => ({
  widgetRegistry: {
    stat: {
      component: ({ widget }: any) => <div data-testid="stat-widget">{widget.title}</div>,
    },
    chart: {
      component: ({ widget }: any) => <div data-testid="chart-widget">{widget.title}</div>,
    },
  },
}));

import WidgetRenderer from "../WidgetRenderer";

describe("components/dashboard/widgets/WidgetRenderer", () => {
  it("已知 widget type → 渲染对应组件", () => {
    render(
      <WidgetRenderer widget={{ id: "w1", type: "stat", title: "统计", display: {} } as any} />
    );
    expect(screen.getByTestId("stat-widget")).toBeInTheDocument();
    expect(screen.getByText("统计")).toBeInTheDocument();
  });

  it("未知 widget type → 渲染占位 div", () => {
    render(
      <WidgetRenderer widget={{ id: "w1", type: "unknown-type", title: "X", display: {} } as any} />
    );
    expect(screen.getByText(/未知的Widget类型/)).toBeInTheDocument();
    expect(screen.getByText(/unknown-type/)).toBeInTheDocument();
  });

  it("chart widget → 渲染 chart 组件", () => {
    render(
      <WidgetRenderer widget={{ id: "w2", type: "chart", title: "图表", display: {} } as any} />
    );
    expect(screen.getByTestId("chart-widget")).toBeInTheDocument();
  });

  it("onEdit/onDelete 传递但不渲染", () => {
    const onEdit = vi.fn();
    const onDelete = vi.fn();
    render(
      <WidgetRenderer
        widget={{ id: "w3", type: "stat", title: "T", display: {} } as any}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    );
    expect(screen.getByTestId("stat-widget")).toBeInTheDocument();
  });
});
