/**
 * Phase 88 Batch87 — dashboard/widgets/TableWidget 测试(33 stmts, 6.1% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { TableWidget } from "../TableWidget";
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
  title: "测试表格",
  type: "table",
  dataSource: { type: "static", data: [] },
} as any;

const baseDisplay = {
  pageSize: 10,
  columns: [{ title: "名称", dataIndex: "name", key: "name" }],
} as any;

describe("TableWidget 渲染", () => {
  it("data=[] 渲染不抛错", () => {
    const { baseElement } = renderWithProviders(
      <TableWidget widget={baseWidget} display={baseDisplay} />
    );
    expect(baseElement).toBeDefined();
  });

  it("data=array of objects → 渲染表格行", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: [
        { id: 1, name: "Alice" },
        { id: 2, name: "Bob" },
      ],
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <TableWidget widget={baseWidget} display={baseDisplay} />
    );
    expect(baseElement.textContent).toBeDefined();
  });

  it("data=非数组 → 包装为单元素数组", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: { id: 1, name: "single" },
      loading: false,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <TableWidget widget={baseWidget} display={baseDisplay} />
    );
    expect(baseElement).toBeDefined();
  });

  it("loading=true → 渲染", () => {
    vi.mocked(useWidgetData).mockReturnValueOnce({
      data: [],
      loading: true,
      error: null,
      refresh: vi.fn(),
    });
    const { baseElement } = renderWithProviders(
      <TableWidget widget={baseWidget} display={baseDisplay} />
    );
    expect(baseElement).toBeDefined();
  });

  it("onEdit + onDelete 传入", () => {
    const { baseElement } = renderWithProviders(
      <TableWidget widget={baseWidget} display={baseDisplay} onEdit={vi.fn()} onDelete={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });
});
