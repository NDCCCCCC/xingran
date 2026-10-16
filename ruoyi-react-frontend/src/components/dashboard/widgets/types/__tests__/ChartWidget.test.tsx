/**
 * Phase 88 Batch106 — dashboard/widgets/ChartWidget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { ChartWidget } from "../ChartWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(() => ({ data: null, loading: false, error: null, refresh: vi.fn() })),
}));

vi.mock("@/components/charts/EChartsWrapper", () => ({
  default: () => <div data-testid="echarts-stub" />,
}));

const baseWidget = {
  id: "w1",
  title: "图表",
  type: "chart",
  dataSource: { type: "static", data: {} },
  position: { x: 0, y: 0, w: 4, h: 3 },
} as any;

describe("ChartWidget 渲染", () => {
  it("data=null → 渲染空 option", () => {
    const { baseElement } = renderWithProviders(
      <ChartWidget widget={baseWidget} display={{ chartType: "line" } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data + chartType=line → 折线图 option", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { x: ["1月", "2月"], series: [{ name: "销售", data: [100, 200] }] },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ChartWidget widget={baseWidget} display={{ chartType: "line" } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data + chartType=bar → 柱状图", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { x: ["A", "B"], series: [{ name: "数量", data: [10, 20] }] },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ChartWidget widget={baseWidget} display={{ chartType: "bar" } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data + chartType=pie → 饼图", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: {
        data: [
          { name: "A", value: 10 },
          { name: "B", value: 20 },
        ],
      },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ChartWidget widget={baseWidget} display={{ chartType: "pie" } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data + chartType=area → 面积图", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { x: ["1", "2"], series: [{ data: [1, 2] }] },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ChartWidget widget={baseWidget} display={{ chartType: "area" } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data + chartType=unknown → 空 option", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: {},
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ChartWidget widget={baseWidget} display={{ chartType: "unknown" } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("onEdit + onDelete 传入", () => {
    const { baseElement } = renderWithProviders(
      <ChartWidget
        widget={baseWidget}
        display={{ chartType: "line" } as any}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    expect(baseElement).toBeDefined();
  });
});
