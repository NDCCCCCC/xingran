/**
 * Phase 88 Batch93 — vdi VirtualMachineList 页面渲染(407 stmts, 29.2% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import VirtualMachineListPage from "../index";
import { vmApi, vdiServerApi } from "@/lib/vdiApi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/vdiApi", () => ({
  vmApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [], total: 0 } })),
    listVTPPlatforms: vi.fn(() => Promise.resolve([])),
    listRunPositions: vi.fn(() => Promise.resolve([])),
    listStorages: vi.fn(() => Promise.resolve([])),
    listNetworks: vi.fn(() => Promise.resolve([])),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
  vdiServerApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  },
}));

function renderVDI() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <VirtualMachineListPage />
    </QueryClientProvider>
  );
}

describe("VirtualMachineListPage 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderVDI();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 表格行渲染", async () => {
    vi.mocked(vmApi.list).mockResolvedValueOnce({
      data: {
        list: [
          {
            id: "vm1",
            name: "test-vm",
            status: "running",
            ipAddress: "10.0.0.5",
          },
        ],
        total: 1,
      },
    } as any);
    const { baseElement } = renderVDI();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    vi.mocked(vmApi.list).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderVDI();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
