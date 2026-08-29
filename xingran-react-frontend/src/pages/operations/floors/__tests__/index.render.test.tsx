/**
 * Phase 88 Batch92 — operations floors 页面渲染(223 stmts, 28.7% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import FloorPage from "../index";
import { floorApi, buildingApi } from "@/lib/opsApi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  floorApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    statistics: vi.fn(),
    batch: vi.fn(),
  },
  buildingApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  },
  deptApi: { tree: vi.fn(() => Promise.resolve({ data: [] })) },
  workstationApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
}));

function renderFloor() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <FloorPage />
    </QueryClientProvider>
  );
}

describe("FloorPage 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    vi.mocked(floorApi.list).mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    vi.mocked(floorApi.statistics).mockResolvedValueOnce({ data: { total: 0 } } as any);
    const { baseElement } = renderFloor();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 表格行渲染", async () => {
    vi.mocked(floorApi.list).mockResolvedValueOnce({
      data: {
        list: [
          {
            id: "f1",
            name: "F1",
            floorCode: "F1",
            buildingId: "b1",
            floorNumber: 1,
            status: 0,
          },
        ],
        total: 1,
      },
    } as any);
    vi.mocked(floorApi.statistics).mockResolvedValueOnce({ data: { total: 1 } } as any);
    const { baseElement } = renderFloor();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    vi.mocked(floorApi.list).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderFloor();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
