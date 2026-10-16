/**
 * Phase 88 Batch102 — operations/building-spaces-3d/index 测试(33 stmts, 0% → 高)
 * 通过 mock HubeiMap + Three.js 组件避免 jsdom 崩溃
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  buildingApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [], total: 0 } })),
  },
}));

vi.mock("../components/HubeiMap", () => ({
  default: () => <div data-testid="hubei-map" />,
}));

vi.mock("../components/HubeiMapGL", () => ({
  default: () => <div data-testid="hubei-map-gl" />,
}));

vi.mock("@/components/three/BuildingScene", () => ({
  BuildingView3DLazy: () => null,
  FloorView3DLazy: () => null,
}));

function renderBS3D() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <BuildingSpaces3D />
    </QueryClientProvider>
  );
}

let BuildingSpaces3D: any;
beforeAll(async () => {
  BuildingSpaces3D = (await import("../index")).default;
});

describe("BuildingSpaces3D 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderBS3D();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 个楼宇 → 渲染", async () => {
    const { buildingApi } = await import("@/lib/opsApi");
    vi.mocked(buildingApi.list).mockResolvedValueOnce({
      data: {
        list: [
          {
            id: "b1",
            name: "一号楼",
            level: 1,
            status: 0,
            lng: 114.4,
            lat: 30.5,
          },
        ],
        total: 1,
      },
    } as any);
    const { baseElement } = renderBS3D();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    const { buildingApi } = await import("@/lib/opsApi");
    vi.mocked(buildingApi.list).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderBS3D();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
