/**
 * Phase 88 Batch58 — operations/workstations 子组件 + workorder 剩余页面
 *
 * workstations index 死锁,转测子组件:LocationAliasDrawer + 视图 + modals
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import LocationAliasDrawer from "../operations/workstations/LocationAliasDrawer";
import WorkorderStatistics from "../workorder/statistics";
import WorkorderCategories from "../workorder/categories";

async function renderAndAssert(
  page: React.ReactElement,
  endpoints: Record<string, unknown> = {}
): Promise<void> {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(
    () => {
      expect(rendered.container.firstChild).not.toBeNull();
    },
    { timeout: 10000 }
  );
  await new Promise((r) => setTimeout(r, 300));
}

describe("batch58 workstations 子组件 + workorder 剩余", () => {
  it("LocationAliasDrawer open=true 渲染", async () => {
    await renderAndAssert(<LocationAliasDrawer open onClose={vi.fn()} />);
  }, 15000);

  it("LocationAliasDrawer open=true 渲染(再次验证)", async () => {
    await renderAndAssert(<LocationAliasDrawer open onClose={vi.fn()} />);
  }, 15000);

  it("WorkorderStatistics 渲染", async () => {
    await renderAndAssert(<WorkorderStatistics />, {
      "/workorder/statistics/list": { data: {} },
    });
  }, 15000);

  it("WorkorderCategories 渲染", async () => {
    await renderAndAssert(<WorkorderCategories />, {
      "/workorder/categories/list": { data: { list: [], total: 0 } },
    });
  }, 15000);
});
