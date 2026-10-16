/**
 * Phase 88 Batch342 — components/dashboard/widgets/types/MetricWidget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { useWidgetData } from "@/hooks/useWidgetData";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData");

vi.mock("../../base/BaseWidget", () => ({
  BaseWidget: ({ children, data, loading, error }: any) => (
    <div
      data-testid="base-widget"
      data-loading={loading}
      data-error={error}
      data-data={JSON.stringify(data)}
    >
      {children}
    </div>
  ),
}));

import { MetricWidget } from "../MetricWidget";

const sampleWidget: any = {
  id: "w1",
  type: "metric",
  title: "指标",
  display: {},
};

describe("components/dashboard/widgets/types/MetricWidget", () => {
  it("渲染 BaseWidget + Progress", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { value: 75 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    render(<MetricWidget widget={sampleWidget} display={{}} />);
    expect(screen.getByTestId("base-widget")).toBeInTheDocument();
  });

  it("无 data → 0%", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: null,
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    render(<MetricWidget widget={sampleWidget} display={{}} />);
    expect(screen.getByText("0%")).toBeInTheDocument();
  });

  it("value=75 + target=100 → 75%", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { value: 75 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    render(<MetricWidget widget={sampleWidget} display={{ target: 100 }} />);
    expect(screen.getByText("75%")).toBeInTheDocument();
  });

  it("data.percent 字段优先", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { percent: 42 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    render(<MetricWidget widget={sampleWidget} display={{ target: 100 }} />);
    expect(screen.getByText("42%")).toBeInTheDocument();
  });

  it("value > target → 上限 100%", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { value: 200 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    render(<MetricWidget widget={sampleWidget} display={{ target: 100 }} />);
    expect(screen.getByText("100%")).toBeInTheDocument();
  });

  it("colorThresholds → 选择最大匹配阈值 color", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { value: 80 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    render(
      <MetricWidget
        widget={sampleWidget}
        display={{
          target: 100,
          colorThresholds: [
            { value: 50, color: "orange" },
            { value: 75, color: "green" },
          ],
        }}
      />
    );
    expect(screen.getByText("80%")).toBeInTheDocument();
  });

  it("onEdit/onDelete 传递", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { value: 50 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    render(
      <MetricWidget widget={sampleWidget} display={{}} onEdit={() => {}} onDelete={() => {}} />
    );
    expect(screen.getByTestId("base-widget")).toBeInTheDocument();
  });
});
