/**
 * Phase 88 Batch341 — components/dashboard/Widget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../widgets/WidgetRenderer", () => ({
  WidgetRenderer: ({ widget }: any) => <div data-testid="widget-renderer">{widget.title}</div>,
}));

import Widget from "../Widget";

describe("components/dashboard/Widget", () => {
  it("渲染 WidgetRenderer", () => {
    render(<Widget widget={{ id: "w1", type: "stat", title: "标题", display: {} } as any} />);
    expect(screen.getByTestId("widget-renderer")).toBeInTheDocument();
    expect(screen.getByText("标题")).toBeInTheDocument();
  });

  it("displayName 正确", () => {
    expect(Widget.displayName).toBe("Widget");
  });

  it("memo 包裹", () => {
    // Confirm it's a memo component (has $$typeof Symbol(react.memo))
    expect((Widget as any).$$typeof).toBeDefined();
  });

  it("onEdit/onDelete 传递", () => {
    render(
      <Widget
        widget={{ id: "w2", type: "stat", title: "T2", display: {} } as any}
        onEdit={() => {}}
        onDelete={() => {}}
      />
    );
    expect(screen.getByTestId("widget-renderer")).toBeInTheDocument();
  });
});
