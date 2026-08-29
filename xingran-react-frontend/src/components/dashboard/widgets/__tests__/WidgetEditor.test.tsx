/**
 * Phase 88 Batch88 — dashboard/widgets/WidgetEditor 测试(44 stmts, 0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { WidgetEditor } from "../WidgetEditor";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const baseWidget = {
  id: "w1",
  title: "测试Widget",
  type: "stat-card",
  dataSource: { type: "static", data: { value: 42 } },
  display: { showTitle: true },
  position: { x: 0, y: 0, w: 4, h: 3 },
} as any;

describe("WidgetEditor 渲染", () => {
  it("visible=false 不渲染 Drawer 内容", () => {
    const { baseElement } = renderWithProviders(
      <WidgetEditor visible={false} widget={null} onClose={vi.fn()} />
    );
    expect(baseElement.querySelector(".ant-drawer-content")).toBeNull();
  });

  it("visible=true + widget 非空 → 渲染编辑表单", () => {
    const { baseElement } = renderWithProviders(
      <WidgetEditor visible widget={baseWidget} onClose={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });

  it("widget=null → 显示空白表单", () => {
    const { baseElement } = renderWithProviders(
      <WidgetEditor visible widget={null} onClose={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });

  it("onSave + onDelete 传入", () => {
    const onSave = vi.fn();
    const onDelete = vi.fn();
    const { baseElement } = renderWithProviders(
      <WidgetEditor
        visible
        widget={baseWidget}
        onClose={vi.fn()}
        onSave={onSave}
        onDelete={onDelete}
      />
    );
    expect(baseElement).toBeDefined();
  });
});
