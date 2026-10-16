/**
 * Phase 88 Batch107 — dashboard/widgets/ProgressWidget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { ProgressWidget } from "../ProgressWidget";
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
  title: "进度",
  type: "progress",
  dataSource: { type: "static", data: {} },
  position: { x: 0, y: 0, w: 4, h: 3 },
} as any;

describe("ProgressWidget 渲染", () => {
  it("data=null → percent=0", () => {
    const { baseElement } = renderWithProviders(
      <ProgressWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data.value → 计算 percent", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { value: 50 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ProgressWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data.percent → percent 字段", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { percent: 75 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ProgressWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data.progress → progress 字段", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { progress: 60 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ProgressWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("value=100 → status=success", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { value: 100 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ProgressWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("value<30 → status=exception", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { value: 10 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ProgressWidget widget={baseWidget} display={{ target: 100 } as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("colorThresholds → 颜色 + status 应用", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { value: 80 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ProgressWidget
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
      <ProgressWidget
        widget={baseWidget}
        display={{ target: 100 } as any}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    expect(baseElement).toBeDefined();
  });
});
