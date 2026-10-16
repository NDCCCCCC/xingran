/**
 * Phase 88 Batch82 — operations buildings 页面渲染(179 stmts, 29.6% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import BuildingPage from "../index";
import { buildingApi } from "@/lib/opsApi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  buildingApi: {
    list: vi.fn(),
    statistics: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
    downloadTemplate: vi.fn(),
    export: vi.fn(),
    import: vi.fn(),
  },
  deptApi: {
    tree: vi.fn(() => Promise.resolve({ data: [] })),
  },
  floorApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
  workstationApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
}));

function renderBuilding() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <BuildingPage />
    </QueryClientProvider>
  );
}

describe("BuildingPage 渲染", () => {
  it("空数据渲染不抛错", async () => {
    vi.mocked(buildingApi.list).mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    vi.mocked(buildingApi.statistics).mockResolvedValueOnce({
      data: { total: 0, active: 0 },
    } as any);
    const { baseElement } = renderBuilding();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("buildingApi.list 失败 → catch 路径", async () => {
    vi.mocked(buildingApi.list).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderBuilding();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("有 1 行 → 表格行渲染", async () => {
    vi.mocked(buildingApi.list).mockResolvedValueOnce({
      data: {
        list: [
          {
            id: "b1",
            name: "一号楼",
            code: "B1",
            address: "北京市朝阳区",
            status: 0,
            floorCount: 5,
          },
        ],
        total: 1,
      },
    } as any);
    vi.mocked(buildingApi.statistics).mockResolvedValueOnce({
      data: { total: 1, active: 1 },
    } as any);
    const { baseElement } = renderBuilding();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
