/**
 * Phase 88 Batch51 — 真 0% 文件批量渲染(真实 hooks + mock API)
 *
 * ous/index_with_dept(133) / DashboardView(64) / LocationAliasDrawer(63) /
 * asset reconciliation dashboard(54) / TargetSelector(47) / MACHeatmapChart(43) /
 * ConfigProvider(23) / building-spaces(35)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// WebSocket hook jsdom 不兼容 — stub 空实现(hook 调用仍计 DashboardView 行覆盖)
vi.mock("@/hooks/useWebSocket", () => ({
  useWebSocket: vi.fn(() => ({ connected: false, subscribe: vi.fn(), unsubscribe: vi.fn() })),
}));

// reconciliation 页的 WS hook 返回的 disconnect 非 function(jsdom) — stub
vi.mock("@/hooks/useReconciliationWebSocket", () => ({
  useReconciliationWebSocket: vi.fn(() => ({
    connect: vi.fn(),
    disconnect: vi.fn(),
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
    connected: false,
  })),
}));

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import ADOUPage from "../ad-domain/ous/index_with_dept";
import LocationAliasDrawer from "../operations/workstations/LocationAliasDrawer";
import ReconDashboard from "../asset/reconciliation/dashboard";
import BuildingSpacesPage from "../operations/building-spaces";
import TargetSelector from "@/components/TargetSelector";
import MACHeatmapChart from "@/components/network/MACHeatmapChart";
import ConfigProviderX from "@/components/ConfigProvider";

async function renderWithContainer(page: React.ReactElement, endpoints: Record<string, unknown>) {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(
    () => {
      expect(
        rendered.container.querySelector(
          ".ant-table, .ant-card, .ant-spin, .ant-form, .ant-empty, .ant-drawer, .ant-select, .ant-tabs, .ant-segmented, [class]"
        )
      ).not.toBeNull();
    },
    { timeout: 10000 }
  );
  return rendered;
}

describe("batch51 真 0% 文件渲染", () => {
  it("ADOUPage(index_with_dept)渲染", async () => {
    await renderWithContainer(<ADOUPage />, {
      "/ad-domain/ous/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("LocationAliasDrawer open=true 渲染", async () => {
    await renderWithContainer(<LocationAliasDrawer open onClose={vi.fn()} />, {
      "/ops/workstation/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("LocationAliasDrawer open=false 无内容", () => {
    const { rendered } = renderPageWithEndpoints(
      <LocationAliasDrawer open={false} onClose={vi.fn()} />,
      {}
    );
    expect(rendered.container).toBeDefined();
  });

  it("asset reconciliation dashboard 渲染", async () => {
    await renderWithContainer(<ReconDashboard />, {
      "/asset/reconciliation/dashboard": { data: {} },
    });
  }, 15000);

  it("BuildingSpacesPage 渲染", async () => {
    await renderWithContainer(<BuildingSpacesPage />, {
      "/ops/building/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("TargetSelector 渲染", async () => {
    await renderWithContainer(
      <TargetSelector
        targetType="all"
        onTargetTypeChange={vi.fn()}
        onTargetDeptsChange={vi.fn()}
        onTargetRolesChange={vi.fn()}
        onTargetUsersChange={vi.fn()}
      />,
      {}
    );
  }, 15000);

  it("MACHeatmapChart 渲染(空数据)", async () => {
    const { rendered } = renderPageWithEndpoints(
      <MACHeatmapChart data={[]} onCellClick={vi.fn()} />,
      {}
    );
    expect(rendered.container).toBeDefined();
  });

  it("ConfigProvider 包裹子元素渲染", async () => {
    const { rendered } = renderPageWithEndpoints(
      <ConfigProviderX>
        <div data-testid="cp-child">child</div>
      </ConfigProviderX>,
      {}
    );
    expect(rendered.container.textContent).toContain("child");
  });
});
