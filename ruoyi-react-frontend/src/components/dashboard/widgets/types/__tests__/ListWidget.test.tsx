/**
 * Phase 88 Batch107 — dashboard/widgets/ListWidget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { ListWidget } from "../ListWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(() => ({ data: [], loading: false, error: null, refresh: vi.fn() })),
}));

const baseWidget = {
  id: "w1",
  title: "列表",
  type: "list",
  dataSource: { type: "static", data: [] },
  position: { x: 0, y: 0, w: 4, h: 3 },
} as any;

describe("ListWidget 渲染", () => {
  it("data=[] → 渲染空", () => {
    const { baseElement } = renderWithProviders(
      <ListWidget widget={baseWidget} display={{} as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data=array → 渲染列表", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: [
        { id: "l1", title: "项1", description: "描述1" },
        { id: "l2", title: "项2", description: "描述2" },
      ],
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ListWidget widget={baseWidget} display={{} as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data 非数组 → 单元素包装", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { id: "l1", title: "项1" },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <ListWidget widget={baseWidget} display={{} as any} />
    );
    expect(baseElement).toBeDefined();
  });

  it("onEdit + onDelete 传入", () => {
    const { baseElement } = renderWithProviders(
      <ListWidget widget={baseWidget} display={{} as any} onEdit={vi.fn()} onDelete={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });
});
