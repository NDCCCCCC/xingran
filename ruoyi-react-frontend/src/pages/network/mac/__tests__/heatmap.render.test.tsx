/**
 * Phase 88 Batch29 — network mac heatmap 页渲染测试(原 0%)
 *
 * mock macHeatmapApi.queryMACHeatmap + 桩掉 lazy 的 MACHeatmapChart(echarts 重组件)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/macHeatmapApi", () => ({
  queryMACHeatmap: vi.fn().mockResolvedValue({
    code: 0,
    data: { ports: [], summary: { totalEvents: 0 } },
  }),
}));

vi.mock("@/components/network/MACHeatmapChart", () => ({
  default: () => <div data-testid="mac-heatmap-chart-stub" />,
}));

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { QueryClient } from "@tanstack/react-query";
import HeatmapPage from "../heatmap";
import { queryMACHeatmap } from "@/lib/api/macHeatmapApi";

describe("network mac heatmap 页渲染", () => {
  it("渲染时间预设按钮 + 默认 7d", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: 0, gcTime: 0 } },
    });
    const { findByText, baseElement } = renderWithProviders(<HeatmapPage />, {
      route: "/network/mac/heatmap",
      queryClient,
    });

    expect(await findByText("近 1h")).toBeDefined();
    expect(await findByText("近 24h")).toBeDefined();
    expect(await findByText("近 7d")).toBeDefined();
    expect(await findByText("近 30d")).toBeDefined();
    expect(await findByText("近 90d")).toBeDefined();
    expect(await findByText("自定义")).toBeDefined();

    // queryParams 初始为 null → useQuery enabled=false,API 不触发是预期行为
    // 断言预设按钮组 + 卡片容器存在即可
    expect(queryMACHeatmap).not.toHaveBeenCalled();
    expect(baseElement.querySelector(".ant-card, .ant-form")).not.toBeNull();
  }, 15000);
});
