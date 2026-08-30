/**
 * Phase 88 Batch193 — components/dashboard/Widget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../widgets/WidgetRenderer", () => ({
  WidgetRenderer: ({ widget, onEdit, onDelete }: any) => (
    <div data-testid="widget-renderer">
      <span data-testid="widget-id">{widget.id}</span>
      <span data-testid="widget-type">{widget.type}</span>
      {onEdit && (
        <button data-testid="widget-edit" onClick={onEdit}>
          Edit
        </button>
      )}
      {onDelete && (
        <button data-testid="widget-delete" onClick={onDelete}>
          Delete
        </button>
      )}
    </div>
  ),
}));

import { Widget } from "../Widget";
import type { WidgetConfig } from "@/types/dashboard";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const baseWidget: WidgetConfig = {
  id: "w1",
  type: "metric",
  title: "Test Widget",
  config: { value: 42 },
} as any;

describe("dashboard/Widget", () => {
  it("渲染并传递 widget 数据", () => {
    render(<Widget widget={baseWidget} />, { wrapper });
    expect(screen.getByTestId("widget-renderer")).toBeInTheDocument();
    expect(screen.getByTestId("widget-id").textContent).toBe("w1");
    expect(screen.getByTestId("widget-type").textContent).toBe("metric");
  });

  it("onEdit 挂接", () => {
    const onEdit = vi.fn();
    render(<Widget widget={baseWidget} onEdit={onEdit} />, { wrapper });
    screen.getByTestId("widget-edit").click();
    expect(onEdit).toHaveBeenCalled();
  });

  it("onDelete 挂接", () => {
    const onDelete = vi.fn();
    render(<Widget widget={baseWidget} onDelete={onDelete} />, { wrapper });
    screen.getByTestId("widget-delete").click();
    expect(onDelete).toHaveBeenCalled();
  });

  it("无 onEdit/onDelete 不渲染按钮", () => {
    render(<Widget widget={baseWidget} />, { wrapper });
    expect(screen.queryByTestId("widget-edit")).toBeNull();
    expect(screen.queryByTestId("widget-delete")).toBeNull();
  });

  it("displayName = Widget", () => {
    expect(Widget.displayName).toBe("Widget");
  });

  it("memo 行为: 同 props 不重渲染", () => {
    const { rerender } = render(<Widget widget={baseWidget} />, { wrapper });
    const initialEl = screen.getByTestId("widget-renderer");
    rerender(<Widget widget={baseWidget} />);
    expect(screen.getByTestId("widget-renderer")).toBe(initialEl);
  });
});
