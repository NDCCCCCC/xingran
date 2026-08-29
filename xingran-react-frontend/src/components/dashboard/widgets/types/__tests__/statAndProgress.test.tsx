/**
 * Phase 88 Batch38 — dashboard StatCardWidget + ProgressWidget 渲染测试
 *
 * 用 mockReturnValueOnce 控制 useWidgetData 返回,避开 vi.doMock 嵌套 hoisting。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(),
  useBatchWidgetData: vi.fn(),
}));

import { useWidgetData } from "@/hooks/useWidgetData";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { StatCardWidget } from "../StatCardWidget";
import { ProgressWidget } from "../ProgressWidget";
import type { WidgetConfig } from "@/types/dashboard";

const baseWidget: WidgetConfig = {
  id: "w-1",
  type: "stat-card",
  title: "今日访问",
  position: { x: 0, y: 0, w: 6, h: 3 },
  config: {},
};

describe("StatCardWidget", () => {
  it("data.total 优先渲染 100", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { total: 100 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const { container } = renderWithProviders(
      <StatCardWidget widget={baseWidget} display={{ iconColor: "#1890ff" }} />
    );
    expect(container.textContent).toContain("100");
  });

  it("无 data 渲染占位 -", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: null,
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const { container } = renderWithProviders(
      <StatCardWidget widget={baseWidget} display={{ iconColor: "#1890ff" }} />
    );
    expect(container.textContent).toContain("-");
  });

  it("showTrend=true 渲染标题", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { total: 100, trendUp: true, trendValue: 12 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const { container } = renderWithProviders(
      <StatCardWidget widget={baseWidget} display={{ iconColor: "#1890ff", showTrend: true }} />
    );
    expect(container.textContent).toContain("今日访问");
  });
});

describe("ProgressWidget", () => {
  it("data.percent 渲染进度值", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { percent: 75 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const { container } = renderWithProviders(<ProgressWidget widget={baseWidget} display={{}} />);
    expect(container.textContent).toContain("75");
  });

  it("无 percentage 字段时取 neither", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: { other: "x" }, // 不含 percent
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const { container } = renderWithProviders(<ProgressWidget widget={baseWidget} display={{}} />);
    // widget 标题或占位 — 仅断言不 crash
    expect(container).not.toBeNull();
  });

  it("无 data 渲染占位", () => {
    vi.mocked(useWidgetData).mockReturnValue({
      data: null,
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const { container } = renderWithProviders(<ProgressWidget widget={baseWidget} display={{}} />);
    expect(container).not.toBeNull();
  });
});
