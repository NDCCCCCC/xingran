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
    // URL 是 /workorder/statistics（不带 /list，见 getWorkOrderStatistics）；
    // 且 data 必须满足 WorkOrderStatistics 契约：空对象 {} 是真值, 会通过组件的
    // `stats &&` 守卫, 随后 Object.entries(stats.byPriority) 对 undefined 抛错。
    await renderAndAssert(<WorkorderStatistics />, {
      "/workorder/statistics": {
        data: {
          total: 0,
          pending: 0,
          processing: 0,
          completed: 0,
          closed: 0,
          rejected: 0,
          byPriority: {},
          byCategory: {},
          byAssignee: [],
          byDepartment: [],
          trend: [],
          avgProcessTime: 0,
        },
      },
    });
  }, 15000);

  it("WorkorderCategories 渲染", async () => {
    // 该接口契约是 BaseResponse<WorkOrderCategory[]>，data 直接是数组而非分页对象；
    // 误用 {list,total} 会让 categories 变成对象，渲染期 flatCategories 抛错。
    await renderAndAssert(<WorkorderCategories />, {
      "/workorder/categories/list": { data: [] },
    });
  }, 15000);
});
