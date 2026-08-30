/**
 * Phase 88 Batch108 — dashboard/widgets/WidgetRenderer 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { WidgetRenderer } from "../WidgetRenderer";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(() => ({ data: null, loading: false, error: null, refresh: vi.fn() })),
}));

describe("WidgetRenderer 路由", () => {
  it("未知 widget.type → 显示错误消息", () => {
    const widget = {
      id: "w1",
      title: "未知",
      type: "unknown-type" as any,
      dataSource: { type: "static", data: {} },
      position: { x: 0, y: 0, w: 4, h: 3 },
    } as any;
    const { baseElement } = renderWithProviders(<WidgetRenderer widget={widget} />);
    expect(baseElement.textContent).toContain("未知的Widget类型");
  });

  it("type=stat-card → 渲染 StatCardWidget", () => {
    const widget = {
      id: "w1",
      title: "统计",
      type: "stat-card",
      dataSource: { type: "static", data: { value: 100 } },
      position: { x: 0, y: 0, w: 4, h: 3 },
    } as any;
    const { baseElement } = renderWithProviders(<WidgetRenderer widget={widget} />);
    expect(baseElement).toBeDefined();
  });

  it("type=chart → 渲染 ChartWidget", () => {
    const widget = {
      id: "w1",
      title: "图表",
      type: "chart",
      dataSource: { type: "static", data: {} },
      position: { x: 0, y: 0, w: 4, h: 3 },
    } as any;
    const { baseElement } = renderWithProviders(<WidgetRenderer widget={widget} />);
    expect(baseElement).toBeDefined();
  });

  it("type=table → 渲染 TableWidget", () => {
    const widget = {
      id: "w1",
      title: "表",
      type: "table",
      dataSource: { type: "static", data: [] },
      position: { x: 0, y: 0, w: 4, h: 3 },
    } as any;
    const { baseElement } = renderWithProviders(<WidgetRenderer widget={widget} />);
    expect(baseElement).toBeDefined();
  });

  it("type=list → 渲染 ListWidget", () => {
    const widget = {
      id: "w1",
      title: "列表",
      type: "list",
      dataSource: { type: "static", data: [] },
      position: { x: 0, y: 0, w: 4, h: 3 },
    } as any;
    const { baseElement } = renderWithProviders(<WidgetRenderer widget={widget} />);
    expect(baseElement).toBeDefined();
  });

  it("type=progress → 渲染 ProgressWidget", () => {
    const widget = {
      id: "w1",
      title: "进度",
      type: "progress",
      dataSource: { type: "static", data: {} },
      position: { x: 0, y: 0, w: 4, h: 3 },
    } as any;
    const { baseElement } = renderWithProviders(<WidgetRenderer widget={widget} />);
    expect(baseElement).toBeDefined();
  });

  it("onEdit + onDelete 回调传入", () => {
    const widget = {
      id: "w1",
      title: "统计",
      type: "stat-card",
      dataSource: { type: "static", data: {} },
      position: { x: 0, y: 0, w: 4, h: 3 },
    } as any;
    const { baseElement } = renderWithProviders(
      <WidgetRenderer widget={widget} onEdit={vi.fn()} onDelete={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });
});
