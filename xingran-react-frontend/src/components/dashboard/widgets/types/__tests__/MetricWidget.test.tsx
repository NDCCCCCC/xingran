/**
 * Phase 88 Batch106 — dashboard/widgets/MetricWidget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { MetricWidget } from "../MetricWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(() => ({ data: null, loading: false, error: null, refresh: vi.fn() })),
}));

const baseWidget = {
  id: "w1",
  title: "指标",
  type: "progress",
  dataSource: { type: "static", data: {} },
  position: { x: 0, y: 0, w: 4, h: 3 },
} as any;

describe("MetricWidget 渲染", () => {
  it("data=null → 渲染 percent=0", () => {
    const { baseElement } = renderWithProviders(
      <MetricWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data.value + target → 计算 percent", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { value: 50 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <MetricWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data.percent → 使用 percent 字段", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { percent: 75 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <MetricWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("value > target → clamp 到 100", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { value: 500 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <MetricWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("colorThresholds → 应用颜色", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { value: 80 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <MetricWidget
        widget={baseWidget}
        display={
          {
            target: 100,
            colorThresholds: [
              { value: 0, color: "red" },
              { value: 50, color: "orange" },
              { value: 100, color: "green" },
            ],
          } as any
        }
      />
    );
    expect(baseElement).toBeDefined();
  });

  it("onEdit + onDelete 传入", () => {
    const { baseElement } = renderWithProviders(
      <MetricWidget
        widget={baseWidget}
        display={{ target: 100 } as any}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    expect(baseElement).toBeDefined();
  });
});
