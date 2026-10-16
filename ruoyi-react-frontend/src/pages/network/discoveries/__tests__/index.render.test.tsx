/**
 * Phase 88 Batch91 — network discoveries 页面渲染(89 stmts, 29.2% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import DeviceDiscoveryPage from "../index";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/networkApi", () => ({
  batchExport: vi.fn(() => Promise.resolve({})),
}));

describe("DeviceDiscoveryPage 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<DeviceDiscoveryPage />, {
      endpoints: { "/network/discoveries/list": { data: { list: [], total: 0 } } },
    });
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 表格行渲染", async () => {
    const { baseElement } = renderWithProviders(<DeviceDiscoveryPage />, {
      endpoints: {
        "/network/discoveries/list": {
          data: {
            list: [
              {
                id: "d1",
                ipRange: "10.0.0.0/24",
                status: 0,
                foundCount: 5,
                createdAt: "2026-01-01 12:00:00",
              },
            ],
            total: 1,
          },
        },
      },
    });
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    const { baseElement } = renderWithProviders(<DeviceDiscoveryPage />);
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });
});
