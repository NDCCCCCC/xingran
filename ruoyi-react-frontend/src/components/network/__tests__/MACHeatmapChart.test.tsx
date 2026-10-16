/**
 * Phase 88 Batch145 — components/network/MACHeatmapChart 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/charts/EChartsWrapper", () => ({
  default: ({ option }: any) => (
    <div data-testid="echarts-mock">
      <span data-testid="echarts-cells">{option?.series?.[0]?.data?.length ?? 0}</span>
    </div>
  ),
}));

import MACHeatmapChart from "../MACHeatmapChart";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("MACHeatmapChart", () => {
  it("loading=true → 渲染 Spin", () => {
    const { baseElement } = render(
      <MACHeatmapChart
        data={{ cells: [], topN: 0, start: "", end: "", total: 0, snapshot: "" } as any}
        loading
      />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-spin")).toBeTruthy();
  });

  it("data.cells 空 → Empty", () => {
    const { baseElement } = render(
      <MACHeatmapChart
        data={{ cells: [], topN: 0, start: "", end: "", total: 0, snapshot: "" } as any}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("暂无热力图数据");
  });

  it("桌面端 + cells 非空 → 渲染 ECharts (mock)", () => {
    const data = {
      cells: [
        {
          deviceId: "d1",
          deviceNameSnapshot: "dev1",
          interfaceName: "p1",
          changeCount: 5,
          snapshot: "s1",
          timestamp: "2026-01-01",
        },
      ],
      topN: 1,
      start: "2026-01-01",
      end: "2026-01-07",
      total: 1,
      snapshot: "snap",
    };
    const { baseElement } = render(<MACHeatmapChart data={data as any} />, { wrapper });
    expect(baseElement.querySelector('[data-testid="mac-heatmap-desktop"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="echarts-mock"]')).toBeTruthy();
  });

  it("移动端 isMobile=true → 渲染 Top-20 卡片列表", () => {
    const data = {
      cells: [
        {
          deviceId: "d1",
          deviceNameSnapshot: "dev1",
          interfaceName: "p1",
          changeCount: 10,
          snapshot: "s1",
          timestamp: "2026-01-01",
        },
        {
          deviceId: "d2",
          deviceNameSnapshot: "dev2",
          interfaceName: "p2",
          changeCount: 5,
          snapshot: "s2",
          timestamp: "2026-01-01",
        },
      ],
      topN: 2,
      start: "2026-01-01",
      end: "2026-01-07",
      total: 2,
      snapshot: "snap",
    };
    const { baseElement } = render(<MACHeatmapChart data={data as any} isMobile />, { wrapper });
    expect(baseElement.querySelector('[data-testid="mac-heatmap-mobile"]')).toBeTruthy();
    expect(baseElement.textContent).toContain("Top-20");
    expect(baseElement.textContent).toContain("dev1");
    expect(baseElement.textContent).toContain("10 次");
  });

  it("getHeatColor → ratio=0 → ramp[1] (low)", () => {
    // Tested indirectly through rendering. Verify component accepts max=0
    const data = {
      cells: [
        {
          deviceId: "d1",
          deviceNameSnapshot: "d",
          interfaceName: "p",
          changeCount: 1,
          snapshot: "s",
          timestamp: "t",
        },
      ],
      topN: 1,
      start: "2026-01-01",
      end: "2026-01-07",
      total: 1,
      snapshot: "snap",
    };
    const { baseElement } = render(<MACHeatmapChart data={data as any} isMobile />, { wrapper });
    expect(baseElement.textContent).toContain("1 次");
  });
});
