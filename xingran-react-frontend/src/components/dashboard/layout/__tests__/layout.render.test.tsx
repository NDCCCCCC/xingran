/**
 * Phase 88 batch6 — dashboard layout 子组件渲染(正确 props)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { DashboardGrid } from "../DashboardGrid";
import { GridItem } from "../GridItem";
import { LayoutToolbar } from "../LayoutToolbar";
import { TemplateSelector } from "../TemplateSelector";

const widget: any = {
  id: "w1",
  type: "stat-card",
  title: "测试 Widget",
  position: { x: 0, y: 0, w: 6, h: 3 },
  dataSource: { type: "static", data: null },
  display: { type: "stat-card" },
};

async function expectRenders(ui: React.ReactElement) {
  const { rendered } = renderPageWithEndpoints(ui, {});
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("dashboard layout 组件渲染", () => {
  it("DashboardGrid(空 widgets)", async () => {
    await expectRenders(
      <DashboardGrid widgets={[]} onLayoutChange={vi.fn()}>
        <div>empty</div>
      </DashboardGrid>
    );
  });
  it("DashboardGrid(1 widget)", async () => {
    await expectRenders(
      <DashboardGrid widgets={[widget]} onLayoutChange={vi.fn()}>
        <div data-testid="grid-child">child</div>
      </DashboardGrid>
    );
  });
  it("GridItem", async () => {
    await expectRenders(
      <GridItem widget={widget}>
        <div>item-content</div>
      </GridItem>
    );
  });
  it("LayoutToolbar", async () => {
    await expectRenders(<LayoutToolbar />);
  });
  it("TemplateSelector(closed)", async () => {
    await expectRenders(<TemplateSelector visible={false} onClose={vi.fn()} onSelect={vi.fn()} />);
  });
});
