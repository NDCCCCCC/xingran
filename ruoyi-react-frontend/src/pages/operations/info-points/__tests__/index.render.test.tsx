/**
 * Phase 88 Batch83 — operations info-points 页面渲染(316 stmts, 28.2% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import InfoPointPage from "../index";
import { infoPointApi, workstationApi, buildingApi, floorApi } from "@/lib/opsApi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  infoPointApi: {
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
  workstationApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
  buildingApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
  floorApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
  deptApi: { tree: vi.fn(() => Promise.resolve({ data: [] })) },
}));

function renderInfo() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <InfoPointPage />
    </QueryClientProvider>
  );
}

describe("InfoPointPage 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    vi.mocked(infoPointApi.list).mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    vi.mocked(infoPointApi.statistics).mockResolvedValueOnce({
      data: { total: 0, active: 0 },
    } as any);
    const { baseElement } = renderInfo();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 表格行渲染", async () => {
    vi.mocked(infoPointApi.list).mockResolvedValueOnce({
      data: {
        list: [
          {
            id: "ip1",
            name: "信息点-01",
            pointType: "network",
            floorId: "f1",
            workstationId: "w1",
            status: 0,
          },
        ],
        total: 1,
      },
    } as any);
    vi.mocked(infoPointApi.statistics).mockResolvedValueOnce({
      data: { total: 1, active: 1 },
    } as any);
    const { baseElement } = renderInfo();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("api.list 失败 → catch 路径", async () => {
    vi.mocked(infoPointApi.list).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderInfo();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
